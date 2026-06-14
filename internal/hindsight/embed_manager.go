package hindsight

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"os"
	stdexec "os/exec"
	"strings"
	"sync"
	"time"

	"github.com/tlarevo/hindsight-tui/internal/config"
	"github.com/tlarevo/hindsight-tui/internal/process"
)

const (
	defaultEmbedCommand   = "hindsight-embed"
	defaultEmbedAPIURL    = "http://127.0.0.1:8888"
	defaultEmbedTimeout   = 30 * time.Second
	defaultEmbedHealthTTL = 5 * time.Second
)

// ErrEmbedNotInstalled is returned (wrapped) when the hindsight-embed
// executable cannot be found on PATH.
var ErrEmbedNotInstalled = stderrors.New("hindsight-embed is not installed")

type EmbedManager struct {
	Command   string
	APIURL    string
	Runner    process.Runner
	Timeout   time.Duration
	HealthTTL time.Duration

	mu           sync.Mutex
	healthyUntil time.Time
}

func EmbedInstallHint() string {
	return "hindsight-embed was not found. Install it with one of:\n" +
		"  uvx hindsight-embed@latest configure\n" +
		"  pipx install hindsight-embed\n" +
		"Then run: hindsight-embed configure"
}

func (m *EmbedManager) CheckInstalled(ctx context.Context) error {
	_, err := m.run(ctx, "--help")
	if err != nil {
		return err
	}
	return nil
}

func (m *EmbedManager) EnsureRunning(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Memoized health: a recent success also serializes the parallel cmds
	// Bubble Tea fires per screen load, which would otherwise each fork a
	// status/start subprocess.
	if time.Now().Before(m.healthyUntil) {
		return nil
	}

	statusOutput, statusErr := m.run(ctx, "daemon", "status")
	if statusErr != nil && stderrors.Is(statusErr, ErrEmbedNotInstalled) {
		return statusErr
	}

	if pingHealth(ctx, m.apiURLOrDefault()) == nil {
		m.healthyUntil = time.Now().Add(m.healthTTLOrDefault())
		return nil
	}

	// Daemon is not up: start it. A non-install error here is ignored (the
	// daemon may already be starting); its output is kept for diagnostics.
	startOutput, startErr := m.run(ctx, "daemon", "start")
	if startErr != nil && stderrors.Is(startErr, ErrEmbedNotInstalled) {
		return startErr
	}

	timeout := m.timeoutOrDefault()
	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		lastErr = pingHealth(pollCtx, m.apiURLOrDefault())
		if lastErr == nil {
			m.healthyUntil = time.Now().Add(m.healthTTLOrDefault())
			return nil
		}

		select {
		case <-pollCtx.Done():
			return m.timeoutError(statusOutput, statusErr, startOutput, lastErr)
		case <-ticker.C:
		}
	}
}

func (m *EmbedManager) Status(ctx context.Context) (string, error) {
	output, err := m.run(ctx, "daemon", "status")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (m *EmbedManager) Stop(ctx context.Context) error {
	_, err := m.run(ctx, "daemon", "stop")
	m.mu.Lock()
	m.healthyUntil = time.Time{}
	m.mu.Unlock()
	return err
}

func (m *EmbedManager) Logs(ctx context.Context, follow bool) ([]byte, error) {
	args := []string{"daemon", "logs"}
	if follow {
		args = append(args, "--follow")
	}
	return m.run(ctx, args...)
}

func (m *EmbedManager) Configure(ctx context.Context) error {
	_, err := m.run(ctx, "configure")
	return err
}

func (m *EmbedManager) run(ctx context.Context, args ...string) ([]byte, error) {
	runner := m.runnerOrDefault()
	output, err := runner.Run(ctx, m.commandOrDefault(), args...)
	if err != nil && stderrors.Is(err, stdexec.ErrNotFound) {
		return output, fmt.Errorf("%s: %w", EmbedInstallHint(), ErrEmbedNotInstalled)
	}
	return output, err
}

func (m *EmbedManager) commandOrDefault() string {
	if m != nil && strings.TrimSpace(m.Command) != "" {
		return m.Command
	}
	if managed, err := config.ManagedExecutablePath(defaultEmbedCommand); err == nil {
		if info, statErr := os.Stat(managed); statErr == nil && !info.IsDir() {
			return managed
		}
	}
	return defaultEmbedCommand
}

func (m *EmbedManager) apiURLOrDefault() string {
	if m == nil || strings.TrimSpace(m.APIURL) == "" {
		return defaultEmbedAPIURL
	}
	return strings.TrimRight(m.APIURL, "/")
}

func (m *EmbedManager) runnerOrDefault() process.Runner {
	if m == nil || m.Runner == nil {
		return process.ExecRunner{}
	}
	return m.Runner
}

func (m *EmbedManager) timeoutOrDefault() time.Duration {
	if m == nil || m.Timeout <= 0 {
		return defaultEmbedTimeout
	}
	return m.Timeout
}

func (m *EmbedManager) healthTTLOrDefault() time.Duration {
	if m == nil || m.HealthTTL <= 0 {
		return defaultEmbedHealthTTL
	}
	return m.HealthTTL
}

func pingHealth(ctx context.Context, baseURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return err
	}

	client := http.Client{Timeout: time.Second}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("health returned %s", response.Status)
	}
	return nil
}

func (m *EmbedManager) timeoutError(statusOutput []byte, statusErr error, startOutput []byte, lastErr error) error {
	message := fmt.Sprintf(
		"timed out waiting for %s health at %s/health after %s (attempted 'daemon start'); try 'hindsight-embed daemon logs'. Common causes: port 8888 conflict or missing HINDSIGHT_EMBED_LLM_API_KEY.",
		m.commandOrDefault(),
		m.apiURLOrDefault(),
		m.timeoutOrDefault(),
	)
	if statusErr != nil {
		message += fmt.Sprintf(" daemon status failed: %v.", statusErr)
	}
	if trimmed := strings.TrimSpace(string(statusOutput)); trimmed != "" {
		message += fmt.Sprintf(" daemon status output: %s.", trimmed)
	}
	if trimmed := strings.TrimSpace(string(startOutput)); trimmed != "" {
		message += fmt.Sprintf(" daemon start output: %s.", trimmed)
	}
	if lastErr != nil {
		message += fmt.Sprintf(" last health error: %v.", lastErr)
	}
	return stderrors.New(message)
}
