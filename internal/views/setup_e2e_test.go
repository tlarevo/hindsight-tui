package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"

	"hindsight-tui/internal/config"
	"hindsight-tui/internal/hindsight"
	"hindsight-tui/internal/keymap"
	"hindsight-tui/internal/state"
	"hindsight-tui/internal/theme"
)

func setupE2EView(t *testing.T, backend config.Backend) (*BootstrapView, *Shared, string) {
	t.Helper()
	tmpDir := t.TempDir()
	origConfigHome := xdg.ConfigHome
	xdg.ConfigHome = tmpDir
	t.Cleanup(func() { xdg.ConfigHome = origConfigHome })
	cfgPath := filepath.Join(tmpDir, "hindsight-tui", "config.yaml")

	cfg := config.Default()
	cfg.Backend = backend
	pal := theme.Resolve("auto")
	km := keymap.Default()
	shared := &Shared{
		Config: &cfg,
		State: &state.AppState{
			ActiveBank:  cfg.DefaultBank,
			Backend:     cfg.Backend,
			CurrentView: state.RouteDashboard,
			SetupActive: true,
		},
		Client:  hindsight.NewDemoClient(),
		KeyMap:  km,
		Palette: pal,
	}
	return NewBootstrapView(shared), shared, cfgPath
}

func executeSetupSave(t *testing.T, v *BootstrapView) *BootstrapView {
	t.Helper()
	_, cmd := v.Update(bsEnter())
	if cmd == nil {
		t.Fatal("Save produced nil cmd")
	}
	return bsSend(t, v, cmd())
}

func TestSetupE2EDemoFlowSavesConfig(t *testing.T) {
	v, shared, cfgPath := setupE2EView(t, config.BackendDemo)

	v = bsSend(t, v, bsEnter()) // welcome -> backend
	v = bsSend(t, v, bsDown())  // http
	v = bsSend(t, v, bsDown())  // demo
	v = bsSend(t, v, bsEnter()) // demo -> bank, skips install/api/auth
	if v.step != bootstrapStepBank {
		t.Fatalf("after demo select: step=%d, want bank", v.step)
	}

	v = bsSend(t, v, bsEnter()) // edit bank
	v = bsSend(t, v, bsEnter()) // commit default -> theme
	v = bsSend(t, v, bsEnter()) // theme auto -> review
	if v.step != bootstrapStepReview {
		t.Fatalf("after theme: step=%d, want review", v.step)
	}

	v = executeSetupSave(t, v)
	_ = v

	if shared.Config.Backend != config.BackendDemo {
		t.Errorf("shared backend = %s, want demo", shared.Config.Backend)
	}
	if shared.State.SetupActive {
		t.Error("SetupActive should be false after save")
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatalf("config file not created at %s", cfgPath)
	}

	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Backend != config.BackendDemo {
		t.Errorf("saved backend = %s, want demo", loaded.Backend)
	}
	if loaded.DefaultBank != "default" {
		t.Errorf("saved bank = %s, want default", loaded.DefaultBank)
	}
	if loaded.Theme != "auto" {
		t.Errorf("saved theme = %s, want auto", loaded.Theme)
	}

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(raw), "backend: demo") {
		t.Errorf("YAML missing 'backend: demo':\n%s", raw)
	}
	t.Logf("Config saved successfully:\n%s", raw)
}

func TestSetupE2EEmbedFlowSkipsInstall(t *testing.T) {
	v, _, cfgPath := setupE2EView(t, config.BackendEmbed)
	t.Setenv("PATH", "")

	v = bsSend(t, v, bsEnter()) // welcome -> backend
	v = bsSend(t, v, bsEnter()) // embed -> install
	if v.step != bootstrapStepInstall {
		t.Fatalf("embed without binary: step=%d, want install", v.step)
	}

	skipIdx := -1
	for i, item := range v.installers {
		if item.value == "skip" {
			skipIdx = i
			break
		}
	}
	if skipIdx < 0 {
		t.Fatal("no Skip option in install step")
	}
	v.installerIdx = skipIdx
	v = bsSend(t, v, bsEnter()) // skip -> apiURL
	if v.step != bootstrapStepAPIURL {
		t.Fatalf("after skip: step=%d, want apiURL", v.step)
	}

	v = bsSend(t, v, bsEnter()) // edit API URL
	v = bsSend(t, v, bsEnter()) // commit default -> auth
	v = bsSend(t, v, bsEnter()) // edit auth
	v = bsSend(t, v, bsEnter()) // commit empty -> bank
	v = bsSend(t, v, bsEnter()) // edit bank
	v = bsSend(t, v, bsEnter()) // commit default -> theme
	v = bsSend(t, v, bsEnter()) // theme auto -> review
	if v.step != bootstrapStepReview {
		t.Fatalf("after theme: step=%d, want review", v.step)
	}

	v = executeSetupSave(t, v)
	_ = v

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatalf("config file not created at %s", cfgPath)
	}

	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Backend != config.BackendEmbed {
		t.Errorf("backend = %s, want embed", loaded.Backend)
	}
	if loaded.APIURL != "http://127.0.0.1:8888" {
		t.Errorf("api_url = %s, want http://127.0.0.1:8888", loaded.APIURL)
	}
	t.Logf("Embed config saved: backend=%s api_url=%s bank=%s theme=%s", loaded.Backend, loaded.APIURL, loaded.DefaultBank, loaded.Theme)
}
