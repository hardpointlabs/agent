package main

import (
	"encoding/json"
	"fmt"
	"os"
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

func (ac *AgentConfig) String() string {
	return fmt.Sprintf("Org ID: %s", ac.OrgId)
}

// CLI Arguments
type Args struct {
	SkipTls       bool           `arg:"--skip-tls,env:SKIP_TLS" default:"false" help:"Bypass TLS certificate validation"`
	Relay         string         `arg:"env" default:"relay.hardpoint.dev:443" help:"Relay endpoint"`
	Config        string         `arg:"required,env" help:"Path to configuration file"`
	KeyDir        string         `default:"/var/lib/hardpoint"`
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
