package control

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/hardpointlabs/agent/auth"
	"github.com/hardpointlabs/agent/config"
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
	StateAdvertising
	StateOK
	StateError
)

type controlStream struct {
	*lpstream.FrameCodec
	OrgId      string
	AuthState  AuthState
	QuicStream *quic.Stream
	keyPair    *auth.KeyPair
	Services   []config.ServiceConfig
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
		c.AuthState = StateAdvertising
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
			c.AuthState = StateAdvertising
			return nil
		} else {
			c.AuthState = StateError
			log.Printf("Unknown response from relay: '%s'", string(resp))
			return ErrHandshakeFailed
		}
	}
}

func (c *controlStream) sendServices() error {
	servicesJSON, err := json.Marshal(c.Services)
	if err != nil {
		log.Println("Error marshaling services")
		c.AuthState = StateError
		return err
	}

	message := []byte("SERVICES.")
	message = append(message, servicesJSON...)

	log.Println("DOING SEND")
	if err := c.WriteFrame(message); err != nil {
		log.Println("Error writing SERVICES message")
		c.AuthState = StateError
		return err
	}
	log.Println("SENT")

	log.Println("READING")
	resp, err := c.ReadFrame()
	if err != nil {
		c.AuthState = StateError
		return err
	}
	log.Println("READ DONE")

	if string(resp) == "OK" {
		log.Println("Services advertised successfully")
		c.AuthState = StateOK
		return nil
	}

	log.Printf("Unexpected response to SERVICES: '%s'", string(resp))
	c.AuthState = StateError
	return ErrHandshakeFailed
}

func (c *controlStream) findService(name string) *config.ServiceConfig {
	for i := range c.Services {
		if c.Services[i].Name == name {
			return &c.Services[i]
		}
	}
	return nil
}

func pipe(reader io.Reader, writer io.Writer, closer io.Closer, done chan<- struct{}) {
	defer func() {
		done <- struct{}{}
	}()
	io.Copy(writer, reader)
	closer.Close()
}

func (c *controlStream) handleConnect(serviceName string, stream *quic.Stream) error {
	service := c.findService(serviceName)
	if service == nil {
		log.Printf("Unknown service: %s", serviceName)
		c.WriteFrame([]byte("ERROR.unknown_service"))
		stream.Close()
		return nil
	}

	addr := net.JoinHostPort(service.Host, fmt.Sprintf("%d", service.Port))
	tcpConn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", addr, err)
		c.WriteFrame([]byte("ERROR.connection_failed"))
		stream.Close()
		return nil
	}
	defer tcpConn.Close()

	c.WriteFrame([]byte("OK"))

	frameCodec := lpstream.NewFrameCodec(stream)
	quicReader := &quicStreamReader{FrameCodec: frameCodec}
	quicWriter := &quicStreamWriter{FrameCodec: frameCodec}

	done := make(chan struct{}, 2)
	go pipe(quicReader, tcpConn, tcpConn, done)
	go pipe(tcpConn, quicWriter, stream, done)

	<-done
	<-done
	return nil
}

type quicStreamReader struct {
	*lpstream.FrameCodec
}

func (r *quicStreamReader) Read(p []byte) (int, error) {
	frame, err := r.ReadFrame()
	if err != nil {
		return 0, err
	}
	n := copy(p, frame)
	return n, nil
}

type quicStreamWriter struct {
	*lpstream.FrameCodec
}

func (w *quicStreamWriter) Write(p []byte) (int, error) {
	err := w.WriteFrame(p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
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
		case StateAdvertising:
			err = c.sendServices()
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

func CreateCoordinator(connection *quic.Conn, keyPair *auth.KeyPair, config *config.AgentConfig) (*Coordinator, error) {
	stream, err := connection.OpenStreamSync(context.Background())
	if err != nil {
		return nil, err
	}
	controlStream := &controlStream{
		QuicStream: stream,
		FrameCodec: lpstream.NewFrameCodec(stream),
		AuthState:  StateHello,
		keyPair:    keyPair,
		OrgId:      config.OrgId,
		Services:   config.Services,
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
	frameCodec := lpstream.NewFrameCodec(stream)
	msg, err := frameCodec.ReadFrame()
	if err != nil {
		log.Printf("Error reading from stream: %v", err)
		stream.Close()
		return
	}

	msgStr := string(msg)
	if !strings.HasPrefix(msgStr, "CONNECT.") {
		log.Printf("Unexpected message: %s", msgStr)
		stream.Close()
		return
	}

	serviceName := strings.TrimPrefix(msgStr, "CONNECT.")
	c.controlStream.handleConnect(serviceName, stream)
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
