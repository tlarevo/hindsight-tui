package views

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/config"
	"hindsight-tui/internal/hindsight"
	"hindsight-tui/internal/state"
	"hindsight-tui/internal/theme"
)

const managedUVVersion = "0.11.21"

const (
	bootstrapStepWelcome = iota
	bootstrapStepBackend
	bootstrapStepInstall
	bootstrapStepAPIURL
	bootstrapStepAuthToken
	bootstrapStepBank
	bootstrapStepTheme
	bootstrapStepReview
)

var bootstrapBackends = []simpleListItem{
	{title: "embed", desc: "local embedded server", value: "embed"},
	{title: "http", desc: "remote HTTP API", value: "http"},
	{title: "demo", desc: "demo mode, no server needed", value: "demo"},
}

var bootstrapThemes = []simpleListItem{
	{title: "auto", desc: "follow terminal colors", value: "auto"},
	{title: "dark", desc: "dark theme", value: "dark"},
	{title: "light", desc: "light theme", value: "light"},
}

var bootstrapReviewActions = []simpleListItem{
	{title: "Save & Continue", value: "save"},
	{title: "Go Back", value: "back"},
}

type installResultMsg struct {
	Output string
	Err    error
}

type verifyResultMsg struct {
	OK  bool
	Err error
}

// BootstrapView is a step-by-step setup wizard for first-run configuration.
type BootstrapView struct {
	shared     *Shared
	step       int
	width      int
	height     int
	backendIdx int
	apiURL     string
	authToken  string
	bank       string
	themeIdx   int
	reviewIdx  int
	editor     fieldEditor
	verifyOK   bool
	verifyErr  string
	loading    bool

	// Install step fields
	embedInstalled  bool
	embedBinaryPath string
	installers      []simpleListItem
	installerIdx    int
	installOutput   string
	installRunning  bool
	installDone     bool
	installOK       bool
}

func NewBootstrapView(shared *Shared) *BootstrapView {
	cfg := shared.Config
	backendIdx := 0
	for i, item := range bootstrapBackends {
		if item.value == string(cfg.Backend) {
			backendIdx = i
			break
		}
	}
	themeIdx := 0
	for i, item := range bootstrapThemes {
		if item.value == cfg.Theme {
			themeIdx = i
			break
		}
	}
	return &BootstrapView{
		shared:     shared,
		step:       bootstrapStepWelcome,
		apiURL:     cfg.APIURL,
		authToken:  cfg.AuthToken,
		bank:       cfg.DefaultBank,
		backendIdx: backendIdx,
		themeIdx:   themeIdx,
	}
}

func (v *BootstrapView) Title() string          { return "Setup" }
func (v *BootstrapView) TextEntryFocused() bool { return v.editor.active }
func (v *BootstrapView) Init() tea.Cmd          { return nil }

func (v *BootstrapView) currentBackend() string {
	return bootstrapBackends[v.backendIdx].value
}

func (v *BootstrapView) currentTheme() string {
	return bootstrapThemes[v.themeIdx].value
}

func (v *BootstrapView) nextStep() int {
	switch v.step {
	case bootstrapStepWelcome:
		return bootstrapStepBackend
	case bootstrapStepBackend:
		backend := v.currentBackend()
		if backend == "demo" {
			return bootstrapStepBank
		}
		if backend == "http" {
			return bootstrapStepAPIURL
		}
		if v.embedInstalled {
			return bootstrapStepAPIURL
		}
		return bootstrapStepInstall
	case bootstrapStepInstall:
		return bootstrapStepAPIURL
	case bootstrapStepAPIURL:
		return bootstrapStepAuthToken
	case bootstrapStepAuthToken:
		return bootstrapStepBank
	case bootstrapStepBank:
		return bootstrapStepTheme
	case bootstrapStepTheme:
		return bootstrapStepReview
	default:
		return v.step
	}
}

func (v *BootstrapView) prevStep() int {
	switch v.step {
	case bootstrapStepReview:
		return bootstrapStepTheme
	case bootstrapStepTheme:
		return bootstrapStepBank
	case bootstrapStepBank:
		if v.currentBackend() == "demo" {
			return bootstrapStepBackend
		}
		return bootstrapStepAuthToken
	case bootstrapStepAuthToken:
		return bootstrapStepAPIURL
	case bootstrapStepAPIURL:
		if v.currentBackend() == "http" {
			return bootstrapStepBackend
		}
		if v.embedInstalled {
			return bootstrapStepBackend
		}
		return bootstrapStepInstall
	case bootstrapStepInstall:
		return bootstrapStepBackend
	case bootstrapStepBackend:
		return bootstrapStepWelcome
	default:
		return v.step
	}
}

