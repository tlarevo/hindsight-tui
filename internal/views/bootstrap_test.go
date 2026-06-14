package views

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tlarevo/hindsight-tui/internal/config"
	"github.com/tlarevo/hindsight-tui/internal/hindsight"
	"github.com/tlarevo/hindsight-tui/internal/keymap"
	"github.com/tlarevo/hindsight-tui/internal/state"
	"github.com/tlarevo/hindsight-tui/internal/theme"
)

func testShared() *Shared {
	cfg := config.Default()
	return &Shared{
		Config: &cfg,
		State: &state.AppState{
			ActiveBank:  cfg.DefaultBank,
			Backend:     cfg.Backend,
			CurrentView: state.RouteSetup,
		},
		Client:  hindsight.NewDemoClient(),
		KeyMap:  keymap.Default(),
		Palette: theme.Resolve("auto"),
	}
}

func testBootstrapView() *BootstrapView {
	return NewBootstrapView(testShared())
}

func bsEnter() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEnter}
}

func bsDown() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyDown}
}

func bsUp() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyUp}
}

func bsEsc() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEscape}
}

func bsSend(t *testing.T, v *BootstrapView, msg tea.Msg) *BootstrapView {
	t.Helper()
	next, _ := v.Update(msg)
	bv, ok := next.(*BootstrapView)
	if !ok {
		t.Fatalf("Update returned %T, want *BootstrapView", next)
	}
	return bv
}

func bsViewText(t *testing.T, v *BootstrapView) string {
	t.Helper()
	return v.View(80, 24)
}

func TestBootstrapWelcomeStep(t *testing.T) {
	t.Parallel()
	v := testBootstrapView()

	if v.step != bootstrapStepWelcome {
		t.Fatalf("step = %d, want %d (welcome)", v.step, bootstrapStepWelcome)
	}
	body := bsViewText(t, v)
	if !strings.Contains(body, "Welcome to Hindsight") {
		t.Fatalf("welcome text missing, got:\n%s", body)
	}

	// Advance to step 1
	v = bsSend(t, v, bsEnter())
	if v.step != bootstrapStepBackend {
		t.Fatalf("after Enter: step = %d, want %d (backend)", v.step, bootstrapStepBackend)
	}
}

func TestBootstrapBackendSelection(t *testing.T) {
	t.Parallel()
	v := testBootstrapView()
	v = bsSend(t, v, bsEnter()) // → backend

	body := bsViewText(t, v)
	if !strings.Contains(body, "embed") || !strings.Contains(body, "http") || !strings.Contains(body, "demo") {
		t.Fatalf("backend list incomplete:\n%s", body)
	}

	// Move down to "http"
	v = bsSend(t, v, bsDown())
	if v.backendIdx != 1 {
		t.Fatalf("after Down: backendIdx = %d, want 1", v.backendIdx)
	}

	// Select "http" → should skip install, go to API URL (step 3)
	v = bsSend(t, v, bsEnter())
	if v.step != bootstrapStepAPIURL {
		t.Fatalf("after select http: step = %d, want %d (apiURL)", v.step, bootstrapStepAPIURL)
	}
}

func TestBootstrapSkipsInstallForHTTP(t *testing.T) {
	t.Parallel()
	v := testBootstrapView()
	v = bsSend(t, v, bsEnter()) // → backend

	// Select "http" (index 1)
	v = bsSend(t, v, bsDown())
	v = bsSend(t, v, bsEnter())

	if v.step != bootstrapStepAPIURL {
		t.Fatalf("http backend should skip install: step = %d, want %d", v.step, bootstrapStepAPIURL)
	}
}

func TestBootstrapSkipsInstallForDemo(t *testing.T) {
	t.Parallel()
	v := testBootstrapView()
	v = bsSend(t, v, bsEnter()) // → backend

	// Select "demo" (index 2)
	v = bsSend(t, v, bsDown())
	v = bsSend(t, v, bsDown())
	v = bsSend(t, v, bsEnter())

	if v.step != bootstrapStepBank {
		t.Fatalf("demo backend should skip to bank: step = %d, want %d", v.step, bootstrapStepBank)
	}
}

