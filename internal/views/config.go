package views

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/tlarevo/hindsight-tui/internal/config"
	"github.com/tlarevo/hindsight-tui/internal/hindsight"
	"github.com/tlarevo/hindsight-tui/internal/ui"
)

var configEnvKeys = []string{
	"HINDSIGHT_EMBED_LLM_PROVIDER",
	"HINDSIGHT_EMBED_LLM_API_KEY",
	"HINDSIGHT_EMBED_LLM_MODEL",
	"HINDSIGHT_EMBED_BANK_ID",
	"HINDSIGHT_EMBED_DAEMON_IDLE_TIMEOUT",
	"HINDSIGHT_API_LLM_PROVIDER",
	"HINDSIGHT_API_LLM_API_KEY",
	"HINDSIGHT_API_LLM_MODEL",
	"HINDSIGHT_API_DATABASE_URL",
	"HINDSIGHT_API_VECTOR_EXTENSION",
	"HINDSIGHT_CP_DATAPLANE_API_URL",
}

const (
	configPaneForm = iota
	configPaneActions
	configPaneOutput
)

type configField struct {
	label string
	value string
}

type configSaveMsg struct {
	Config config.Config
	Client hindsight.Client
	Embed  *hindsight.EmbedManager
	Path   string
	Err    error
}

type configActionResultMsg struct {
	Title       string
	Body        string
	Err         error
	ClearHealth bool
}

type ConfigView struct {
	shared       *Shared
	width        int
	height       int
	pane         int
	fieldIndex   int
	actionIndex  int
	outputOffset int
	fields       []configField
	editor       fieldEditor
	actions      []simpleListItem
	loading      bool
	statusTitle  string
	statusBody   string
}

func NewConfigView(shared *Shared) *ConfigView {
	view := &ConfigView{
		shared: shared,
		fields: []configField{
			{label: "backend", value: string(shared.Config.Backend)},
			{label: "api_url", value: shared.Config.APIURL},
			{label: "default_bank", value: shared.Config.DefaultBank},
			{label: "theme", value: shared.Config.Theme},
			{label: "compact", value: strconv.FormatBool(shared.Config.Compact)},
			{label: "timeout_ms", value: strconv.Itoa(shared.Config.TimeoutMS)},
		},
		actions: []simpleListItem{
			{title: "Save config", desc: "Write config.yaml and rebuild client", value: "save"},
			{title: "Run doctor", desc: "Show non-interactive diagnostics notes", value: "doctor"},
			{title: "Open embed configure", desc: "Run hindsight-embed configure", value: "configure"},
			{title: "Stop embed daemon", desc: "Run hindsight-embed daemon stop", value: "stop"},
		},
		statusTitle: "Ready",
		statusBody:  "Config is local-first. Save persists ~/.config/hindsight-tui/config.yaml.",
	}
	return view
}

func (v *ConfigView) Title() string { return "Config" }

func (v *ConfigView) TextEntryFocused() bool {
	return v.editor.active
}

func (v *ConfigView) Init() tea.Cmd { return nil }

func (v *ConfigView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch typed := msg.(type) {
	case configSaveMsg:
		v.loading = false
		if typed.Err != nil {
			v.statusTitle = "Save failed"
			v.statusBody = renderFriendlyError(typed.Err)
		} else {
			*v.shared.Config = typed.Config
			v.shared.Client = typed.Client
			v.shared.Embed = typed.Embed
			v.shared.State.Backend = typed.Config.Backend
			v.statusTitle = "Saved"
			v.statusBody = fmt.Sprintf("Saved config to %s. Backend is now %s. Embed health is checked on the next dashboard refresh.", typed.Path, typed.Config.Backend)
			v.resyncFields()
		}
		return v, nil
	case configActionResultMsg:
		v.loading = false
		if typed.Err != nil {
			v.statusTitle = typed.Title
			v.statusBody = renderFriendlyError(typed.Err)
		} else {
			if typed.ClearHealth {
				v.shared.Health = nil
			}
			v.statusTitle = typed.Title
			v.statusBody = typed.Body
		}
		return v, nil
	case tea.KeyPressMsg:
		if v.editor.active {
			return v.updateForm(typed)
		}
		if key.Matches(typed, v.shared.KeyMap.Refresh) {
			v.resyncFields()
			v.statusTitle = "Reloaded"
			v.statusBody = "Form values reloaded from shared config state."
			return v, nil
		}
		if key.Matches(typed, v.shared.KeyMap.Save) {
			cfg, err := v.snapshotConfig()
			if err != nil {
				return v, func() tea.Msg { return configSaveMsg{Err: err} }
			}
			v.loading = true
			return v, saveConfigCmd(cfg)
		}
		if key.Matches(typed, v.shared.KeyMap.NextPane) {
			v.pane = (v.pane + 1) % 3
			return v, nil
		}
		if key.Matches(typed, v.shared.KeyMap.PrevPane) {
			v.pane = (v.pane + 2) % 3
			return v, nil
		}
		switch v.pane {
		case configPaneForm:
			return v.updateForm(typed)
		case configPaneActions:
			return v.updateActions(typed)
		default:
			return v.updateOutput(typed)
		}
	default:
		return v, nil
	}
}

