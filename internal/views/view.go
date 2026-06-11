package views

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/config"
	"hindsight-tui/internal/domain"
	"hindsight-tui/internal/hindsight"
	"hindsight-tui/internal/keymap"
	"hindsight-tui/internal/state"
)

// moveFocusIn returns the focus value delta steps from current within order,
// wrapping at both ends. Shared by the Recall/Reflect/Retain forms.
func moveFocusIn(order []int, current, delta int) int {
	if len(order) == 0 {
		return current
	}
	index := 0
	for i, focus := range order {
		if focus == current {
			index = i
			break
		}
	}
	index += delta
	if index < 0 {
		index = len(order) - 1
	}
	if index >= len(order) {
		index = 0
	}
	return order[index]
}

// sharedTimeout returns the configured request timeout, defaulting to 30s.
func sharedTimeout(shared *Shared) time.Duration {
	if shared == nil || shared.Config == nil || shared.Config.TimeoutMS <= 0 {
		return 30 * time.Second
	}
	return time.Duration(shared.Config.TimeoutMS) * time.Millisecond
}

type View interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (View, tea.Cmd)
	View(width, height int) string
	Title() string
	TextEntryFocused() bool
}

type Shared struct {
	Config  *config.Config
	State   *state.AppState
	Client  hindsight.Client
	Embed   *hindsight.EmbedManager
	Version *domain.VersionInfo
	Health  *domain.HealthStatus
	KeyMap  keymap.KeyMap
}

type NavigateMsg struct {
	Route state.Route
}

type OpenReflectMsg struct {
	Bank  string
	Query string
}

type OpenRetainMsg struct {
	Bank    string
	Content string
	Context string
	Tags    []string
}

type DashboardLoadedMsg struct {
	Health     *domain.HealthStatus
	Version    *domain.VersionInfo
	Banks      []domain.BankSummary
	Stats      map[string]any
	Operations *domain.Page[map[string]any]
	ActiveBank string
	Err        error
}

type BankProfileLoadedMsg struct {
	Profile *domain.BankProfile
	Stats   map[string]any
	Config  *domain.BankConfig
	Err     error
}

type ExportedFileMsg struct {
	Path string
	Err  error
}
