package control

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/mlkem"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/hardpointlabs/agent/auth"
	"github.com/hardpointlabs/lpstream"
	"github.com/quic-go/quic-go"
	"golang.org/x/crypto/hkdf"
)

// we need some kind of state machine where we
// 1) say hello to the relay, state who we are and what services we have
// 2) block until we receive an OK from the relay
// 3) wait for relay to send connect requests
// 4) Probably crash obviously if the stream is closed unexpectedly

type Coordinator struct {
	connection    *quic.Conn
	controlStream *controlStream
}

type AuthState int

const (
	StateHello AuthState = iota
	StateWaitingPubKey
	StateWaitingApproval
	StateOK
	StateError
)

type controlStream struct {
	*lpstream.FrameCodec
	OrgId      string
	AuthState  AuthState
	QuicStream *quic.Stream
	keyPair    *auth.KeyPair
}

func (c *controlStream) Close() error {
	return c.QuicStream.Close()
}

var ErrHandshakeFailed = errors.New("Handshake failure")

func (c *controlStream) sendHello() error {
	if err := c.WriteFrame(c.helloMessage(c.OrgId, make(map[string]string))); err != nil {
		log.Println("Error writing HELLO message")
		c.AuthState = StateError
		return err
	}

	resp, err := c.ReadFrame()
	if err != nil {
		return err
	}
	if string(resp) == "OK" {
		log.Println("Agent key already approved")
		c.AuthState = StateOK
		return nil
	} else if string(resp) == "SENDPK" {
		log.Println("Relay is requesting public key")
		c.AuthState = StateWaitingPubKey
		return nil
	} else if string(resp) == "WAIT" {
		log.Println("Awaiting user approval")
		c.AuthState = StateWaitingApproval
		return nil
	} else {
		c.AuthState = StateError
		log.Printf("Unknown response from relay: '%s'", string(resp))
		return ErrHandshakeFailed
	}
}

func (c *controlStream) sendPubKey() error {
	log.Println("Sending pubkey")
	if err := c.WriteFrame(c.keyPair.Public); err != nil {
		log.Println("Error sending public key")
		c.AuthState = StateError
		return err
	}

	resp, err := c.ReadFrame()
	if err != nil {
		c.AuthState = StateError
		return err
	}

	if string(resp) == "OK" {
		c.AuthState = StateWaitingApproval
		return nil
	} else {
		c.AuthState = StateError
		log.Printf("Unknown response from relay: '%s'", string(resp))
		return ErrHandshakeFailed
	}
}

func (c *controlStream) waitApproval() error {
	log.Println("Waiting for approval")
	for {
		if err := c.WriteFrame([]byte("WAITPING")); err != nil {
			log.Println("Error sending ping")
			c.AuthState = StateError
			return err
		}

		resp, err := c.ReadFrame()
		if err != nil {
			c.AuthState = StateError
			return err
		}
		if string(resp) == "WAIT" {
			time.Sleep(time.Second * 10)
		} else if string(resp) == "OK" {
			c.AuthState = StateOK
			return nil
		} else {
			c.AuthState = StateError
			log.Printf("Unknown response from relay: '%s'", string(resp))
			return ErrHandshakeFailed
		}
	}
}

func pipe(reader, writer io.ReadWriteCloser, done chan<- struct{}) {
	defer func() {
		log.Printf("pipe done: reader=%T, writer=%T", reader, writer)
		done <- struct{}{}
	}()
	n, err := io.Copy(writer, reader)
	log.Printf("io.Copy transferred %d bytes, err=%v", n, err)
}

func (c *controlStream) doHandshake() error {
	var err error = nil
	for {
		switch c.AuthState {
		case StateHello:
			err = c.sendHello()
		case StateWaitingPubKey:
			err = c.sendPubKey()
		case StateWaitingApproval:
			err = c.waitApproval()
		case StateOK:
			return nil
		default:
			err = ErrHandshakeFailed
		}
		if err != nil {
			return err
		}
	}
}

