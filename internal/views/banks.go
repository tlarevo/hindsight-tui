package views

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/domain"
	apperrors "hindsight-tui/internal/errors"
	"hindsight-tui/internal/hindsight"
	"hindsight-tui/internal/ui"
)

const (
	banksPaneList = iota
	banksPaneForm
	banksPaneActions
	banksPaneOutput
)

const (
	bankFieldID = iota
	bankFieldName
	bankFieldReflectMission
	bankFieldRetainMission
	bankFieldExtractionMode
	bankFieldCustomInstructions
	bankFieldEnableObservations
	bankFieldDeleteConfirm
	bankFieldExportPath
	bankFieldImportPath
)

type bankField struct {
	label string
	value string
}

type banksListLoadedMsg struct {
	Banks    []domain.BankSummary
	Selected string
	Err      error
}

type bankDetailsLoadedMsg struct {
	BankID            string
	Profile           *domain.BankProfile
	Stats             map[string]any
	StatsAvailable    bool
	Config            *domain.BankConfig
	ConfigAvailable   bool
	ConfigUnavailable string
	Err               error
	StatsErr          error
}

type banksMutationMsg struct {
	Title             string
	Body              string
	Banks             []domain.BankSummary
	Selected          string
	ActiveBank        string
	Profile           *domain.BankProfile
	Stats             map[string]any
	StatsAvailable    bool
	Config            *domain.BankConfig
	ConfigAvailable   bool
	ConfigUnavailable string
	Err               error
	StatsErr          error
}

type BanksView struct {
	shared            *Shared
	width             int
	height            int
	pane              int
	fieldIndex        int
	bankIndex         int
	actionIndex       int
	outputOffset      int
	loading           bool
	statusTitle       string
	statusBody        string
	banks             []domain.BankSummary
	bankItems         []simpleListItem
	actions           []simpleListItem
	selectedBank      string
	profile           *domain.BankProfile
	stats             map[string]any
	statsAvailable    bool
	bankConfig        *domain.BankConfig
	configAvailable   bool
	configUnavailable string
	statsErr          error
	fields            []bankField
	editor            fieldEditor
}

func NewBanksView(shared *Shared) *BanksView {
	view := &BanksView{
		shared: shared,
		actions: []simpleListItem{
			{title: "Use selected bank", desc: "Set active bank", value: "select"},
			{title: "Create bank", desc: "PUT form bank_id", value: "create"},
			{title: "Update selected bank", desc: "PATCH selected bank", value: "update"},
			{title: "Delete selected bank", desc: "Requires typed confirmation", value: "delete"},
			{title: "Export template", desc: "Write raw JSON to export_path", value: "export"},
			{title: "Import template", desc: "POST import?dry_run=false", value: "import"},
			{title: "Refresh", desc: "Reload bank list and details", value: "refresh"},
		},
		statusTitle: "Ready",
		statusBody:  "Select a bank, edit fields, then choose an action.",
		fields: []bankField{
			{label: "bank_id", value: shared.State.ActiveBank},
			{label: "name", value: ""},
			{label: "reflect_mission", value: ""},
			{label: "retain_mission (blank = keep current)", value: ""},
			{label: "retain_extraction_mode", value: ""},
			{label: "retain_custom_instructions", value: ""},
			{label: "enable_observations", value: ""},
			{label: "delete_confirm", value: ""},
			{label: "export_path", value: ""},
			{label: "Import file path", value: ""},
		},
	}
	return view
}

func (v *BanksView) Title() string { return "Banks" }

func (v *BanksView) TextEntryFocused() bool {
	return v.editor.active
}

func (v *BanksView) Init() tea.Cmd {
	v.loading = true
	return loadBanksListCmd(v.snapshot(), v.shared.State.ActiveBank)
}

// banksSnapshot freezes the Shared values a banks command needs, so the command
// goroutine never reads Shared while the Update loop mutates it.
type banksSnapshot struct {
	client      hindsight.Client
	version     *domain.VersionInfo
	defaultBank string
	activeBank  string
	timeout     time.Duration
}

func (v *BanksView) snapshot() banksSnapshot {
	return banksSnapshot{
		client:      v.shared.Client,
		version:     v.shared.Version,
		defaultBank: v.shared.Config.DefaultBank,
		activeBank:  v.shared.State.ActiveBank,
		timeout:     sharedTimeout(v.shared),
	}
}