func (v *ConfigView) updateForm(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.editor.active {
		switch {
		case key.Matches(msg, v.shared.KeyMap.Select):
			v.fields[v.fieldIndex].value = v.editor.Value()
			v.editor.Stop()
			return v, nil
		case msg.Key().Code == tea.KeyEscape:
			v.editor.Stop()
			return v, nil
		}
		return v, v.editor.Update(msg)
	}
	if key.Matches(msg, v.shared.KeyMap.Down) {
		v.fieldIndex = moveIndex(v.fieldIndex, 1, len(v.fields))
		return v, nil
	}
	if key.Matches(msg, v.shared.KeyMap.Up) {
		v.fieldIndex = moveIndex(v.fieldIndex, -1, len(v.fields))
		return v, nil
	}
	if key.Matches(msg, v.shared.KeyMap.Select) {
		return v, v.editor.Start(v.fields[v.fieldIndex].value)
	}
	return v, nil
}

func (v *ConfigView) updateActions(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if key.Matches(msg, v.shared.KeyMap.Down) {
		v.actionIndex = moveIndex(v.actionIndex, 1, len(v.actions))
		return v, nil
	}
	if key.Matches(msg, v.shared.KeyMap.Up) {
		v.actionIndex = moveIndex(v.actionIndex, -1, len(v.actions))
		return v, nil
	}
	if key.Matches(msg, v.shared.KeyMap.Select) {
		v.loading = true
		return v, v.actionCmd(v.actions[v.actionIndex].value)
	}
	return v, nil
}

func (v *ConfigView) updateOutput(msg tea.KeyPressMsg) (View, tea.Cmd) {
	lines := strings.Count(v.outputText(), "\n") + 1
	maxOffset := max(0, lines-max(v.height/2, 6))
	if key.Matches(msg, v.shared.KeyMap.Down) {
		v.outputOffset = min(v.outputOffset+1, maxOffset)
	}
	if key.Matches(msg, v.shared.KeyMap.Up) {
		v.outputOffset = max(v.outputOffset-1, 0)
	}
	return v, nil
}

func (v *ConfigView) snapshotConfig() (config.Config, error) {
	cfg := *v.shared.Config
	cfg.Backend = config.Backend(strings.TrimSpace(v.fields[0].value))
	cfg.APIURL = strings.TrimSpace(v.fields[1].value)
	cfg.DefaultBank = strings.TrimSpace(v.fields[2].value)
	cfg.Theme = strings.TrimSpace(v.fields[3].value)
	compactRaw := strings.TrimSpace(v.fields[4].value)
	compact, err := strconv.ParseBool(compactRaw)
	if err != nil {
		return cfg, fmt.Errorf("compact must be true or false (got %q)", compactRaw)
	}
	cfg.Compact = compact
	timeoutRaw := strings.TrimSpace(v.fields[5].value)
	timeout, err := strconv.Atoi(timeoutRaw)
	if err != nil || timeout <= 0 {
		return cfg, fmt.Errorf("timeout_ms must be a positive integer (got %q)", timeoutRaw)
	}
	cfg.TimeoutMS = timeout
	return cfg, nil
}

func saveConfigCmd(cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		if err := validateConfig(cfg); err != nil {
			return configSaveMsg{Err: err}
		}
		path, err := config.ResolvePath("")
		if err != nil {
			return configSaveMsg{Err: err}
		}
		if err := config.Save("", cfg); err != nil {
			return configSaveMsg{Err: err}
		}
		client, embed := hindsight.NewFromConfig(cfg)
		return configSaveMsg{Config: cfg, Client: client, Embed: embed, Path: path}
	}
}

func validateConfig(cfg config.Config) error {
	switch cfg.Backend {
	case config.BackendEmbed, config.BackendHTTP, config.BackendDemo:
	default:
		return fmt.Errorf("invalid backend %q; expected embed, http, or demo", cfg.Backend)
	}
	theme := strings.TrimSpace(cfg.Theme)
	if theme != "auto" && theme != "dark" && theme != "light" {
		return fmt.Errorf("invalid theme %q; expected auto, dark, or light", cfg.Theme)
	}
	if strings.TrimSpace(cfg.APIURL) == "" {
		return fmt.Errorf("api_url is required")
	}
	if strings.TrimSpace(cfg.DefaultBank) == "" {
		return fmt.Errorf("default_bank is required")
	}
	if cfg.TimeoutMS <= 0 {
		return fmt.Errorf("timeout_ms must be greater than zero")
	}
	return nil
}

