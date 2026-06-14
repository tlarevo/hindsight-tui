package views

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/domain"
)

// --- Pure functions ---

func TestValidateBankRequestValidEmptyMode(t *testing.T) {
	t.Parallel()
	if err := validateBankRequest("my-bank", domain.CreateBankRequest{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateBankRequestInvalidID(t *testing.T) {
	t.Parallel()
	err := validateBankRequest("", domain.CreateBankRequest{})
	if err == nil {
		t.Fatal("expected error for empty bank ID")
	}
	if !strings.Contains(err.Error(), "bank_id is required") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestValidateBankRequestBogusExtractionMode(t *testing.T) {
	t.Parallel()
	mode := "bogus"
	err := validateBankRequest("ok-bank", domain.CreateBankRequest{
		RetainExtractionMode: &mode,
	})
	if err == nil {
		t.Fatal("expected error for bogus extraction mode")
	}
	if !strings.Contains(err.Error(), "retain_extraction_mode") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestValidateBankRequestCustomModeOptionalInstructions(t *testing.T) {
	t.Parallel()
	mode := "custom"
	if err := validateBankRequest("ok-bank", domain.CreateBankRequest{
		RetainExtractionMode: &mode,
	}); err != nil {
		t.Fatalf("unexpected error for custom mode without instructions: %v", err)
	}
}

func TestValidateBankRequestNonCustomModeWithInstructions(t *testing.T) {
	t.Parallel()
	mode := "concise"
	instructions := "some instructions"
	err := validateBankRequest("ok-bank", domain.CreateBankRequest{
		RetainExtractionMode:     &mode,
		RetainCustomInstructions: &instructions,
	})
	if err == nil {
		t.Fatal("expected error: non-custom mode with custom instructions")
	}
	if !strings.Contains(err.Error(), "only valid when retain_extraction_mode=custom") {
		t.Fatalf("error = %q", err.Error())
	}
}

// --- indexForBank ---

func TestIndexForBankFound(t *testing.T) {
	t.Parallel()
	items := []simpleListItem{
		{title: "a", value: "a"},
		{title: "b", value: "b"},
		{title: "c", value: "c"},
	}
	got := indexForBank(items, "b")
	if got != 1 {
		t.Fatalf("indexForBank = %d, want 1", got)
	}
}

func TestIndexForBankNotFound(t *testing.T) {
	t.Parallel()
	items := []simpleListItem{
		{title: "a", value: "a"},
	}
	got := indexForBank(items, "missing")
	if got != 0 {
		t.Fatalf("indexForBank = %d, want 0", got)
	}
}

func TestIndexForBankEmpty(t *testing.T) {
	t.Parallel()
	got := indexForBank(nil, "any")
	if got != 0 {
		t.Fatalf("indexForBank = %d, want 0", got)
	}
}

// --- createBankRequest ---

func TestCreateBankRequestBlankFields(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)
	req := view.createBankRequest()
	if req.Name != nil {
		t.Errorf("Name = %v, want nil", req.Name)
	}
	if req.ReflectMission != nil {
		t.Errorf("ReflectMission = %v, want nil", req.ReflectMission)
	}
	if req.RetainMission != nil {
		t.Errorf("RetainMission = %v, want nil", req.RetainMission)
	}
}

func TestCreateBankRequestPopulatesFields(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)

	view.fields[bankFieldName].value = "My Bank"
	view.fields[bankFieldReflectMission].value = "Think hard"
	view.fields[bankFieldRetainMission].value = "Remember stuff"
	view.fields[bankFieldExtractionMode].value = "concise"
	view.fields[bankFieldEnableObservations].value = "true"

	req := view.createBankRequest()
	if req.Name == nil || *req.Name != "My Bank" {
		t.Errorf("Name = %v, want 'My Bank'", req.Name)
	}
	if req.ReflectMission == nil || *req.ReflectMission != "Think hard" {
		t.Errorf("ReflectMission = %v, want 'Think hard'", req.ReflectMission)
	}
	if req.RetainMission == nil || *req.RetainMission != "Remember stuff" {
		t.Errorf("RetainMission = %v, want 'Remember stuff'", req.RetainMission)
	}
	if req.RetainExtractionMode == nil || *req.RetainExtractionMode != "concise" {
		t.Errorf("RetainExtractionMode = %v, want 'concise'", req.RetainExtractionMode)
	}
	if req.EnableObservations == nil || !*req.EnableObservations {
		t.Errorf("EnableObservations = %v, want true", req.EnableObservations)
	}
}

func TestCreateBankRequestCustomModePopulatesInstructions(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)

	view.fields[bankFieldExtractionMode].value = "custom"
	view.fields[bankFieldCustomInstructions].value = "do special things"

	req := view.createBankRequest()
	if req.RetainCustomInstructions == nil || *req.RetainCustomInstructions != "do special things" {
		t.Errorf("RetainCustomInstructions = %v, want 'do special things'", req.RetainCustomInstructions)
	}
}

func TestCreateBankRequestFalseEnableObservations(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)

	view.fields[bankFieldEnableObservations].value = "false"
	req := view.createBankRequest()
	if req.EnableObservations == nil || *req.EnableObservations {
		t.Errorf("EnableObservations = %v, want false", req.EnableObservations)
	}
}

// --- State transitions via Update ---

func TestBanksListLoadedWithError(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)

	updated, _ := view.Update(banksListLoadedMsg{Err: fmt.Errorf("network error")})
	v := updated.(*BanksView)
	if v.statusTitle != "Load failed" {
		t.Errorf("statusTitle = %q, want 'Load failed'", v.statusTitle)
	}
}