func (c *controlStream) helloMessage(orgId string, additionalInfo map[string]string) []byte {
	fingerprint := c.keyPair.Fingerprint()
	timestamp := timeNowBytes()

	helloMessage := []byte("HELLO,")
	helloMessage = append(helloMessage, fingerprint...)
	helloMessage = append(helloMessage, ',')
	helloMessage = append(helloMessage, []byte(orgId)...)
	helloMessage = append(helloMessage, ',')
	helloMessage = append(helloMessage, timestamp...)
	helloMessage = append(helloMessage, ',')
	sig, err := c.keyPair.Sign(helloMessage)
	if err != nil {
		log.Panicf("Error signing HELLO message %v\n", err)
	}

	helloMessage = append(helloMessage, sig...)
	return helloMessage
}

type connectRequest struct {
	Dest             string
	EncapsulationKey []byte // Probably remove
}

func parseConnectMessage(message string) (connectRequest, error) {
	parts := strings.Split(message, ",")
	if len(parts) != 3 {
		return connectRequest{}, fmt.Errorf("invalid CONNECT message format: expected 3 parts, got %d", len(parts))
	}

	if parts[0] != "CONNECT" {
		return connectRequest{}, fmt.Errorf("invalid message type: expected CONNECT, got %s", parts[0])
	}

	dest := parts[1]

	return connectRequest{
		Dest: dest,
	}, nil
}

const (
	ivLength         = 12
	authTagLength    = 16
	derivedKeyLength = 32
)

type gcmCodec struct {
	frameDecoder *lpstream.Decoder
	frameEncoder *lpstream.Encoder
	gcm          cipher.AEAD
}

func newGcmCodec(sharedSecret []byte, codec *lpstream.FrameCodec) *gcmCodec {
	reader := hkdf.New(sha256.New, sharedSecret, nil, nil)
	derivedKey := make([]byte, derivedKeyLength)
	if _, err := io.ReadFull(reader, derivedKey); err != nil {
		panic(err)
	}
	block, _ := aes.NewCipher(derivedKey)
	gcm, _ := cipher.NewGCM(block)
	return &gcmCodec{frameDecoder: codec.Decoder, frameEncoder: codec.Encoder, gcm: gcm}
}

func (rw *gcmCodec) Read(p []byte) (n int, err error) {
	encryptedFrame, err := rw.frameDecoder.ReadFrame()
	if err != nil {
		log.Println("err reading frame for some reason")
		return 0, err
	}

	log.Println("got incoming encrypted data frame")

	iv := encryptedFrame[:ivLength]
	ciphertext := make([]byte, len(encryptedFrame)-ivLength)
	copy(ciphertext, encryptedFrame[ivLength:])
	authTag := encryptedFrame[len(encryptedFrame)-authTagLength:]

	log.Printf("frame len=%d", len(encryptedFrame))
	log.Printf("iv len=%d", len(iv))
	log.Printf("tag len=%d", len(authTag))

	log.Println("iv", fmt.Sprintf("%x", sha256.Sum256(iv)))
	log.Println("tag", fmt.Sprintf("%x", sha256.Sum256(authTag)))
	log.Println("ciphertext", fmt.Sprintf("%x", sha256.Sum256(ciphertext)))

	plaintext, err := rw.gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		log.Println("error decrypting")
		return 0, err
	}

	fmt.Println("Got some plaintext yo !")

	copied := copy(p, plaintext)
	log.Printf("Copied %d / %d bytes", copied, len(plaintext))
	return copied, nil
}