func TestBootstrapAPIURLDefault(t *testing.T) {
	t.Parallel()
	v := testBootstrapView()
	// Advance to backend → select embed → might hit install step
	v = bsSend(t, v, bsEnter()) // → backend
	// embed is index 0, already selected
	v = bsSend(t, v, bsEnter()) // → install or apiURL

	// If we landed on install, skip it
	if v.step == bootstrapStepInstall {
		for i, item := range v.installers {
			if item.value == "skip" {
				v.installerIdx = i
				break
			}
		}
		v = bsSend(t, v, bsEnter()) // → apiURL
	}

	if v.step != bootstrapStepAPIURL {
		t.Fatalf("step = %d, want %d (apiURL)", v.step, bootstrapStepAPIURL)
	}

	if v.apiURL != "http://127.0.0.1:8888" {
		t.Fatalf("default apiURL = %q, want %q", v.apiURL, "http://127.0.0.1:8888")
	}

	body := bsViewText(t, v)
	if !strings.Contains(body, "http://127.0.0.1:8888") {
		t.Fatalf("apiURL not shown:\n%s", body)
	}
}

func TestBootstrapBankDefault(t *testing.T) {
	t.Parallel()
	v := testBootstrapView()
	v.step = bootstrapStepBank

	if v.bank != "default" {
		t.Fatalf("default bank = %q, want %q", v.bank, "default")
	}

	body := bsViewText(t, v)
	if !strings.Contains(body, "default") {
		t.Fatalf("bank not shown:\n%s", body)
	}
}

func TestBootstrapThemeSelection(t *testing.T) {
	t.Parallel()
	v := testBootstrapView()
	v.step = bootstrapStepTheme

	body := bsViewText(t, v)
	if !strings.Contains(body, "auto") || !strings.Contains(body, "dark") || !strings.Contains(body, "light") {
		t.Fatalf("theme list incomplete:\n%s", body)
	}

	// Move down to "dark"
	v = bsSend(t, v, bsDown())
	if v.themeIdx != 1 {
		t.Fatalf("after Down: themeIdx = %d, want 1", v.themeIdx)
	}
}

func TestBootstrapReviewShowsSummary(t *testing.T) {
	t.Parallel()
	v := testBootstrapView()
	v.step = bootstrapStepReview
	v.apiURL = "http://localhost:9999"
	v.bank = "research"
	v.themeIdx = 2 // light

	body := bsViewText(t, v)
	if !strings.Contains(body, "Configuration Summary") {
		t.Fatalf("summary title missing:\n%s", body)
	}
	if !strings.Contains(body, "http://localhost:9999") {
		t.Fatalf("apiURL missing from summary:\n%s", body)
	}
	if !strings.Contains(body, "research") {
		t.Fatalf("bank missing from summary:\n%s", body)
	}
	if !strings.Contains(body, "light") {
		t.Fatalf("theme missing from summary:\n%s", body)
	}
}

func TestBootstrapBackNavigation(t *testing.T) {
	t.Parallel()
	v := testBootstrapView()
	v = bsSend(t, v, bsEnter()) // → backend

	// Go forward then back
	v = bsSend(t, v, bsEnter()) // → install or apiURL
	if v.step == bootstrapStepInstall {
		for i, item := range v.installers {
			if item.value == "skip" {
				v.installerIdx = i
				break
			}
		}
		v = bsSend(t, v, bsEnter()) // → apiURL
	}
	stepBefore := v.step
	v = bsSend(t, v, bsEsc()) // → back
	if v.step == stepBefore {
		t.Fatalf("Esc did not go back: still at step %d", v.step)
	}
}

func TestBootstrapEscapeFromWelcomeQuits(t *testing.T) {
	t.Parallel()
	v := testBootstrapView()

	// Esc from welcome should produce a quit command
	_, cmd := v.Update(bsEsc())
	if cmd == nil {
		t.Fatal("Esc from welcome produced nil cmd (expected tea.Quit)")
	}
}