func TestBanksListLoadedWithBanks(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)

	banks := []domain.BankSummary{
		{BankID: "alpha", Name: stringPtr("Alpha")},
		{BankID: "beta", Name: stringPtr("Beta")},
	}
	updated, _ := view.Update(banksListLoadedMsg{Banks: banks, Selected: "beta"})
	v := updated.(*BanksView)

	if len(v.banks) != 2 {
		t.Fatalf("banks len = %d, want 2", len(v.banks))
	}
	if v.selectedBank != "beta" {
		t.Errorf("selectedBank = %q, want 'beta'", v.selectedBank)
	}
}

func TestBankDetailsLoadedSyncsView(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)

	stats := map[string]any{"memory_count": float64(42)}
	profile := &domain.BankProfile{BankID: "alpha"}
	updated, _ := view.Update(bankDetailsLoadedMsg{
		BankID:  "alpha",
		Profile: profile,
		Stats:   stats,
	})
	v := updated.(*BanksView)

	if v.profile != profile {
		t.Error("profile not synced")
	}
	if v.stats["memory_count"] != float64(42) {
		t.Errorf("stats = %v", v.stats)
	}
}

func TestBanksMutationWithError(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)

	updated, _ := view.Update(banksMutationMsg{
		Title: "Create bank",
		Err:   fmt.Errorf("conflict"),
	})
	v := updated.(*BanksView)
	if v.statusTitle != "Create bank" {
		t.Errorf("statusTitle = %q, want 'Create bank'", v.statusTitle)
	}
	if !strings.Contains(v.statusBody, "conflict") {
		t.Errorf("statusBody = %q, want contains 'conflict'", v.statusBody)
	}
}

func TestBanksMutationWithActiveBank(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)

	updated, _ := view.Update(banksMutationMsg{
		Title:      "Active bank",
		Body:       "Set.",
		ActiveBank: "gamma",
	})
	v := updated.(*BanksView)
	if shared.State.ActiveBank != "gamma" {
		t.Errorf("ActiveBank = %q, want 'gamma'", shared.State.ActiveBank)
	}
	if v.statusTitle != "Active bank" {
		t.Errorf("statusTitle = %q", v.statusTitle)
	}
}

func TestBanksListNavigationDown(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)

	banks := []domain.BankSummary{
		{BankID: "a"},
		{BankID: "b"},
	}
	updated, _ := view.Update(banksListLoadedMsg{Banks: banks, Selected: "a"})
	v := updated.(*BanksView)

	// Down key
	updated, _ = v.Update(keyPress("j", 'j'))
	v = updated.(*BanksView)
	if v.bankIndex != 1 {
		t.Errorf("bankIndex = %d, want 1 after down", v.bankIndex)
	}
}

func TestBanksSelectSetsActiveBank(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)

	banks := []domain.BankSummary{
		{BankID: "a"},
		{BankID: "b"},
	}
	updated, _ := view.Update(banksListLoadedMsg{Banks: banks, Selected: "a"})
	v := updated.(*BanksView)
	v.bankIndex = 1

	// Select key
	updated, _ = v.Update(keyPress("", tea.KeyEnter))
	v = updated.(*BanksView)
	if shared.State.ActiveBank != "b" {
		t.Errorf("ActiveBank = %q, want 'b'", shared.State.ActiveBank)
	}
}

func TestBanksPaneCycling(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)

	// Tab cycles forward through 4 panes
	for i := 0; i < 4; i++ {
		updated, _ := view.Update(keyPress("", tea.KeyTab))
		view = updated.(*BanksView)
	}
	if view.pane != 0 {
		t.Errorf("pane = %d after 4 tabs, want 0", view.pane)
	}

	// Shift+Tab cycles backward
	updated, _ := view.Update(tea.KeyPressMsg(tea.Key{Mod: tea.ModShift, Code: tea.KeyTab}))
	view = updated.(*BanksView)
	if view.pane != 3 {
		t.Errorf("pane = %d after shift-tab, want 3", view.pane)
	}
}

func TestBanksFormEditorStartStop(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewBanksView(shared)
	view.pane = banksPaneForm

	// Enter starts editor
	updated, cmd := view.Update(keyPress("", tea.KeyEnter))
	view = updated.(*BanksView)
	if !view.editor.active {
		t.Fatal("editor should be active after enter")
	}
	if cmd == nil {
		t.Fatal("expected command from editor.Start")
	}
	// Escape stops editor without committing
	updated, _ = view.Update(keyPress("", tea.KeyEscape))
	view = updated.(*BanksView)
	if view.editor.active {
		t.Fatal("editor should be inactive after escape")
	}
}
