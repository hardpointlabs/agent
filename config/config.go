package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type AgentConfig struct {
	OrgId string `json:"org_id"`
}

func (ac *AgentConfig) String() string {
	return fmt.Sprintf("Org ID: %s", ac.OrgId)
}

// CLI Arguments
type Args struct {
	SkipTls     bool         `arg:"--skip-tls,env:SKIP_TLS" default:"false" help:"Bypass TLS certificate validation"`
	Relay       string       `arg:"env" default:"relay.hardpoint.dev:443" help:"Relay endpoint"`
	KeyDir      string       `default:"/var/lib/hardpoint"`
	AgentConfig *AgentConfig `arg:"-"`
	ListenCmd   *ListenCmd   `arg:"subcommand:listen" help:"Start the agent and listen for connections"`
}

type ListenCmd struct {
	Config string `arg:"required,--config,env" default "/etc/hardpointd/config.yaml" help:"Path to configuration file"`
}

func (Args) Version() string {
	return fmt.Sprintf("agent %s (%.7s)", Version, Commit)
}

func ParseAgentConfig(configPath string) (*AgentConfig, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var agentConfig *AgentConfig
	err = decoder.Decode(&agentConfig)
	if err != nil {
		return nil, err
	}
	return agentConfig, nil
}