func findEmbedBinary() (string, bool) {
	if path, err := exec.LookPath("hindsight-embed"); err == nil {
		return path, true
	}
	managed, err := config.ManagedExecutablePath("hindsight-embed")
	if err != nil {
		return "", false
	}
	if info, err := os.Stat(managed); err == nil && !info.IsDir() {
		return managed, true
	}
	return "", false
}
func (v *BootstrapView) advanceStep() tea.Cmd {
	next := v.nextStep()
	if next == bootstrapStepInstall {
		if path, ok := findEmbedBinary(); ok {
			v.embedInstalled = true
			v.embedBinaryPath = path
			next = bootstrapStepAPIURL
		} else {
			v.installers = nil
			v.installerIdx = 0
			if _, err := exec.LookPath("uv"); err == nil {
				v.installers = append(v.installers, simpleListItem{
					title: "uv",
					desc:  "uv tool install (recommended)",
					value: "uv",
				})
			} else {
				v.installers = append(v.installers, simpleListItem{
					title: "Managed uv",
					desc:  "download uv into Hindsight TUI data dir",
					value: "managed-uv",
				})
			}
			for _, inst := range []struct{ name, desc string }{
				{"pipx", "pipx install (isolated venv)"},
				{"pip", "pip install --user"},
			} {
				if _, err := exec.LookPath(inst.name); err == nil {
					v.installers = append(v.installers, simpleListItem{
						title: inst.name,
						desc:  inst.desc,
						value: inst.name,
					})
				}
			}
			v.installers = append(v.installers, simpleListItem{
				title: "Skip",
				desc:  "continue without installing",
				value: "skip",
			})
		}
	}
	if next == bootstrapStepReview {
		v.loading = true
	}
	v.step = next
	if v.step == bootstrapStepReview {
		return v.verifyConnCmd()
	}
	return nil
}

func (v *BootstrapView) retreatStep() {
	v.step = v.prevStep()
}

func (v *BootstrapView) buildConfig() config.Config {
	cfg := *v.shared.Config
	cfg.Backend = config.Backend(v.currentBackend())
	cfg.APIURL = v.apiURL
	cfg.DefaultBank = v.bank
	cfg.Theme = v.currentTheme()
	cfg.AuthToken = v.authToken
	return cfg
}

func managedUVTarget() (string, error) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		return "aarch64-apple-darwin", nil
	case "darwin/amd64":
		return "x86_64-apple-darwin", nil
	case "linux/arm64":
		return "aarch64-unknown-linux-gnu", nil
	case "linux/amd64":
		return "x86_64-unknown-linux-gnu", nil
	default:
		return "", fmt.Errorf("managed uv bootstrap is unsupported on %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

func managedUVPath() (string, error) {
	return config.ManagedExecutablePath("uv")
}

func ensureManagedUV(ctx context.Context) (string, string, error) {
	uvPath, err := managedUVPath()
	if err != nil {
		return "", "", err
	}
	if info, err := os.Stat(uvPath); err == nil && !info.IsDir() {
		return uvPath, fmt.Sprintf("Using managed uv at %s\n", uvPath), nil
	}

	target, err := managedUVTarget()
	if err != nil {
		return "", "", err
	}
	url := fmt.Sprintf("https://github.com/astral-sh/uv/releases/download/%s/uv-%s.tar.gz", managedUVVersion, target)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("download uv: %s", resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(uvPath), 0o755); err != nil {
		return "", "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(uvPath), "uv-*")
	if err != nil {
		return "", "", err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		tmp.Close()
		return "", "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	found := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			tmp.Close()
			return "", "", err
		}
		if hdr.Typeflag != tar.TypeReg || filepath.Base(hdr.Name) != "uv" {
			continue
		}
		if _, err := io.Copy(tmp, tr); err != nil {
			tmp.Close()
			return "", "", err
		}
		found = true
		break
	}
	if err := tmp.Close(); err != nil {
		return "", "", err
	}
	if !found {
		return "", "", fmt.Errorf("uv binary not found in %s", url)
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return "", "", err
	}
	if err := os.Rename(tmpPath, uvPath); err != nil {
		return "", "", err
	}
	return uvPath, fmt.Sprintf("Downloaded managed uv %s to %s\n", managedUVVersion, uvPath), nil
}

func (v *BootstrapView) verifyConnCmd() tea.Cmd {
	cfg := v.buildConfig()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client, embed := hindsight.NewFromConfig(cfg)
		if cfg.Backend == config.BackendEmbed && embed != nil {
			if err := embed.CheckInstalled(ctx); err != nil {
				return verifyResultMsg{OK: false, Err: err}
			}
		}
		_, err := client.Health(ctx)
		if err != nil {
			return verifyResultMsg{OK: false, Err: err}
		}
		return verifyResultMsg{OK: true}
	}
}

