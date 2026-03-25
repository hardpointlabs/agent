package control

import (
	"context"
	"encoding/binary"
	"errors"
	"log"
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

func CreateCoordinator(connection *quic.Conn, keyPair *auth.KeyPair, config *config.AgentConfig) (*Coordinator, error) {
	stream, err := connection.OpenStreamSync(context.Background())
	if err != nil {
		return nil, err
	}
	controlStream := &controlStream{QuicStream: stream, FrameCodec: lpstream.NewFrameCodec(stream), AuthState: StateHello, keyPair: keyPair, OrgId: config.OrgId}
	return &Coordinator{connection: connection, controlStream: controlStream}, nil
}

func (c *Coordinator) Start() error {
	err := c.controlStream.doHandshake()
	if err == nil {
		log.Println("Handshake succeeded")
	}

	return err
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