func TestBootstrapFullDemoFlow(t *testing.T) {
	t.Parallel()
	v := testBootstrapView()

	// Step 0 → 1: Welcome → Backend
	v = bsSend(t, v, bsEnter())
	if v.step != bootstrapStepBackend {
		t.Fatalf("step=%d, want backend", v.step)
	}

	// Step 1: Select "demo" (index 2)
	v = bsSend(t, v, bsDown())
	v = bsSend(t, v, bsDown())
	if v.backendIdx != 2 {
		t.Fatalf("backendIdx=%d, want 2 (demo)", v.backendIdx)
	}

	// Select demo → should jump to bank (step 5), skipping install/apiurl/auth
	v = bsSend(t, v, bsEnter())
	if v.step != bootstrapStepBank {
		t.Fatalf("after demo select: step=%d, want %d (bank)", v.step, bootstrapStepBank)
	}

	// Step 5: Bank — accept default "default"
	if v.bank != "default" {
		t.Fatalf("bank=%q, want %q", v.bank, "default")
	}
	v = bsSend(t, v, bsEnter()) // start editing
	if !v.editor.active {
		t.Fatal("bank step: editor should be active after Enter")
	}
	v = bsSend(t, v, bsEnter()) // commit and advance
	if v.step != bootstrapStepTheme {
		t.Fatalf("after bank: step=%d, want %d (theme)", v.step, bootstrapStepTheme)
	}

	// Step 6: Theme — accept default "auto"
	if v.themeIdx != 0 {
		t.Fatalf("themeIdx=%d, want 0 (auto)", v.themeIdx)
	}
	v = bsSend(t, v, bsEnter())
	if v.step != bootstrapStepReview {
		t.Fatalf("after theme: step=%d, want %d (review)", v.step, bootstrapStepReview)
	}
	body := bsViewText(t, v)
	for _, want := range []string{"demo", "http://127.0.0.1:8888", "default", "auto"} {
		if !strings.Contains(body, want) {
			t.Fatalf("review missing %q:\n%s", want, body)
		}
	}

	// Select "Save & Continue"
	v = bsSend(t, v, bsEnter())
	// The save fires saveConfigCmd (async). The view stays on review.
	// We can't easily await the tea.Cmd, but we verified the review renders correctly.
	_ = v
}

func TestBootstrapInstallStepShownForEmbed(t *testing.T) {
	t.Setenv("PATH", "")
	// Create view with embed backend; hindsight-embed is hidden from PATH.
	shared := testShared()
	shared.Config.Backend = config.BackendEmbed
	v := NewBootstrapView(shared)

	// Welcome → Backend (embed already selected)
	v = bsSend(t, v, bsEnter())
	if v.backendIdx != 0 {
		t.Fatalf("backendIdx=%d, want 0 (embed)", v.backendIdx)
	}

	// Advance from backend → should hit install step (embed, not installed)
	v = bsSend(t, v, bsEnter())
	if v.step != bootstrapStepInstall {
		t.Fatalf("embed without binary: step=%d, want %d (install)", v.step, bootstrapStepInstall)
	}

	body := bsViewText(t, v)
	if !strings.Contains(body, "not installed") && !strings.Contains(body, "Installer") {
		t.Fatalf("install step should show installer list or not-installed:\n%s", body)
	}

	hasSkip := false
	hasManagedUV := false
	for _, item := range v.installers {
		if item.value == "skip" {
			hasSkip = true
		}
		if item.value == "managed-uv" {
			hasManagedUV = true
		}
	}
	if !hasSkip {
		t.Fatal("install step missing 'Skip' option")
	}
	if !hasManagedUV {
		t.Fatalf("install step missing managed uv fallback: %#v", v.installers)
	}

	// Select Skip → should advance to API URL
	for i, item := range v.installers {
		if item.value == "skip" {
			v.installerIdx = i
			break
		}
	}
	v = bsSend(t, v, bsEnter())
	if v.step != bootstrapStepAPIURL {
		t.Fatalf("after skip: step=%d, want %d (apiURL)", v.step, bootstrapStepAPIURL)
	}
}

func TestInstallEmbedArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		installer string
		want      []string
		wantErr   bool
	}{
		{name: "uv", installer: "uv", want: []string{"tool", "install", "hindsight-embed"}},
		{name: "managed uv handled separately", installer: "managed-uv", wantErr: true},
		{name: "pipx", installer: "pipx", want: []string{"install", "hindsight-embed"}},
		{name: "pip", installer: "pip", want: []string{"install", "--user", "hindsight-embed"}},
		{name: "uvx rejected", installer: "uvx", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := installEmbedArgs(tt.installer)
			if tt.wantErr {
				if err == nil {
					t.Fatal("installEmbedArgs() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("installEmbedArgs() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("installEmbedArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
