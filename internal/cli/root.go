package cli

import (
	"github.com/spf13/cobra"

	"hindsight-tui/internal/app"
)

var version = "dev"

func newRootCmd(run func(app.Options) error) *cobra.Command {
	var configPath string
	var backend string
	var apiURL string
	var demo bool
	var doctor bool
	var authToken string
	var setup bool

	cmd := &cobra.Command{
		Use:     "hindsight-tui",
		Short:   "Hindsight terminal UI",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if demo {
				backend = "demo"
			}
			return run(app.Options{
				ConfigPath:        configPath,
				BackendOverride:   backend,
				APIURLOverride:    apiURL,
				AuthTokenOverride: authToken,
				Doctor:            doctor,
				Setup:             setup,
			})
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&configPath, "config", "", "config file override")
	flags.StringVar(&backend, "backend", "", "backend override: embed|http|demo")
	flags.StringVar(&apiURL, "api-url", "", "API URL override")
	flags.BoolVar(&demo, "demo", false, "alias for --backend demo")
	flags.BoolVar(&doctor, "doctor", false, "run diagnostics and exit")
	flags.BoolVar(&setup, "setup", false, "run interactive setup wizard")
	flags.StringVar(&authToken, "auth-token", "", "Authorization token for the Hindsight API (env: HINDSIGHT_TUI_AUTH_TOKEN)")

	return cmd
}

func Execute() error {
	return newRootCmd(app.Run).Execute()
}
