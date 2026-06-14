package errors

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"
	"strings"

	appconfig "github.com/tlarevo/hindsight-tui/internal/config"
	"github.com/tlarevo/hindsight-tui/internal/hindsight"
)

type UserError struct {
	Title  string
	Detail string
	Fixes  []string
}

func Friendly(err error) UserError {
	if err == nil {
		return UserError{}
	}

	var malformed *appconfig.MalformedConfigError
	if stderrors.As(err, &malformed) {
		return UserError{
			Title:  "Config is invalid",
			Detail: malformed.Error(),
			Fixes: []string{
				"Open Config and correct the YAML syntax.",
				"Compare with example.config.yaml.",
			},
		}
	}

	if looksLikeMissingEmbed(err) {
		return UserError{
			Title:  "hindsight-embed is not installed",
			Detail: err.Error(),
			Fixes: []string{
				"Install hindsight-embed with uvx or pipx.",
				"Run hindsight-embed configure.",
			},
		}
	}

	var se *hindsight.StatusError
	if stderrors.As(err, &se) {
		switch se.Code {
		case 401, 403:
			return UserError{
				Title:  "Authorization failed",
				Detail: err.Error(),
				Fixes: []string{
					"Check the configured API URL.",
					"Set the required provider credentials in your environment.",
					"If you use an Authorization header, make sure it is complete.",
				},
			}
		case 404, 405:
			return UserError{
				Title:  "Endpoint is unavailable",
				Detail: err.Error(),
				Fixes: []string{
					"Upgrade the Hindsight server to a version that exposes this feature.",
					"Use another view or disable this workflow on older servers.",
				},
			}
		case 400, 422:
			return UserError{
				Title:  "Input needs attention",
				Detail: err.Error(),
				Fixes: []string{
					"Review the highlighted form fields.",
					"Remove invalid metadata or unsupported values.",
				},
			}
		}
	}

	if isConnectionRefused(err) {
		return UserError{
			Title:  "Cannot reach the Hindsight API",
			Detail: err.Error(),
			Fixes: []string{
				"Start hindsight-embed or hindsight-api.",
				"Check whether port 8888 is already in use.",
				"Verify the configured API URL.",
			},
		}
	}

	if stderrors.Is(err, context.DeadlineExceeded) {
		return UserError{
			Title:  "Request timed out",
			Detail: err.Error(),
			Fixes: []string{
				"Increase timeout_ms in Config.",
				"Check server load and network.",
			},
		}
	}

	return UserError{
		Title:  "Request failed",
		Detail: err.Error(),
		Fixes:  []string{"Retry the action after checking your configuration and server status."},
	}
}

func looksLikeMissingEmbed(err error) bool {
	if stderrors.Is(err, hindsight.ErrEmbedNotInstalled) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "executable file not found")
}

func isConnectionRefused(err error) bool {
	var opErr *net.OpError
	if stderrors.As(err, &opErr) {
		if strings.Contains(strings.ToLower(opErr.Err.Error()), "refused") {
			return true
		}
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "connection refused") || strings.Contains(text, "dial tcp")
}

func (u UserError) Error() string {
	if u.Title == "" {
		return u.Detail
	}
	if u.Detail == "" {
		return u.Title
	}
	return fmt.Sprintf("%s: %s", u.Title, u.Detail)
}
