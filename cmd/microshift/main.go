package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ausil/microshift-2.0/pkg/config"
	"github.com/ausil/microshift-2.0/pkg/daemon"
	"github.com/ausil/microshift-2.0/pkg/version"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		runCmd(os.Args[1:])
		return
	}

	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "version":
		fmt.Printf("MicroShift %s (commit: %s)\n", version.Version, version.GitCommit)
	case "show-config":
		showConfigCmd(os.Args[2:])
	default:
		runCmd(os.Args[1:])
	}
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "/etc/microshift/config.yaml", "path to configuration file")
	fs.Parse(args)

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		// If config file doesn't exist, use defaults
		cfg = config.NewDefaultConfig()
		log.Printf("Using default configuration: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	d := daemon.New(cfg)
	if err := d.Run(); err != nil {
		log.Fatalf("MicroShift failed: %v", err)
	}
}

func showConfigCmd(args []string) {
	fs := flag.NewFlagSet("show-config", flag.ExitOnError)
	configPath := fs.String("config", "/etc/microshift/config.yaml", "path to configuration file")
	fs.Parse(args)

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		cfg = config.NewDefaultConfig()
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatalf("Failed to marshal config: %v", err)
	}
	fmt.Print(string(data))
}