func loadBanksListCmd(snap banksSnapshot, preferred string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), snap.timeout)
		defer cancel()
		banks, err := snap.client.ListBanks(ctx)
		if err != nil {
			return banksListLoadedMsg{Err: err}
		}
		selected := activeBankFromListShared(preferred, snap.defaultBank, banks)
		return banksListLoadedMsg{Banks: banks, Selected: selected}
	}
}

func loadBankDetailsCmd(snap banksSnapshot, bankID string) tea.Cmd {
	return func() tea.Msg {
		msg := bankDetailsLoadedMsg{BankID: bankID}
		if strings.TrimSpace(bankID) == "" {
			return msg
		}
		ctx, cancel := context.WithTimeout(context.Background(), snap.timeout)
		defer cancel()
		profile, err := snap.client.GetBank(ctx, bankID)
		if err != nil {
			msg.Err = err
			return msg
		}
		msg.Profile = profile
		stats, err := snap.client.BankStats(ctx, bankID)
		if err == nil {
			msg.Stats = stats
			msg.StatsAvailable = true
		} else {
			msg.StatsErr = err
		}
		if snap.version != nil && snap.version.Features.BankConfigAPI {
			cfg, err := snap.client.GetBankConfig(ctx, bankID)
			if err == nil {
				msg.Config = cfg
				msg.ConfigAvailable = true
			} else {
				msg.ConfigUnavailable = apperrors.Friendly(err).Title
			}
		}
		return msg
	}
}

func (v *BanksView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch typed := msg.(type) {
	case banksListLoadedMsg:
		v.loading = false
		if typed.Err != nil {
			v.statusTitle = "Load failed"
			v.statusBody = renderFriendlyError(typed.Err)
			return v, nil
		}
		v.banks = typed.Banks
		v.bankItems = listItemsFromBanks(typed.Banks)
		v.selectedBank = typed.Selected
		v.bankIndex = indexForBank(v.bankItems, typed.Selected)
		if len(typed.Banks) > 0 {
			v.shared.State.ActiveBank = activeBankFromListShared(v.shared.State.ActiveBank, v.shared.Config.DefaultBank, typed.Banks)
		}
		if typed.Selected == "" {
			return v, nil
		}
		v.loading = true
		return v, loadBankDetailsCmd(v.snapshot(), typed.Selected)
	case bankDetailsLoadedMsg:
		v.loading = false
		if typed.Err != nil {
			v.statusTitle = "Bank details failed"
			v.statusBody = renderFriendlyError(typed.Err)
			return v, nil
		}
		v.selectedBank = typed.BankID
		v.profile = typed.Profile
		v.stats = typed.Stats
		v.statsAvailable = typed.StatsAvailable
		v.statsErr = typed.StatsErr
		v.bankConfig = typed.Config
		v.configAvailable = typed.ConfigAvailable
		v.configUnavailable = typed.ConfigUnavailable
		v.syncFormFromLoadedBank()
		return v, nil
	case banksMutationMsg:
		v.loading = false
		if typed.Err != nil {
			v.statusTitle = typed.Title
			v.statusBody = renderFriendlyError(typed.Err)
			return v, nil
		}
		v.statusTitle = typed.Title
		v.statusBody = typed.Body
		if typed.Banks != nil {
			v.banks = typed.Banks
			v.bankItems = listItemsFromBanks(typed.Banks)
		}
		if typed.Selected != "" || len(v.bankItems) == 0 {
			v.selectedBank = typed.Selected
			v.bankIndex = indexForBank(v.bankItems, typed.Selected)
		}
		if typed.ActiveBank != "" {
			v.shared.State.ActiveBank = typed.ActiveBank
		}
		if typed.Profile != nil || typed.Selected == "" {
			v.profile = typed.Profile
			v.stats = typed.Stats
			v.statsAvailable = typed.StatsAvailable
			v.statsErr = typed.StatsErr
			v.bankConfig = typed.Config
			v.configAvailable = typed.ConfigAvailable
			v.configUnavailable = typed.ConfigUnavailable
			v.syncFormFromLoadedBank()
		}
		return v, nil
	case tea.KeyPressMsg:
		if v.editor.active {
			return v.updateForm(typed)
		}
		if key.Matches(typed, v.shared.KeyMap.Refresh) {
			v.loading = true
			return v, loadBanksListCmd(v.snapshot(), v.selectedBank)
		}
		if key.Matches(typed, v.shared.KeyMap.NextPane) {
			v.pane = (v.pane + 1) % 4
			return v, nil
		}
		if key.Matches(typed, v.shared.KeyMap.PrevPane) {
			v.pane = (v.pane + 3) % 4
			return v, nil
		}
		switch v.pane {
		case banksPaneList:
			return v.updateBankList(typed)
		case banksPaneForm:
			return v.updateForm(typed)
		case banksPaneActions:
			return v.updateActions(typed)
		default:
			return v.updateOutput(typed)
		}
	default:
		return v, nil
	}
}

