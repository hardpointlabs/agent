package control

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"time"

	_ "embed"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlog"
)

//go:embed ca.crt
var caCert []byte

func loadCACertPool() (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA cert")
	}
	return certPool, nil
}

const agentProtocol = "hp-1.0"

func DialRelay(relayAddress string, skipTls bool) (*quic.Conn, error) {

	caCertPool, err := loadCACertPool()
	if err != nil {
		log.Println("Unable to load CA cert")
		return nil, err
	}
	tlsConf := &tls.Config{
		RootCAs:            caCertPool,
		NextProtos:         []string{agentProtocol},
		InsecureSkipVerify: skipTls,
	}
	quicConfig := &quic.Config{
		HandshakeIdleTimeout: 10 * time.Second,
		KeepAlivePeriod:      30 * time.Second,
		Tracer:               qlog.DefaultConnectionTracer,
	}
	log.Printf("Connecting to relay %s...\n", relayAddress)
	return quic.DialAddr(context.Background(), relayAddress, tlsConf, quicConfig)
}
