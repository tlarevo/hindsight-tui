package config

import (
	stderrors "errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

type Backend string

const (
	BackendEmbed Backend = "embed"
	BackendHTTP  Backend = "http"
	BackendDemo  Backend = "demo"
)

type Config struct {
	Backend     Backend `yaml:"backend"`
	APIURL      string  `yaml:"api_url"`
	DefaultBank string  `yaml:"default_bank"`
	Theme       string  `yaml:"theme"`
	Compact     bool    `yaml:"compact"`
	TimeoutMS   int     `yaml:"timeout_ms"`
	// AuthToken is runtime-only (CLI flag / env) and never persisted to disk.
	AuthToken string `yaml:"-"`
}

type MalformedConfigError struct {
	Path string
	Err  error
}

func (e *MalformedConfigError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("malformed config: %v", e.Err)
	}
	return fmt.Sprintf("malformed config %q: %v", e.Path, e.Err)
}

func (e *MalformedConfigError) Unwrap() error {
	return e.Err
}

func Default() Config {
	return Config{
		Backend:     BackendEmbed,
		APIURL:      "http://127.0.0.1:8888",
		DefaultBank: "default",
		Theme:       "auto",
		Compact:     false,
		TimeoutMS:   30000,
	}
}

func DefaultPath() (string, error) {
	configHome := xdg.ConfigHome
	if configHome == "" {
		return "", stderrors.New("xdg config home is empty")
	}
	return filepath.Join(configHome, "hindsight-tui", "config.yaml"), nil
}

func ResolvePath(path string) (string, error) {
	if path != "" {
		return path, nil
	}
	return DefaultPath()
}

func Load(path string) (Config, error) {
	cfg := Default()

	resolvedPath, err := ResolvePath(path)
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		if stderrors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config %q: %w", resolvedPath, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Default(), &MalformedConfigError{Path: resolvedPath, Err: err}
	}

	switch cfg.Backend {
	case BackendEmbed, BackendHTTP, BackendDemo, "":
	default:
		return Default(), &MalformedConfigError{Path: resolvedPath, Err: fmt.Errorf("invalid backend %q (expected embed|http|demo)", cfg.Backend)}
	}
	if cfg.TimeoutMS < 0 {
		return Default(), &MalformedConfigError{Path: resolvedPath, Err: fmt.Errorf("timeout_ms must be >= 0")}
	}

	if cfg.Backend == "" {
		cfg.Backend = BackendEmbed
	}
	if cfg.APIURL == "" {
		cfg.APIURL = Default().APIURL
	}
	if cfg.DefaultBank == "" {
		cfg.DefaultBank = Default().DefaultBank
	}
	if cfg.Theme == "" {
		cfg.Theme = Default().Theme
	}
	if cfg.TimeoutMS == 0 {
		cfg.TimeoutMS = Default().TimeoutMS
	}

	return cfg, nil
}

func Save(path string, cfg Config) error {
	resolvedPath, err := ResolvePath(path)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := os.WriteFile(resolvedPath, data, 0o600); err != nil {
		return fmt.Errorf("write config %q: %w", resolvedPath, err)
	}

	return nil
}
