package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type AgentConfig struct {
	OrgId string `yaml:"org_id"`
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
	Config      string       `arg:"--config,env" default:"/etc/hardpointd/config.yaml" help:"Path to configuration file"`
}

type ListenCmd struct {
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

	var agentConfig AgentConfig
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&agentConfig)
	if err != nil {
		return nil, err
	}
	return &agentConfig, nil
}
