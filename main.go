package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"time"

	_ "embed"

	"github.com/alexflint/go-arg"
	"github.com/hardpointlabs/agent/auth"
	"github.com/hardpointlabs/agent/config"
	"github.com/hardpointlabs/agent/control"
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

func clientMain(args config.Args) error {
	log.Println("Agent started")
	keyPair, err := auth.LoadOrCreateKeyPair(args.KeyDir)
	if err != nil {
		log.Println("Unable to load/create key pair")
		return err
	}
	log.Printf("Using key pair with fingerprint %s to identify this agent", keyPair.Fingerprint())

	caCertPool, err := loadCACertPool()
	if err != nil {
		log.Println("Unable to load CA cert")
		return err
	}

	tlsConf := &tls.Config{
		RootCAs:            caCertPool,
		NextProtos:         []string{agentProtocol},
		InsecureSkipVerify: args.SkipTls,
	}
	quicConfig := &quic.Config{
		HandshakeIdleTimeout: 10 * time.Second,
		KeepAlivePeriod:      30 * time.Second,
		Tracer:               qlog.DefaultConnectionTracer,
	}

	log.Printf("Connecting to relay %s...\n", args.Relay)
	conn, err := quic.DialAddr(context.Background(), args.Relay, tlsConf, quicConfig)
	if err != nil {
		log.Println("Unable to establish relay connection")
		return err
	}

	coordinator, err := control.CreateCoordinator(conn, keyPair, args.AgentConfig)
	if err != nil {
		return err
	}
	defer coordinator.Close()

	return coordinator.Start()
}

func main() {
	var args config.Args
	p, err := arg.NewParser(arg.Config{
		EnvPrefix: "HARDPOINT_",
	}, &args)
	if err != nil {
		log.Fatalf("Failed to create argument parser: %v", err)
	}
	p.MustParse(os.Args[1:])

	agentConf, err := config.ParseAgentConfig(args)
	if err != nil {
		p.Fail(fmt.Sprintf("Couldn't load config file: %v", err))
	}

	args.AgentConfig = agentConf

	err = clientMain(args)
	if err != nil {
		log.Panicf("Something went wrong: %v\n", err)
	}
}