func (v *BanksView) updateBankList(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if key.Matches(msg, v.shared.KeyMap.Down) {
		v.bankIndex = moveIndex(v.bankIndex, 1, len(v.bankItems))
		return v, v.loadCurrentBankDetails()
	}
	if key.Matches(msg, v.shared.KeyMap.Up) {
		v.bankIndex = moveIndex(v.bankIndex, -1, len(v.bankItems))
		return v, v.loadCurrentBankDetails()
	}
	if key.Matches(msg, v.shared.KeyMap.Select) {
		selected := v.currentBankID()
		if selected != "" {
			v.shared.State.ActiveBank = selected
			v.statusTitle = "Active bank"
			v.statusBody = fmt.Sprintf("Active bank set to %s.", selected)
		}
		return v, nil
	}
	return v, nil
}

func (v *BanksView) loadCurrentBankDetails() tea.Cmd {
	selected := v.currentBankID()
	if selected == "" || selected == v.selectedBank {
		return nil
	}
	v.selectedBank = selected
	v.loading = true
	return loadBankDetailsCmd(v.snapshot(), selected)
}

func (v *BanksView) updateForm(msg tea.KeyPressMsg) (View, tea.Cmd) {
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

func (v *BanksView) updateActions(msg tea.KeyPressMsg) (View, tea.Cmd) {
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

func (v *BanksView) updateOutput(msg tea.KeyPressMsg) (View, tea.Cmd) {
	lines := strings.Count(v.outputText(), "\n") + 1
	maxOffset := max(0, lines-max(v.height/4, 6))
	if key.Matches(msg, v.shared.KeyMap.Down) {
		v.outputOffset = min(v.outputOffset+1, maxOffset)
	}
	if key.Matches(msg, v.shared.KeyMap.Up) {
		v.outputOffset = max(v.outputOffset-1, 0)
	}
	return v, nil
}

func (v *BanksView) currentBankID() string {
	if v.bankIndex < 0 || v.bankIndex >= len(v.bankItems) {
		return ""
	}
	return v.bankItems[v.bankIndex].value
}

func indexForBank(items []simpleListItem, bankID string) int {
	for i, item := range items {
		if item.value == bankID {
			return i
		}
	}
	return 0
}

func (v *BanksView) actionCmd(action string) tea.Cmd {
	switch action {
	case "select":
		selected := v.currentBankID()
		return func() tea.Msg {
			if strings.TrimSpace(selected) == "" {
				return banksMutationMsg{Title: "Select bank", Body: "No bank is selected."}
			}
			return banksMutationMsg{Title: "Active bank", Body: fmt.Sprintf("Active bank set to %s.", selected), ActiveBank: selected}
		}
	case "create":
		return createBankCmd(v.snapshot(), v.formBankID(), v.createBankRequest())
	case "update":
		return updateBankCmd(v.snapshot(), v.selectedBank, v.createBankRequest())
	case "delete":
		return deleteBankCmd(v.snapshot(), v.selectedBank, strings.TrimSpace(v.fields[bankFieldDeleteConfirm].value))
	case "export":
		return exportBankCmd(v.snapshot(), v.selectedBank, strings.TrimSpace(v.fields[bankFieldExportPath].value))
	case "import":
		return importBankCmd(v.snapshot(), v.selectedBank, strings.TrimSpace(v.fields[bankFieldImportPath].value))
	case "refresh":
		return loadBanksListCmd(v.snapshot(), v.selectedBank)
	default:
		return nil
	}
}

func (v *BanksView) formBankID() string {
	return strings.TrimSpace(v.fields[bankFieldID].value)
}

func (v *BanksView) createBankRequest() domain.CreateBankRequest {
	name := strings.TrimSpace(v.fields[bankFieldName].value)
	reflectMission := strings.TrimSpace(v.fields[bankFieldReflectMission].value)
	retainMission := strings.TrimSpace(v.fields[bankFieldRetainMission].value)
	extractionMode := strings.TrimSpace(v.fields[bankFieldExtractionMode].value)
	customInstructions := strings.TrimSpace(v.fields[bankFieldCustomInstructions].value)
	enableRaw := strings.TrimSpace(v.fields[bankFieldEnableObservations].value)
	req := domain.CreateBankRequest{}
	if name != "" {
		req.Name = stringPtr(name)
	}
	if reflectMission != "" {
		req.ReflectMission = stringPtr(reflectMission)
	}
	if retainMission != "" {
		req.RetainMission = stringPtr(retainMission)
	}
	if extractionMode != "" {
		req.RetainExtractionMode = stringPtr(extractionMode)
	}
	if extractionMode == "custom" && customInstructions != "" {
		req.RetainCustomInstructions = stringPtr(customInstructions)
	}
	if enableRaw != "" {
		if parsed, err := strconv.ParseBool(enableRaw); err == nil {
			req.EnableObservations = &parsed
		}
	}
	return req
}

func validateBankRequest(bankID string, req domain.CreateBankRequest) error {
	if err := ui.ValidateBankID(bankID); err != nil {
		return err
	}
	if req.RetainExtractionMode != nil {
		switch *req.RetainExtractionMode {
		case "", "concise", "verbose", "custom":
		default:
			return fmt.Errorf("retain_extraction_mode must be empty, concise, verbose, or custom")
		}
	}
	if req.RetainExtractionMode != nil && *req.RetainExtractionMode != "custom" && req.RetainCustomInstructions != nil {
		return fmt.Errorf("retain_custom_instructions is only valid when retain_extraction_mode=custom")
	}
	return nil
}

func createBankCmd(snap banksSnapshot, bankID string, req domain.CreateBankRequest) tea.Cmd {
	return func() tea.Msg {
		if err := validateBankRequest(bankID, req); err != nil {
			return banksMutationMsg{Title: "Create bank failed", Err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), snap.timeout)
		defer cancel()
		if _, err := snap.client.CreateOrUpdateBank(ctx, bankID, req); err != nil {
			return banksMutationMsg{Title: "Create bank failed", Err: err}
		}
		return reloadBanksState(snap, bankID, bankID, fmt.Sprintf("Stored bank %s with PUT.", bankID))
	}
}

func updateBankCmd(snap banksSnapshot, selectedBank string, req domain.CreateBankRequest) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(selectedBank) == "" {
			return banksMutationMsg{Title: "Update bank failed", Err: fmt.Errorf("select a bank before updating")}
		}
		if err := validateBankRequest(selectedBank, req); err != nil {
			return banksMutationMsg{Title: "Update bank failed", Err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), snap.timeout)
		defer cancel()
		if _, err := snap.client.PatchBank(ctx, selectedBank, req); err != nil {
			return banksMutationMsg{Title: "Update bank failed", Err: err}
		}
		return reloadBanksState(snap, selectedBank, snap.activeBank, fmt.Sprintf("Patched bank %s.", selectedBank))
	}
}