func (v *ConfigView) actionCmd(action string) tea.Cmd {
	switch action {
	case "save":
		cfg, err := v.snapshotConfig()
		if err != nil {
			return func() tea.Msg { return configSaveMsg{Err: err} }
		}
		return saveConfigCmd(cfg)
	case "doctor":
		return func() tea.Msg {
			return configActionResultMsg{
				Title: "Doctor",
				Body: strings.Join([]string{
					"Run one of:",
					"go run ./cmd/hindsight-tui --doctor",
					"go run ./cmd/hindsight-tui --doctor --backend demo",
					"For embed, confirm hindsight-embed is installed, configured, and can bind 127.0.0.1:8888.",
				}, "\n"),
			}
		}
	case "configure":
		return configEmbedActionCmd(v.shared, "configure")
	case "stop":
		return configEmbedActionCmd(v.shared, "stop")
	default:
		return nil
	}
}

func configEmbedActionCmd(shared *Shared, action string) tea.Cmd {
	backend := shared.Config.Backend
	embed := shared.Embed
	return func() tea.Msg {
		if backend != config.BackendEmbed || embed == nil {
			return configActionResultMsg{Title: "Embed action", Body: "Embed actions are available only when backend=embed."}
		}
		ctx := context.Background()
		switch action {
		case "configure":
			if err := embed.Configure(ctx); err != nil {
				return configActionResultMsg{Title: "Embed configure failed", Err: err}
			}
			return configActionResultMsg{Title: "Embed configure", Body: "hindsight-embed configure completed. Re-run doctor or refresh Dashboard to re-check health."}
		case "stop":
			if err := embed.Stop(ctx); err != nil {
				return configActionResultMsg{Title: "Stop failed", Err: err}
			}
			return configActionResultMsg{Title: "Embed stopped", Body: "Stopped the embed daemon. Dashboard health will show degraded until it starts again.", ClearHealth: true}
		default:
			return configActionResultMsg{Title: "Embed action", Body: "Unknown action."}
		}
	}
}

func (v *ConfigView) resyncFields() {
	v.fields[0].value = string(v.shared.Config.Backend)
	v.fields[1].value = v.shared.Config.APIURL
	v.fields[2].value = v.shared.Config.DefaultBank
	v.fields[3].value = v.shared.Config.Theme
	v.fields[4].value = strconv.FormatBool(v.shared.Config.Compact)
	v.fields[5].value = strconv.Itoa(v.shared.Config.TimeoutMS)
}

func (v *ConfigView) outputText() string {
	parts := []string{fmt.Sprintf("%s\n%s", v.statusTitle, v.statusBody), "Environment"}
	for _, key := range configEnvKeys {
		parts = append(parts, fmt.Sprintf("%s = %s", key, ui.RedactEnvValue(key, os.Getenv(key))))
	}
	return strings.Join(parts, "\n\n")
}

func clippedLines(text string, offset int, limit int) string {
	lines := strings.Split(text, "\n")
	if offset < 0 {
		offset = 0
	}
	if offset > len(lines) {
		offset = len(lines)
	}
	end := len(lines)
	if limit > 0 {
		end = min(offset+limit, len(lines))
	}
	return strings.Join(lines[offset:end], "\n")
}

func (v *ConfigView) View(width, height int) string {
	v.width = width
	v.height = height
	if width <= 0 || height <= 0 {
		return ""
	}
	p := v.shared.Palette
	formLines := make([]string, 0, len(v.fields)+2)
	for i, field := range v.fields {
		focused := v.pane == configPaneForm && i == v.fieldIndex
		value := field.value
		if focused && v.editor.active {
			value = v.editor.View()
		}
		formLines = append(formLines, renderField(p, field.label, value, focused))
	}
	if v.loading {
		formLines = append(formLines, "", p.Spinner.Render("⟳")+" Working...")
	}
	left := p.Panel("Config", strings.Join(formLines, "\n"), 0)
	right := ui.Lines(
		renderMenu(p, "Actions", v.actions, v.actionIndex, v.pane == configPaneActions),
		p.Panel("Status + redacted env", clippedLines(v.outputText(), v.outputOffset, max(height/2, 8)), 0),
	)
	footer := p.Footer.Render(fmt.Sprintf("ctrl+s save · tab switch pane · enter edit/run · esc cancel · backend=%s", v.shared.Config.Backend))
	return ui.Lines(p.Primary.Render("Config"), ui.TwoColumn(left, right, max(width-2, 20)), footer)
}
