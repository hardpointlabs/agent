package main

import (
	"fmt"
	"log"
	"os"

	_ "embed"

	"github.com/alexflint/go-arg"
	"github.com/hardpointlabs/agent/auth"
	"github.com/hardpointlabs/agent/config"
	"github.com/hardpointlabs/agent/control"
)

func clientMain(args config.Args) error {
	log.Println("Agent started")
	keyPair, err := auth.LoadOrCreateKeyPair(args.KeyDir)
	if err != nil {
		log.Println("Unable to load/create key pair")
		return err
	}
	log.Printf("Using key pair with fingerprint %s to identify this agent", keyPair.Fingerprint())

	if err != nil {
		log.Println("Unable to establish relay connection")
		return err
	}

	conn, err := control.DialRelay(args.Relay, args.SkipTls)
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
