package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/tlarevo/hindsight-tui/internal/config"
	ownkeymap "github.com/tlarevo/hindsight-tui/internal/keymap"
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
	p := v.shared.Palette
	if v.shared != nil {
		keys = v.shared.KeyMap
		if v.shared.Config != nil && v.shared.Config.Backend != "" {
			backend = string(v.shared.Config.Backend)
		}
	}
	sections := []string{
		p.Panel("Navigation", strings.Join([]string{
			p.Primary.Render("Sidebar") + " is the primary navigation. It's focused when you switch routes.",
			"Use ↑/↓ to move between routes, Enter to switch. Shift+Tab or Esc to leave sidebar.",
			p.Primary.Render("Tab") + " cycles focus between panes within a view.",
			p.Primary.Render("Enter") + " selects or activates the focused element.",
			p.Primary.Render("Esc") + " unfocuses the sidebar, or cancels text entry.",
			p.Primary.Render("?") + " opens this help screen. " + p.Primary.Render("q") + " goes back.",
		}, "\n"), width),
		p.Panel("Core concepts", strings.Join([]string{
			p.Primary.Render("Retain") + " stores new memories in the active bank.",
			p.Primary.Render("Recall") + " searches grounded facts and returns ranked memory results.",
			p.Primary.Render("Reflect") + " asks Hindsight to answer with evidence from stored memories.",
			p.Primary.Render("Banks") + " partition memory by project, team, or workflow.",
			p.Muted.Render("Async retain can index after the write returns; check Operations or retry recall after a short delay."),
			fmt.Sprintf("Backend mode: %s.", backendLabel(config.Backend(backend))),
		}, "\n"), width),
		p.Panel("Global keybindings", strings.Join(globalBindingLines(keys), "\n"), width),
		p.Panel("Help bubble", v.help.FullHelpView(keys.FullHelp()), width),
		p.Panel("Bootstrap commands", strings.Join([]string{
			p.Code.Render("uvx hindsight-embed@latest configure"),
			p.Code.Render("pipx install hindsight-embed"),
			p.Code.Render("hindsight-embed configure"),
			p.Code.Render("pip install hindsight-api"),
			p.Code.Render("hindsight-api"),
			p.Code.Render("npx @vectorize-io/hindsight-control-plane --api-url http://localhost:8888"),
		}, "\n"), width),
		p.Panel("Troubleshooting", strings.Join([]string{
			"If embed will not start, confirm " + p.Code.Render("HINDSIGHT_EMBED_LLM_API_KEY") + " is set.",
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
