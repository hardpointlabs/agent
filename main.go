package main

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/alexflint/go-arg"
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
	SkipTls bool   `arg:"--skip-tls,env:SKIP_TLS" default:"false" help:"Bypass TLS certificate validation"`
	Relay   string `arg:"env" default:"relay.hardpoint.dev:443" help:"Relay endpoint"`
	Config  string `arg:"required,env" help:"Path to configuration file"`
}

func clientMain(args Args) error {
	// err := json.Unmarshal(jsonData, &config)
	log.Printf("Skip TLS verify? %t\n", args.SkipTls)
	tlsConf := &tls.Config{
		InsecureSkipVerify: args.SkipTls,
		NextProtos:         []string{"quic-echo-example"},
	}
	quicConfig := &quic.Config{
		HandshakeIdleTimeout: 10 * time.Second,
		KeepAlivePeriod:      30 * time.Second,
		Tracer:               qlog.DefaultConnectionTracer,
	}
	conn, err := quic.DialAddr(context.Background(), args.Relay, tlsConf, quicConfig)
	if err != nil {
		log.Println("CONN ERR")
		return err
	}
	defer conn.CloseWithError(0, "")

	stream, err := conn.OpenStreamSync(context.Background())

	if err != nil {
		log.Println("STREAM ERR")
		return err
	} else {
		log.Println("Connected to relay")
	}
	defer stream.Close()

	servAddr := "localhost:6379"
	tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
	if err != nil {
		return err
	}

	clientSocket, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return err
	} else {
		log.Printf("Connected to %s\n", servAddr)
	}

	var g errgroup.Group
	copy(&g, stream, clientSocket)
	copy(&g, clientSocket, stream)
	return g.Wait()
}

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
	var args Args
	p, err := arg.NewParser(arg.Config{
		EnvPrefix: "HARDPOINT_",
	}, &args)
	p.MustParse(os.Args[1:])

	err = clientMain(args)
	if err != nil {
		log.Panicf("Something went wrong: %v\n", err)
	}
}