func deleteBankCmd(snap banksSnapshot, selectedBank string, confirmation string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(selectedBank) == "" {
			return banksMutationMsg{Title: "Delete bank failed", Err: fmt.Errorf("select a bank before deleting")}
		}
		if confirmation != selectedBank {
			return banksMutationMsg{Title: "Delete bank failed", Err: fmt.Errorf("type %s in delete_confirm to confirm deletion", selectedBank)}
		}
		ctx, cancel := context.WithTimeout(context.Background(), snap.timeout)
		defer cancel()
		if err := snap.client.DeleteBank(ctx, selectedBank); err != nil {
			return banksMutationMsg{Title: "Delete bank failed", Err: err}
		}
		banks, err := snap.client.ListBanks(ctx)
		if err != nil {
			return banksMutationMsg{Title: "Delete bank failed", Err: err}
		}
		active := snap.activeBank
		if active == selectedBank {
			active = activeBankFromListShared("", snap.defaultBank, banks)
		}
		selected := activeBankFromListShared(active, snap.defaultBank, banks)
		return collectBanksMutation(snap, banks, selected, active, fmt.Sprintf("Deleted bank %s.", selectedBank))
	}
}

func exportBankCmd(snap banksSnapshot, selectedBank string, path string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(selectedBank) == "" {
			return banksMutationMsg{Title: "Export failed", Err: fmt.Errorf("select a bank before exporting")}
		}
		if strings.TrimSpace(path) == "" {
			return banksMutationMsg{Title: "Export failed", Err: fmt.Errorf("export_path is required")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), snap.timeout)
		defer cancel()
		raw, err := snap.client.ExportBankTemplate(ctx, selectedBank)
		if err != nil {
			if hindsight.IsUnsupported(err) {
				return banksMutationMsg{Title: "Export", Body: "Bank template export is unavailable on this Hindsight API version"}
			}
			return banksMutationMsg{Title: "Export failed", Err: err}
		}
		if err := ui.WritePrivateText(path, raw); err != nil {
			return banksMutationMsg{Title: "Export failed", Err: err}
		}
		return banksMutationMsg{Title: "Export", Body: fmt.Sprintf("Wrote raw bank template to %s.", path)}
	}
}

