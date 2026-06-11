package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/config"
	ownkeymap "hindsight-tui/internal/keymap"
	"hindsight-tui/internal/ui"
)

type HelpView struct {
	shared *Shared
	help   help.Model
}

func NewHelpView(shared *Shared) *HelpView {
	model := help.New()
	return &HelpView{shared: shared, help: model}
}

func (v *HelpView) Init() tea.Cmd {
	return nil
}

func (v *HelpView) Update(msg tea.Msg) (View, tea.Cmd) {
	return v, nil
}

func (v *HelpView) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	v.help.SetWidth(width)
	backend := "embed"
	keys := ownkeymap.Default()
	if v.shared != nil {
		keys = v.shared.KeyMap
		if v.shared.Config != nil && v.shared.Config.Backend != "" {
			backend = string(v.shared.Config.Backend)
		}
	}
	sections := []string{
		ui.Panel("Core concepts", strings.Join([]string{
			"Retain stores new memories in the active bank.",
			"Recall searches grounded facts and returns ranked memory results.",
			"Reflect asks Hindsight to answer with evidence from stored memories.",
			"Banks partition memory by project, team, or workflow.",
			"Async retain can index after the write returns; check Operations or retry recall after a short delay.",
			fmt.Sprintf("Backend mode: embed runs the local hindsight-embed daemon; http talks to an already-running Hindsight API; current default is %s.", backendLabel(config.Backend(backend))),
		}, "\n"), width),
		ui.Panel("Global keybindings", strings.Join(globalBindingLines(keys), "\n"), width),
		ui.Panel("Help bubble", v.help.FullHelpView(keys.FullHelp()), width),
		ui.Panel("Bootstrap commands", strings.Join([]string{
			"uvx hindsight-embed@latest configure",
			"pipx install hindsight-embed",
			"hindsight-embed configure",
			"pip install hindsight-api",
			"hindsight-api",
			"npx @vectorize-io/hindsight-control-plane --api-url http://localhost:8888",
		}, "\n"), width),
		ui.Panel("Troubleshooting", strings.Join([]string{
			"If embed will not start, confirm HINDSIGHT_EMBED_LLM_API_KEY is set.",
			"If the daemon reports bind failures, check for a port 8888 conflict.",
			"If startup or indexing fails early, verify ~/.hindsight/ permissions.",
			"If recall is empty right after async retain, wait for indexing or inspect Operations.",
		}, "\n"), width),
	}
	return strings.Join(sections, "\n")
}

func (v *HelpView) Title() string {
	return "Help"
}

func (v *HelpView) TextEntryFocused() bool {
	return false
}

func globalBindingLines(keys ownkeymap.KeyMap) []string {
	bindings := []key.Binding{
		keys.Quit,
		keys.Back,
		keys.NextPane,
		keys.PrevPane,
		keys.Down,
		keys.Up,
		keys.Select,
		keys.Search,
		keys.Help,
		keys.Refresh,
		keys.Save,
		keys.Banks,
		keys.Recall,
		keys.Retain,
		keys.Reflect,
		keys.Advanced,
		keys.Copy,
		keys.Export,
	}
	lines := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		helper := binding.Help()
		if helper.Key == "" && helper.Desc == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s — %s", helper.Key, helper.Desc))
	}
	return lines
}

func backendLabel(backend config.Backend) string {
	switch backend {
	case config.BackendHTTP:
		return "http"
	case config.BackendDemo:
		return "demo"
	default:
		return "embed"
	}
}