func installEmbedArgs(installer string) ([]string, error) {
	switch installer {
	case "uv":
		return []string{"tool", "install", "hindsight-embed"}, nil
	case "pipx":
		return []string{"install", "hindsight-embed"}, nil
	case "pip":
		return []string{"install", "--user", "hindsight-embed"}, nil
	default:
		return nil, fmt.Errorf("unknown installer: %s", installer)
	}
}

func installEmbedCmd(installer string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		command := installer
		outputPrefix := ""
		var args []string
		var env []string
		if installer == "managed-uv" {
			uvPath, output, err := ensureManagedUV(ctx)
			outputPrefix = output
			if err != nil {
				return installResultMsg{Output: outputPrefix, Err: err}
			}
			binDir, err := config.ManagedBinDir()
			if err != nil {
				return installResultMsg{Output: outputPrefix, Err: err}
			}
			toolDir, err := config.DataPath("uv", "tools")
			if err != nil {
				return installResultMsg{Output: outputPrefix, Err: err}
			}
			pythonDir, err := config.DataPath("uv", "python")
			if err != nil {
				return installResultMsg{Output: outputPrefix, Err: err}
			}
			cacheDir, err := config.DataPath("uv", "cache")
			if err != nil {
				return installResultMsg{Output: outputPrefix, Err: err}
			}
			command = uvPath
			args = []string{"tool", "install", "hindsight-embed"}
			env = append(os.Environ(),
				"UV_TOOL_BIN_DIR="+binDir,
				"UV_TOOL_DIR="+toolDir,
				"UV_PYTHON_INSTALL_DIR="+pythonDir,
				"UV_CACHE_DIR="+cacheDir,
			)
		} else {
			var err error
			args, err = installEmbedArgs(installer)
			if err != nil {
				return installResultMsg{Err: err}
			}
		}

		cmd := exec.CommandContext(ctx, command, args...)
		if env != nil {
			cmd.Env = env
		}
		output, err := cmd.CombinedOutput()
		return installResultMsg{Output: outputPrefix + string(output), Err: err}
	}
}