func importBankCmd(snap banksSnapshot, selectedBank string, path string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(selectedBank) == "" {
			return banksMutationMsg{Title: "Import failed", Err: fmt.Errorf("select a bank before importing")}
		}
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			return banksMutationMsg{Title: "Import failed", Err: fmt.Errorf("enter an import file path in the Import file path field")}
		}
		data, err := os.ReadFile(trimmed)
		if err != nil {
			return banksMutationMsg{Title: "Import failed", Err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), snap.timeout)
		defer cancel()
		raw, err := snap.client.ImportBankTemplate(ctx, selectedBank, json.RawMessage(data), false)
		if err != nil {
			if hindsight.IsUnsupported(err) {
				return banksMutationMsg{Title: "Import", Body: "Bank template import is unavailable on this Hindsight API version"}
			}
			return banksMutationMsg{Title: "Import failed", Err: err}
		}
		return banksMutationMsg{Title: "Import", Body: string(raw)}
	}
}

func reloadBanksState(snap banksSnapshot, selected string, active string, body string) banksMutationMsg {
	ctx, cancel := context.WithTimeout(context.Background(), snap.timeout)
	defer cancel()
	banks, err := snap.client.ListBanks(ctx)
	if err != nil {
		return banksMutationMsg{Title: "Refresh failed", Err: err}
	}
	selected = activeBankFromListShared(selected, snap.defaultBank, banks)
	if strings.TrimSpace(active) == "" || !bankExists(banks, active) {
		active = activeBankFromListShared(snap.activeBank, snap.defaultBank, banks)
	}
	return collectBanksMutation(snap, banks, selected, active, body)
}

func collectBanksMutation(snap banksSnapshot, banks []domain.BankSummary, selected string, active string, body string) banksMutationMsg {
	msg := banksMutationMsg{Title: "Banks", Body: body, Banks: banks, Selected: selected, ActiveBank: active}
	if strings.TrimSpace(selected) == "" {
		return msg
	}
	raw := loadBankDetailsCmd(snap, selected)()
	details, ok := raw.(bankDetailsLoadedMsg)
	if !ok {
		return banksMutationMsg{Title: "Banks", Err: fmt.Errorf("unexpected bank details response")}
	}
	if details.Err != nil {
		msg.Err = details.Err
		return msg
	}
	msg.Profile = details.Profile
	msg.Stats = details.Stats
	msg.StatsAvailable = details.StatsAvailable
	msg.StatsErr = details.StatsErr
	msg.Config = details.Config
	msg.ConfigAvailable = details.ConfigAvailable
	msg.ConfigUnavailable = details.ConfigUnavailable
	return msg
}

