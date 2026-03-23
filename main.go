package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/hardpointlabs/agent/auth"
	"github.com/hardpointlabs/agent/control"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlog"
	"golang.org/x/sync/errgroup"
)

// Service config file
type ServiceConfig struct {
	Name string            `json:"name"`
	Host string            `json:"host"`
	Port int16             `json:"port"`
	Tags map[string]string `json:"tags"`
}

type AgentConfig struct {
	OrgId    string          `json:"org_id"`
	Services []ServiceConfig `json:"services"`
}

// CLI Arguments
type Args struct {
	SkipTls       bool           `arg:"--skip-tls,env:SKIP_TLS" default:"false" help:"Bypass TLS certificate validation"`
	Relay         string         `arg:"env" default:"relay.hardpoint.dev:443" help:"Relay endpoint"`
	Config        string         `arg:"required,env" help:"Path to configuration file"`
	KeyDir        string         `arg:"key-dir",env:"KEY_DIR" default:"/var/lib/hardpoint"`
	ServiceConfig *ServiceConfig `arg:"-"`
}

func parseServiceConfig(args Args) (*ServiceConfig, error) {
	file, err := os.Open(args.Config)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var serviceConfig *ServiceConfig
	err = decoder.Decode(&serviceConfig)
	if err != nil {
		return nil, err
	}
	return serviceConfig, nil
}

func clientMain(args Args) error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: args.SkipTls,
		NextProtos:         []string{"quic-echo-example"},
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
	} else {
		log.Println("Relay connection established")
	}
	defer conn.CloseWithError(0, "")

	keyPair, err := auth.LoadOrCreateKeyPair(args.KeyDir)
	if err != nil {
		log.Println("Unable to load/create key pair")
		return err
	}

	coordinator, err := control.CreateCoordinator(conn, keyPair)
	if err != nil {
		return nil
	}
	defer coordinator.Close()

	return nil
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
	var args Args
	p, err := arg.NewParser(arg.Config{
		EnvPrefix: "HARDPOINT_",
	}, &args)
	p.MustParse(os.Args[1:])

	serviceConf, err := parseServiceConfig(args)
	if err != nil {
		p.Fail(fmt.Sprintf("Couldn't load config file: %v", err))
	}

	args.ServiceConfig = serviceConf

	err = clientMain(args)
	if err != nil {
		log.Panicf("Something went wrong: %v\n", err)
	}
}