func (rw *gcmCodec) Write(p []byte) (n int, err error) {
	log.Println("writing response...")

	iv := make([]byte, ivLength)
	if _, err := rand.Read(iv); err != nil {
		return 0, err
	}

	ciphertext := rw.gcm.Seal(nil, iv, p, nil)

	var payload bytes.Buffer
	payload.Grow(ivLength + len(ciphertext))
	payload.Write(iv)
	payload.Write(ciphertext)

	authTag := ciphertext[len(ciphertext)-authTagLength:]
	log.Println("iv", fmt.Sprintf("%x", sha256.Sum256(iv)))
	log.Println("tag", fmt.Sprintf("%x", sha256.Sum256(authTag)))
	log.Println("ciphertext", fmt.Sprintf("%x", sha256.Sum256(ciphertext)))

	err = rw.frameEncoder.WriteFrame(payload.Bytes())
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (rw *gcmCodec) Close() error {
	log.Println("Closing now!")
	return nil
}

func CreateCoordinator(connection *quic.Conn, keyPair *auth.KeyPair, orgId string) (*Coordinator, error) {
	stream, err := connection.OpenStreamSync(context.Background())
	if err != nil {
		return nil, err
	}
	controlStream := &controlStream{
		QuicStream: stream,
		FrameCodec: lpstream.NewFrameCodec(stream),
		AuthState:  StateHello,
		keyPair:    keyPair,
		OrgId:      orgId,
	}
	return &Coordinator{connection: connection, controlStream: controlStream}, nil
}

func (c *Coordinator) Start() error {
	err := c.controlStream.doHandshake()
	if err != nil {
		return err
	}
	log.Println("Handshake succeeded")

	log.Println("Entering stream accept loop")
	return c.acceptLoop()
}

func (c *Coordinator) acceptLoop() error {
	for {
		stream, err := c.connection.AcceptStream(context.Background())
		if err != nil {
			if appErr, ok := errors.AsType[*quic.ApplicationError](err); ok {
				if appErr.ErrorCode == 0x00 {
					// It's *us* that's leaving, presumably due to receiving SIGINT
					return nil
				}
			} else {
				log.Printf("Error accepting stream: %v", err)
			}
			return err
		}
		go c.handleStream(stream)
	}
}

func (c *Coordinator) handleStream(stream *quic.Stream) {
	log.Printf("New stream %d", stream.StreamID())
	codec := lpstream.NewFrameCodec(stream)
	log.Printf("Waiting for frame on stream %d", stream.StreamID())
	msg, err := codec.ReadFrame()
	if err != nil {
		log.Printf("Error reading frame from stream %d: %v (type: %T)", stream.StreamID(), err, err)
		stream.Close()
		return
	}

	log.Printf("Received: %q", msg)

	connReq, err := parseConnectMessage(string(msg))
	if err != nil {
		log.Printf("Failed to parse CONNECT message: %v", err)
		codec.WriteFrame([]byte("ERROR.invalid_connect_message"))
		stream.Close()
		return
	}

	log.Printf("Service requested: %s", connReq.Dest)

	log.Printf("Dialing %s...", connReq.Dest)
	tcpConn, err := net.Dial("tcp", connReq.Dest)
	if err != nil {
		log.Printf("Failed to connect to %s", connReq.Dest)
		codec.WriteFrame([]byte("ERROR.connection_failed"))
		stream.Close()
		return
	}

	dk, _ := mlkem.GenerateKey768()
	encapsulationKey := dk.EncapsulationKey().Bytes()

	ourPubKeyEncoded := base64.StdEncoding.EncodeToString(encapsulationKey)
	log.Printf("Responding with our ECDH pubkey: %s", ourPubKeyEncoded)
	okResp := fmt.Sprintf("OK,%s", ourPubKeyEncoded)
	err = codec.WriteFrame([]byte(okResp))
	if err != nil {
		log.Printf("Error writing OK: %v", err)
		stream.Close()
		stream.CancelRead(0)
		tcpConn.Close()
		return
	}

	ciphertext, _ := codec.ReadFrame()
	log.Println("Got ciphertext from client")

	sharedSecret, err := dk.Decapsulate(ciphertext)
	if err != nil {
		log.Println("Some error getting shared secret", err)
	} else {
		log.Println("shared secret", fmt.Sprintf("%x", sha256.Sum256(sharedSecret)))
	}

	pipeWithEncryption(stream, tcpConn, sharedSecret)
	log.Println("Pipes finished, handleStream returning")
}

func pipeWithEncryption(quicStream *quic.Stream, tcpConn net.Conn, sharedSecret []byte) {
	log.Println("Piping traffic between stream & TCP socket")

	frameCodec := lpstream.NewFrameCodec(quicStream)
	gcmCodec := newGcmCodec(sharedSecret, frameCodec)

	done := make(chan struct{}, 2)
	go pipe(tcpConn, gcmCodec, done)
	go pipe(gcmCodec, tcpConn, done)

	<-done
	<-done

	tcpConn.Close()
	quicStream.Close()
}

func (c *Coordinator) Close() error {
	log.Println("Shutting down coordinator")
	c.connection.CloseWithError(quic.ApplicationErrorCode(0), "Agent going away")
	return c.controlStream.Close()
}

// get current unix time for signing & sending to the relay (for replay protection)
func timeNowBytes() []byte {
	// current timestamp
	unixTime := time.Now().Unix()
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(unixTime))
	return buf
}
