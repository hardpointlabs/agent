package control

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/hardpointlabs/agent/auth"
	"github.com/hardpointlabs/lpstream"
	"github.com/quic-go/quic-go"
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
	if err := c.WriteFrame(c.helloMessage(c.OrgId)); err != nil {
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

func (c *controlStream) findService(name string) *string {
	return &name
}

func pipe(reader io.Reader, writer io.Writer, closer io.Closer, done chan<- struct{}) {
	defer func() {
		log.Printf("pipe done: reader=%T, writer=%T", reader, writer)
		done <- struct{}{}
	}()
	n, err := io.Copy(writer, reader)
	log.Printf("io.Copy transferred %d bytes, err=%v", n, err)
	closer.Close()
	log.Printf("closed closer: %T", closer)
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

func (c *controlStream) helloMessage(orgId string) []byte {
	fingerprint := c.keyPair.Fingerprint()
	timestamp := timeNowBytes()

	helloMessage := []byte("HELLO.")
	helloMessage = append(helloMessage, fingerprint...)
	helloMessage = append(helloMessage, '.')
	helloMessage = append(helloMessage, []byte(orgId)...)
	helloMessage = append(helloMessage, '.')
	helloMessage = append(helloMessage, timestamp...)
	helloMessage = append(helloMessage, '.')
	sig, err := c.keyPair.Sign(helloMessage)
	if err != nil {
		log.Panicf("Error signing HELLO message %v\n", err)
	}

	helloMessage = append(helloMessage, sig...)
	return helloMessage
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
			log.Printf("Error accepting stream: %v", err)
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

	msgStr := string(msg)
	if !strings.HasPrefix(msgStr, "CONNECT.") {
		log.Printf("Unexpected message: %s", msgStr)
		stream.Close()
		return
	}

	serviceName := strings.TrimPrefix(msgStr, "CONNECT.")
	log.Printf("Service requested: %s", serviceName)

	service := c.controlStream.findService(serviceName)
	if service == nil {
		log.Printf("Unknown service: %s", serviceName)
		codec.WriteFrame([]byte("ERROR.unknown_service"))
		stream.Close()
		return
	}

	log.Printf("Dialing %s...", *service)
	tcpConn, err := net.Dial("tcp", *service)
	if err != nil {
		log.Printf("Failed to connect to %s", *service)
		codec.WriteFrame([]byte("ERROR.connection_failed"))
		stream.Close()
		return
	}

	log.Println("Writing OK to stream")
	err = codec.WriteFrame([]byte("OK"))
	if err != nil {
		log.Printf("Error writing OK: %v", err)
		return
	}
	log.Println("OK written, starting pipes (raw byte mode)")

	done := make(chan struct{}, 2)
	go pipe(tcpConn, stream, stream, done)
	go pipe(stream, tcpConn, tcpConn, done)

	log.Println("Waiting for pipes to finish")
	<-done
	<-done
	log.Println("Pipes finished, handleStream returning")
}

func (c *Coordinator) Close() error {
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
