package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/hardpointlabs/agent/auth"
	"github.com/hardpointlabs/agent/config"
	"github.com/hardpointlabs/agent/control"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlog"
	"golang.org/x/sync/errgroup"
)

const agentProtocol = "hp-1.0"

func clientMain(args config.Args) error {
	keyPair, err := auth.LoadOrCreateKeyPair(args.KeyDir)
	if err != nil {
		log.Println("Unable to load/create key pair")
		return err
	}
	log.Printf("Using key pair with fingerprint %s to identify this agent", keyPair.Fingerprint())

	tlsConf := &tls.Config{
		InsecureSkipVerify: args.SkipTls,
		NextProtos:         []string{agentProtocol},
	}
	quicConfig := &quic.Config{
		HandshakeIdleTimeout: 10 * time.Second,
		KeepAlivePeriod:      30 * time.Second,
		Tracer:               qlog.DefaultConnectionTracer,
	}

	log.Println("Connecting to relay...")
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

// copy bytes in one direction between 2 connections
func copy(group *errgroup.Group, dst io.Writer, src io.Reader) {
	group.Go(func() error {
		for {
			_, err := io.Copy(dst, src)
			if err != nil {
				log.Printf("Error copying: %v\n", err)
			}
		}
	})
}

func main() {
	log.Println("Agent started")
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
