package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	_ "embed"

	"github.com/hardpointlabs/agent/auth"
	"github.com/hardpointlabs/agent/config"
	"github.com/hardpointlabs/agent/control"
	"golang.org/x/sync/errgroup"
)

func clientMain(args config.Args) error {
	log.Println("Agent started")
	keyPair, err := auth.LoadOrCreateKeyPair(args.KeyDir)
	if err != nil {
		log.Println("Unable to load/create key pair")
		return err
	}
	log.Printf("Using key pair with fingerprint %s to identify this agent", keyPair.Fingerprint())

	conn, err := control.DialRelay(args.Relay, args.SkipTls)
	if err != nil {
		log.Println("Unable to establish relay connection")
		return err
	}
	coordinator, err := control.CreateCoordinator(conn, keyPair, args.ConnectCmd.OrgId)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(context.Background())
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	g.Go(func() error {
		return coordinator.Start()
	})

	g.Go(func() error {
		<-ctx.Done()
		return coordinator.Close()
	})

	if err = g.Wait(); err != nil {
		fmt.Println("Terminated with error:", err)
	}
	log.Println("Agent stopped")

	return err
}

func main() {
	parsed, err := config.ParseArgsAndLayerDefaults()
	if err != nil {
		log.Fatalf("%v", err)
	}

	switch {
	case parsed.Args.FingerprintCmd != nil:
		auth.ReadFingerprintFromFile(parsed.Args.KeyDir)
	case parsed.Args.ConnectCmd != nil:
		if err := clientMain(parsed.Args); err != nil {
			log.Panicf("Failed to start tunnel: %v\n", err)
		}
	case parsed.Args.InitCmd != nil:
		parsed.SetOrgId()
	default:
		parsed.PrintUsage()
	}
}