func (v *BootstrapView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil
	case configSaveMsg:
		v.loading = false
		if msg.Err != nil {
			return v, nil
		}
		*v.shared.Config = msg.Config
		v.shared.Client = msg.Client
		v.shared.Embed = msg.Embed
		v.shared.State.Backend = msg.Config.Backend
		v.shared.State.SetupActive = false
		return v, func() tea.Msg { return NavigateMsg{Route: state.RouteDashboard} }
	case installResultMsg:
		v.installRunning = false
		v.installDone = true
		v.installOutput = msg.Output
		if msg.Err != nil {
			v.installOK = false
		} else if path, ok := findEmbedBinary(); ok {
			v.installOK = true
			v.embedInstalled = true
			v.embedBinaryPath = path
		} else {
			v.installOK = false
			v.installOutput += "\nInstall completed, but hindsight-embed was not found on PATH or in the Hindsight TUI managed bin directory."
		}
		return v, nil
	case verifyResultMsg:
		v.loading = false
		v.verifyOK = msg.OK
		if msg.Err != nil {
			v.verifyErr = msg.Err.Error()
		}
		return v, nil
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *BootstrapView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.editor.active {
		switch {
		case key.Matches(msg, v.shared.KeyMap.Select):
			value := v.editor.Value()
			v.editor.Stop()
			switch v.step {
			case bootstrapStepAPIURL:
				v.apiURL = value
			case bootstrapStepAuthToken:
				v.authToken = value
			case bootstrapStepBank:
				v.bank = value
			}
			return v, v.advanceStep()
		case msg.Key().Code == tea.KeyEscape:
			v.editor.Stop()
			return v, nil
		}
		return v, v.editor.Update(msg)
	}

	switch v.step {
	case bootstrapStepWelcome:
		switch {
		case key.Matches(msg, v.shared.KeyMap.Select):
			return v, v.advanceStep()
		case msg.Key().Code == tea.KeyEscape:
			return v, tea.Quit
		}
	case bootstrapStepBackend:
		switch {
		case key.Matches(msg, v.shared.KeyMap.Up):
			v.backendIdx = moveIndex(v.backendIdx, -1, len(bootstrapBackends))
			return v, nil
		case key.Matches(msg, v.shared.KeyMap.Down):
			v.backendIdx = moveIndex(v.backendIdx, 1, len(bootstrapBackends))
			return v, nil
		case key.Matches(msg, v.shared.KeyMap.Select):
			return v, v.advanceStep()
		case msg.Key().Code == tea.KeyEscape:
			v.retreatStep()
			return v, nil
		}
	case bootstrapStepInstall:
		return v.handleInstallKey(msg)
	case bootstrapStepAPIURL, bootstrapStepAuthToken, bootstrapStepBank:
		switch {
		case key.Matches(msg, v.shared.KeyMap.Select), key.Matches(msg, v.shared.KeyMap.Up), key.Matches(msg, v.shared.KeyMap.Down):
			var val string
			switch v.step {
			case bootstrapStepAPIURL:
				val = v.apiURL
			case bootstrapStepAuthToken:
				val = v.authToken
			case bootstrapStepBank:
				val = v.bank
			}
			return v, v.editor.Start(val)
		case msg.Key().Code == tea.KeyEscape:
			v.retreatStep()
			return v, nil
		}
	case bootstrapStepTheme:
		switch {
		case key.Matches(msg, v.shared.KeyMap.Up):
			v.themeIdx = moveIndex(v.themeIdx, -1, len(bootstrapThemes))
			return v, nil
		case key.Matches(msg, v.shared.KeyMap.Down):
			v.themeIdx = moveIndex(v.themeIdx, 1, len(bootstrapThemes))
			return v, nil
		case key.Matches(msg, v.shared.KeyMap.Select):
			return v, v.advanceStep()
		case msg.Key().Code == tea.KeyEscape:
			v.retreatStep()
			return v, nil
		}
	case bootstrapStepReview:
		switch {
		case key.Matches(msg, v.shared.KeyMap.Up):
			v.reviewIdx = moveIndex(v.reviewIdx, -1, len(bootstrapReviewActions))
			return v, nil
		case key.Matches(msg, v.shared.KeyMap.Down):
			v.reviewIdx = moveIndex(v.reviewIdx, 1, len(bootstrapReviewActions))
			return v, nil
		case key.Matches(msg, v.shared.KeyMap.Select):
			if bootstrapReviewActions[v.reviewIdx].value == "save" {
				v.loading = true
				return v, saveConfigCmd(v.buildConfig())
			}
			v.retreatStep()
			return v, nil
		case msg.Key().Code == tea.KeyEscape:
			v.retreatStep()
			return v, nil
		}
	}
	return v, nil
}

func (v *BootstrapView) handleInstallKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.installRunning {
		return v, nil
	}
	if v.installDone && v.installOK {
		switch {
		case key.Matches(msg, v.shared.KeyMap.Select):
			return v, v.advanceStep()
		case msg.Key().Code == tea.KeyEscape:
			v.retreatStep()
			return v, nil
		}
		return v, nil
	}
	switch {
	case key.Matches(msg, v.shared.KeyMap.Up):
		v.installerIdx = moveIndex(v.installerIdx, -1, len(v.installers))
		return v, nil
	case key.Matches(msg, v.shared.KeyMap.Down):
		v.installerIdx = moveIndex(v.installerIdx, 1, len(v.installers))
		return v, nil
	case key.Matches(msg, v.shared.KeyMap.Select):
		selected := v.installers[v.installerIdx].value
		if selected == "skip" {
			return v, v.advanceStep()
		}
		v.installRunning = true
		v.installOutput = ""
		return v, installEmbedCmd(selected)
	case msg.Key().Code == tea.KeyEscape:
		v.retreatStep()
		return v, nil
	}
	return v, nil
}

func (v *BootstrapView) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	v.width = width
	v.height = height
	p := v.shared.Palette

	switch v.step {
	case bootstrapStepWelcome:
		return v.viewWelcome(p)
	case bootstrapStepBackend:
		return v.viewBackend(p)
	case bootstrapStepInstall:
		return v.viewInstall(p)
	case bootstrapStepAPIURL:
		return v.viewAPIURL(p)
	case bootstrapStepAuthToken:
		return v.viewAuthToken(p)
	case bootstrapStepBank:
		return v.viewBank(p)
	case bootstrapStepTheme:
		return v.viewTheme(p)
	case bootstrapStepReview:
		return v.viewReview(p)
	}
	return ""
}

