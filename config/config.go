package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alexflint/go-arg"
	"gopkg.in/yaml.v3"
)

func isContainer() bool {
	//  docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// podman
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}

	file, err := os.Open("/proc/1/cgroup")
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if containsContainerMarker(line) {
			return true
		}
	}
	return false
}

func containsContainerMarker(s string) bool {
	markers := []string{"docker", "lxc", "containerd", "kubepods"}
	for _, marker := range markers {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

type agentConfig struct {
	OrgId string `yaml:"org_id"`
}

func (ac *agentConfig) String() string {
	return fmt.Sprintf("Org ID: %s", ac.OrgId)
}

// CLI Arguments
type Args struct {
	SkipTls        bool            `arg:"--skip-tls,env:SKIP_TLS" default:"false" help:"Bypass TLS certificate validation"`
	Relay          string          `arg:"env" default:"relay.hardpoint.dev:443" help:"Relay endpoint"`
	KeyDir         string          `arg:"--key-dir,env" help:"Directory for storing agent key pairs"`
	ConnectCmd     *ConnectCmd     `arg:"subcommand:connect" help:"Connect to your Hardpoint network to serve traffic"`
	FingerprintCmd *FingerprintCmd `arg:"subcommand:fingerprint" help:"Print the agent fingerprint"`
	InitCmd        *InitCmd        `arg:"subcommand:init" help:"Configure the agent the first time on this machine"`
	Config         string          `arg:"--config,env" help:"Path to configuration file"`
}

type InitCmd struct {
	OrgId string `arg:"--org-id,required" help:"Hardpoint Organization ID"`
}

type ConnectCmd struct {
	OrgId string `arg:"--org-id,env:ORG_ID" help:"Hardpoint Organization ID"`
}

type FingerprintCmd struct {
}

type ParseResult struct {
	Args   Args
	parser *arg.Parser
}

func (p *ParseResult) PrintUsage() {
	p.parser.WriteUsage(os.Stdout)
}

func (Args) Version() string {
	return fmt.Sprintf("agent %s (%.7s)", Version, Commit)
}

func (a *Args) parseAgentConfig() (*agentConfig, error) {
	file, err := os.Open(a.Config)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var agentConfig agentConfig
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&agentConfig)
	if err != nil {
		return nil, err
	}
	return &agentConfig, nil
}

func parseArgs() *ParseResult {
	var args Args
	p, err := arg.NewParser(arg.Config{
		EnvPrefix: "HARDPOINT_",
		IgnoreEnv: false,
	}, &args)
	if err != nil {
		log.Fatalf("Failed to create argument parser: %v", err)
	}
	p.MustParse(os.Args[1:])

	return &ParseResult{Args: args, parser: p}
}

func ParseArgsAndLayerDefaults() (*ParseResult, error) {
	result := parseArgs()
	parsed := result.Args

	if parsed.InitCmd == nil && parsed.ConnectCmd == nil && parsed.FingerprintCmd == nil {
		return result, nil
	}

	if parsed.InitCmd != nil {
		return result, nil
	}

	if parsed.KeyDir == "" {
		if isContainer() {
			tmpDir := "/tmp/hardpointd"
			if err := os.MkdirAll(tmpDir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create key directory: %w", err)
			}
			log.Printf("Running in a container, using %s as the key directory\n", tmpDir)
			parsed.KeyDir = tmpDir
		} else {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				configDir := filepath.Join(homeDir, ".config", "hardpointd")
				if _, err := os.Stat(configDir); err == nil {
					parsed.KeyDir = configDir
				} else if runtime.GOOS == "darwin" {
					if err := os.MkdirAll(configDir, 0755); err != nil {
						return nil, fmt.Errorf("failed to create key directory: %w", err)
					}
					parsed.KeyDir = configDir
				}
			}
			if parsed.KeyDir == "" {
				if _, err := os.Stat("/var/lib/hardpointd"); err == nil {
					parsed.KeyDir = "/var/lib/hardpointd"
				} else {
					return nil, fmt.Errorf("key directory not found")
				}
			}
		}
	}

	if parsed.ConnectCmd != nil {
		if parsed.ConnectCmd.OrgId == "" {
			if parsed.Config == "" {
				homeDir, err := os.UserHomeDir()
				if err == nil {
					configPath := filepath.Join(homeDir, ".config", "hardpoint", "config.yaml")
					if _, err := os.Stat(configPath); err == nil {
						parsed.Config = configPath
					}
				}
				if parsed.Config == "" {
					if _, err := os.Stat("/etc/hardpointd/config.yaml"); err == nil {
						parsed.Config = "/etc/hardpointd/config.yaml"
					} else {
						return nil, fmt.Errorf("config file not found")
					}
				}
			}

			agentConf, err := parsed.parseAgentConfig()
			if err != nil {
				log.Fatalf("Couldn't load config file: %v", err)
			}

			parsed.ConnectCmd.OrgId = agentConf.OrgId
		}
	}
	return result, nil
}

func (r *ParseResult) SetOrgId() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "hardpointd")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	orgId := r.Args.InitCmd.OrgId
	if _, err := file.WriteString(fmt.Sprintf("org_id: %s\n", orgId)); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Set Hardpoint Org ID to %s and created config file at %s\n", orgId, configPath)

	return nil
}
