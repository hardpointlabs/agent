package control

import (
	"context"
	"encoding/binary"
	"errors"
	"log"
	"time"

	"github.com/hardpointlabs/agent/auth"
	"github.com/hardpointlabs/lpstream"
	"github.com/quic-go/quic-go"
)

type Service struct {
	Name string
	Tags map[string]string
}

type HelloMessage struct {
	OrgId    string
	Services []Service
}

type ConnectMessage struct {
	ServiceName string
	MessageId   []byte
}

// we need some kind of state machine where we
// 1) say hello to the relay, state who we are and what services we have
// 2) block until we receive an OK from the relay
// 3) wait for relay to send connect requests
// 4) Probably crash obviously if the stream is closed unexpectedly

type Coordinator struct {
	connection    *quic.Conn
	controlStream *controlStream
	keyPair       *auth.KeyPair
}

type controlStream struct {
	*lpstream.FrameCodec
	QuicStream *quic.Stream
}

func (c *controlStream) Close() error {
	return c.QuicStream.Close()
}

func (c *controlStream) HelloMessage() []byte {
	return append([]byte("HELLO"), timeNowBytes()...)
}

func CreateCoordinator(connection *quic.Conn, keyPair *auth.KeyPair) (*Coordinator, error) {
	stream, err := connection.OpenStreamSync(context.Background())
	if err != nil {
		return nil, err
	}
	controlStream := &controlStream{QuicStream: stream, FrameCodec: lpstream.NewFrameCodec(stream)}
	return &Coordinator{connection: connection, controlStream: controlStream, keyPair: keyPair}, nil
}

func (c *Coordinator) Start() error {
	fingerprint := c.keyPair.Fingerprint()
	timestamp := timeNowBytes()

	helloDigest := append(append(append(append([]byte("HELLO."), fingerprint...), '.'), timestamp...), '.')
	sig, err := c.keyPair.Sign(helloDigest)
	helloDigest = append(helloDigest, sig...)
	if err != nil {
		log.Println("Error signing")
		return err
	}

	if err := c.controlStream.WriteFrame(helloDigest); err != nil {
		log.Println("Error writing hello")
		return err
	}

	resp, err := c.controlStream.ReadFrame()
	if err != nil {
		return err
	}
	if string(resp) != "OK" {
		return ErrHandshakeFailed
	}

	log.Println("DONE!!!")

	return nil
}

var ErrHandshakeFailed = errors.New("handshake failed: server did not respond with OK")

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