func stringPtr(value string) *string { return &value }

func (v *BanksView) syncFormFromLoadedBank() {
	if v.profile == nil {
		return
	}
	v.fields[bankFieldID].value = v.profile.BankID
	v.fields[bankFieldName].value = v.profile.Name
	v.fields[bankFieldReflectMission].value = v.profile.Mission
	// BankProfile exposes a single Mission (the reflect mission); CreateBankRequest
	// sends distinct reflect/retain missions only when non-empty. Pre-filling the
	// retain field from Mission made an untouched save clobber the server's
	// distinct retain mission, so leave it blank to mean "keep current".
	v.fields[bankFieldRetainMission].value = ""
	v.fields[bankFieldExtractionMode].value = ""
	v.fields[bankFieldCustomInstructions].value = ""
	v.fields[bankFieldEnableObservations].value = ""
	v.fields[bankFieldDeleteConfirm].value = ""
	if v.bankConfig != nil {
		if raw, ok := v.bankConfig.Overrides["retain_extraction_mode"]; ok {
			v.fields[bankFieldExtractionMode].value = stringifyAny(raw)
		}
		if raw, ok := v.bankConfig.Overrides["retain_custom_instructions"]; ok {
			v.fields[bankFieldCustomInstructions].value = stringifyAny(raw)
		}
		if raw, ok := v.bankConfig.Overrides["enable_observations"]; ok {
			v.fields[bankFieldEnableObservations].value = stringifyAny(raw)
		}
	}
}

func (v *BanksView) detailBody() string {
	parts := []string{}
	if v.profile != nil {
		parts = append(parts, "Profile", ui.PrettyJSON(v.profile))
	}
	if v.statsAvailable {
		parts = append(parts, "Stats", ui.PrettyJSON(v.stats))
	} else if v.statsErr != nil {
		parts = append(parts, "Stats", apperrors.Friendly(v.statsErr).Title)
	} else {
		parts = append(parts, "Stats", "unavailable")
	}
	if v.shared.Version != nil && v.shared.Version.Features.BankConfigAPI {
		if v.configAvailable {
			parts = append(parts, "Config", ui.PrettyJSON(v.bankConfig))
		} else {
			parts = append(parts, "Config", blankFallback(v.configUnavailable, "unavailable"))
		}
	}
	if len(parts) == 0 {
		return "No bank selected"
	}
	return strings.Join(parts, "\n\n")
}

func (v *BanksView) outputText() string {
	return strings.Join([]string{v.statusTitle, v.statusBody}, "\n\n")
}

func (v *BanksView) View(width, height int) string {
	v.width = width
	v.height = height
	if width <= 0 || height <= 0 {
		return ""
	}
	p := v.shared.Palette
	formLines := make([]string, 0, len(v.fields)+3)
	for i, field := range v.fields {
		focused := v.pane == banksPaneForm && i == v.fieldIndex
		value := field.value
		if focused && v.editor.active {
			value = v.editor.View()
		}
		formLines = append(formLines, renderField(p, field.label, value, focused))
	}
	if v.shared.Version == nil || !v.shared.Version.Features.Observations {
		formLines = append(formLines, "", p.Muted.Render("enable_observations is ignored until the server reports observations support."))
	}
	if v.loading {
		formLines = append(formLines, "", p.Spinner.Render("⟳")+" Working...")
	}
	left := renderMenu(p, "Banks", v.bankItems, v.bankIndex, v.pane == banksPaneList)
	right := ui.Lines(
		p.Panel("Selected bank", v.detailBody(), 0),
		p.Panel("Create / edit", strings.Join(formLines, "\n"), 0),
		renderMenu(p, "Actions", v.actions, v.actionIndex, v.pane == banksPaneActions),
		p.Panel("Output", clippedLines(v.outputText(), v.outputOffset, max(height/4, 6)), 0),
	)
	footer := p.Footer.Render(fmt.Sprintf("selected=%s active=%s | enter edit · esc cancel", blankFallback(v.selectedBank, "none"), blankFallback(v.shared.State.ActiveBank, v.shared.Config.DefaultBank)))
	return ui.Lines(p.Primary.Render("Banks"), ui.TwoColumn(left, right, max(width-2, 20)), footer)
}
