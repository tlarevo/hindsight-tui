package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"hindsight-tui/internal/config"
	"hindsight-tui/internal/hindsight"
)

func loadConfigForRun(opts Options) (config.Config, error) {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return config.Config{}, err
	}

	if opts.BackendOverride != "" {
		backend, err := parseBackend(opts.BackendOverride)
		if err != nil {
			return config.Config{}, err
		}
		cfg.Backend = backend
	}

	if opts.APIURLOverride != "" {
		cfg.APIURL = opts.APIURLOverride
	}

	cfg.AuthToken = opts.AuthTokenOverride
	if cfg.AuthToken == "" {
		cfg.AuthToken = os.Getenv("HINDSIGHT_TUI_AUTH_TOKEN")
	}

	return cfg, nil
}

func parseBackend(value string) (config.Backend, error) {
	switch strings.TrimSpace(value) {
	case "embed":
		return config.BackendEmbed, nil
	case "http":
		return config.BackendHTTP, nil
	case "demo":
		return config.BackendDemo, nil
	default:
		return "", fmt.Errorf("invalid backend %q", value)
	}
}

func runDoctor(cfg config.Config, client hindsight.Client) error {
	ctx := context.Background()

	health, err := client.Health(ctx)
	if err != nil {
		return err
	}

	version, err := client.Version(ctx)
	if err != nil {
		return err
	}

	banks, err := client.ListBanks(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("backend: %s\n", cfg.Backend)
	fmt.Printf("api_url: %s\n", cfg.APIURL)
	fmt.Printf("health_ok: %t\n", health.OK)
	fmt.Printf("health_detail: %s\n", strings.TrimSpace(health.Detail))
	if version != nil {
		fmt.Printf("api_version: %s\n", version.APIVersion)
	}
	fmt.Printf("bank_count: %d\n", len(banks))
	for _, bank := range banks {
		fmt.Printf("bank: %s\n", bank.BankID)
	}

	return nil
}
