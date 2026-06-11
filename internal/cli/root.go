package cli

import (
	"github.com/spf13/cobra"

	"hindsight-tui/internal/app"
)

func Execute() error {
	var configPath string
	var backend string
	var apiURL string
	var demo bool
	var doctor bool
	var authToken string

	cmd := &cobra.Command{
		Use:   "hindsight-tui",
		Short: "Hindsight terminal UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			if demo {
				backend = "demo"
			}
			return app.Run(app.Options{
				ConfigPath:        configPath,
				BackendOverride:   backend,
				APIURLOverride:    apiURL,
				AuthTokenOverride: authToken,
				Doctor:            doctor,
			})
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&configPath, "config", "", "config file override")
	flags.StringVar(&backend, "backend", "", "backend override: embed|http|demo")
	flags.StringVar(&apiURL, "api-url", "", "API URL override")
	flags.BoolVar(&demo, "demo", false, "alias for --backend demo")
	flags.BoolVar(&doctor, "doctor", false, "run diagnostics and exit")
	flags.StringVar(&authToken, "auth-token", "", "Authorization token for the Hindsight API (env: HINDSIGHT_TUI_AUTH_TOKEN)")

	return cmd.Execute()
}