func (v *BootstrapView) viewWelcome(p theme.Palette) string {
	lines := []string{
		"",
		p.Primary.Render("Welcome to Hindsight"),
		"",
		p.Muted.Render("This wizard will help you configure your backend,"),
		p.Muted.Render("API connection, and preferences."),
		"",
		p.Muted.Render("Press Enter to begin."),
	}
	return strings.Join(lines, "\n")
}

func (v *BootstrapView) viewBackend(p theme.Palette) string {
	return renderMenu(p, "Backend", bootstrapBackends, v.backendIdx, !v.editor.active)
}

func (v *BootstrapView) viewInstall(p theme.Palette) string {
	if v.installDone && v.installOK {
		path := v.embedBinaryPath
		if path == "" {
			path = "hindsight-embed"
		}
		return p.Panel("Install", fmt.Sprintf(
			"%s %s\n\n%s",
			p.Success.Render("✓"),
			p.Primary.Render("hindsight-embed is installed"),
			p.Muted.Render(path),
		), 0)
	}
	if v.installDone && !v.installOK {
		lines := []string{p.Error.Render("✗ Install failed"), ""}
		if v.installOutput != "" {
			lines = append(lines, p.Muted.Render(v.installOutput))
		}
		lines = append(lines, "", p.Muted.Render("Select an installer to try again, or skip."))
		body := strings.Join(lines, "\n")
		menu := renderMenu(p, "Installer", v.installers, v.installerIdx, true)
		return body + "\n\n" + menu
	}
	if v.installRunning {
		selected := v.installers[v.installerIdx].value
		return p.Panel("Install", fmt.Sprintf(
			"Installing hindsight-embed via %s...\n\n%s",
			p.Primary.Render(selected),
			p.Muted.Render(v.installOutput),
		), 0)
	}
	lines := []string{
		p.Warning.Render("hindsight-embed is not installed."),
		"",
		p.Muted.Render("Select an installer to install hindsight-embed:"),
	}
	body := strings.Join(lines, "\n")
	menu := renderMenu(p, "Installer", v.installers, v.installerIdx, true)
	return body + "\n\n" + menu
}

func (v *BootstrapView) viewAPIURL(p theme.Palette) string {
	if v.editor.active {
		return p.Panel("API URL", renderField(p, "URL", v.editor.View(), true), 0)
	}
	return p.Panel("API URL", renderField(p, "URL", v.apiURL, false), 0)
}

func (v *BootstrapView) viewAuthToken(p theme.Palette) string {
	if v.editor.active {
		return p.Panel("Auth Token", renderField(p, "Token (optional)", v.editor.View(), true), 0)
	}
	display := v.authToken
	if display != "" {
		display = "••••••••"
	}
	hint := p.Muted.Render("Press Enter to edit, or Esc to skip.")
	return p.Panel("Auth Token", renderField(p, "Token", display, false)+"\n"+hint, 0)
}

func (v *BootstrapView) viewBank(p theme.Palette) string {
	if v.editor.active {
		return p.Panel("Default Bank", renderField(p, "Bank", v.editor.View(), true), 0)
	}
	return p.Panel("Default Bank", renderField(p, "Bank", v.bank, false), 0)
}

func (v *BootstrapView) viewTheme(p theme.Palette) string {
	return renderMenu(p, "Theme", bootstrapThemes, v.themeIdx, !v.editor.active)
}

func (v *BootstrapView) viewReview(p theme.Palette) string {
	cfg := v.buildConfig()
	lines := []string{
		p.Primary.Render("Configuration Summary"),
		"",
		renderField(p, "Backend", string(cfg.Backend), false),
		renderField(p, "API URL", cfg.APIURL, false),
	}
	authLabel := "not set"
	if cfg.AuthToken != "" {
		authLabel = "set"
	}
	lines = append(lines, renderField(p, "Auth", authLabel, false))
	lines = append(lines,
		renderField(p, "Bank", cfg.DefaultBank, false),
		renderField(p, "Theme", cfg.Theme, false),
		"",
	)
	if v.loading {
		lines = append(lines, p.Muted.Render("Testing connection..."))
	} else if v.verifyOK {
		lines = append(lines, p.Success.Render("✓ Connection: OK"))
	} else if v.verifyErr != "" {
		lines = append(lines, p.Warning.Render("Connection: ")+p.Muted.Render(v.verifyErr))
	}
	body := strings.Join(lines, "\n")
	menu := renderMenu(p, "Action", bootstrapReviewActions, v.reviewIdx, true)
	return body + "\n\n" + menu
}
