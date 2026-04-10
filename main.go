package main

import (
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

	conn, err := control.DialRelay(args.Relay, args.SkipTls)
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
		IgnoreEnv: true,
	}, &args)
	if err != nil {
		log.Fatalf("Failed to create argument parser: %v", err)
	}
	p.MustParse(os.Args[1:])

	switch {
	case args.ListenCmd != nil:
		agentConf, err := config.ParseAgentConfig(args.Config)
		if err != nil {
			log.Fatalf("Couldn't load config file: %v", err)
		}
		args.AgentConfig = agentConf
		if err := clientMain(args); err != nil {
			log.Panicf("Something went wrong: %v\n", err)
		}
	default:
		p.WriteHelp(os.Stdout)
	}
}
