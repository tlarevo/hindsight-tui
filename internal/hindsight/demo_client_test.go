package hindsight

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"hindsight-tui/internal/domain"
)
func boolPtr(v bool) *bool { return &v }

// ---------------------------------------------------------------------------
// Step 1 — Seed accessors & metadata
// ---------------------------------------------------------------------------

func TestDemoVersion(t *testing.T) {
	c := NewDemoClient()
	v, err := c.Version(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if v.APIVersion != "0.8.1-demo" {
		t.Errorf("APIVersion = %q, want %q", v.APIVersion, "0.8.1-demo")
	}
	if !v.Features.Observations || !v.Features.BankConfigAPI || !v.Features.DocumentExportAPI || !v.Features.DocumentImportAPI || !v.Features.AuditLog || !v.Features.LLMTrace || !v.Features.StoreDocumentText {
		t.Error("expected feature flags to be true")
	}
	if v.Features.MCP || v.Features.Worker || v.Features.FileUploadAPI {
		t.Error("expected MCP/Worker/FileUploadAPI to be false")
	}
}

func TestDemoHealth(t *testing.T) {
	c := NewDemoClient()
	h, err := c.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !h.OK {
		t.Error("OK should be true")
	}
	if h.Detail != "demo backend ready" {
		t.Errorf("Detail = %q, want %q", h.Detail, "demo backend ready")
	}
	if h.Version == nil {
		t.Fatal("Version should be non-nil")
	}
	if h.Version.APIVersion != "0.8.1-demo" {
		t.Errorf("Version.APIVersion = %q", h.Version.APIVersion)
	}
}

func TestDemoListBanksOrder(t *testing.T) {
	c := NewDemoClient()
	banks, err := c.ListBanks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(banks) != 2 {
		t.Fatalf("len(banks) = %d, want 2", len(banks))
	}
	if banks[0].BankID != "default" {
		t.Errorf("banks[0].BankID = %q, want %q", banks[0].BankID, "default")
	}
	if banks[1].BankID != "research" {
		t.Errorf("banks[1].BankID = %q, want %q", banks[1].BankID, "research")
	}
	if banks[0].FactCount != 3 {
		t.Errorf("default FactCount = %d, want 3", banks[0].FactCount)
	}
	if banks[0].Name == nil || *banks[0].Name != "Personal" {
		t.Error("default Name should be 'Personal'")
	}
}

func TestDemoGetBankFound(t *testing.T) {
	c := NewDemoClient()
	p, err := c.GetBank(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	if p.BankID != "default" {
		t.Errorf("BankID = %q", p.BankID)
	}
	if p.Mission != "Personal preferences and operating habits." {
		t.Errorf("Mission = %q", p.Mission)
	}
}

func TestDemoGetBankNotFound(t *testing.T) {
	c := NewDemoClient()
	p, err := c.GetBank(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing bank")
	}
	if p != nil {
		t.Error("expected nil profile")
	}
	if !strings.Contains(err.Error(), `bank "missing" not found`) {
		t.Errorf("error = %q, expected to contain bank-not-found message", err.Error())
	}
}

func TestDemoBankStats(t *testing.T) {
	c := NewDemoClient()
	stats, err := c.BankStats(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	if stats["bank_id"] != "default" {
		t.Errorf("bank_id = %v", stats["bank_id"])
	}
	if stats["fact_count"] != 3 {
		t.Errorf("fact_count = %v, want 3", stats["fact_count"])
	}
	for _, key := range []string{"document_count", "tag_count", "entity_count", "last_document_at"} {
		if _, ok := stats[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}
	if _, ok := stats["latest_operation"]; !ok {
		t.Error("missing latest_operation key")
	}
}

func TestDemoBankStatsMissingBank(t *testing.T) {
	c := NewDemoClient()
	_, err := c.BankStats(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error for missing bank")
	}
}

// ---------------------------------------------------------------------------
// Step 2 — Pure helper functions
// ---------------------------------------------------------------------------

func TestDemoMatchScore(t *testing.T) {
	tests := []struct {
		text, query string
		want        int
	}{
		{"anything", "", 1},
		{"dark mode terminal", "dark terminal", 2},
		{"hello world", "xyz", 0},
		{"Dark Mode", "dark", 1},
	}
	for _, tt := range tests {
		got := matchScore(tt.text, tt.query)
		if got != tt.want {
			t.Errorf("matchScore(%q, %q) = %d, want %d", tt.text, tt.query, got, tt.want)
		}
	}
}

func TestDemoMatchesQuery(t *testing.T) {
	if !matchesQuery("hello", "") {
		t.Error("empty query should match")
	}
	if matchesQuery("hello", "zzz") {
		t.Error("non-matching query should not match")
	}
	if !matchesQuery("Hello World", "world") {
		t.Error("case-insensitive match should succeed")
	}
}

func TestDemoMatchesTagFilter(t *testing.T) {
	if !matchesTagFilter([]string{"a"}, nil) {
		t.Error("nil want should match")
	}
	if !matchesTagFilter([]string{"ui", "x"}, []string{"UI"}) {
		t.Error("case-insensitive tag match should succeed")
	}
	if matchesTagFilter([]string{"a"}, []string{"b"}) {
		t.Error("non-matching tag should fail")
	}
}

func TestDemoInferMemoryType(t *testing.T) {
	obsTag := domain.MemoryItem{Tags: []string{"observation"}}
	if got := inferMemoryType(obsTag); got != "observation" {
		t.Errorf("tag observation → %q, want observation", got)
	}
	obsCtx := domain.MemoryItem{Context: stringPtr("We observed this earlier")}
	if got := inferMemoryType(obsCtx); got != "observation" {
		t.Errorf("context with 'observed' → %q, want observation", got)
	}
	plain := domain.MemoryItem{}
	if got := inferMemoryType(plain); got != "experience" {
		t.Errorf("plain item → %q, want experience", got)
	}
}

func TestDemoInferEntities(t *testing.T) {
	got := inferEntities("The project uses Bubble Tea")
	want := []string{"project", "uses", "bubble"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("inferEntities(...) = %v, want %v", got, want)
	}
	got2 := inferEntities("a bb ccc")
	if len(got2) != 0 {
		t.Errorf("all short words → %v, want []", got2)
	}
}

func TestDemoPage(t *testing.T) {
	rows := make([]map[string]any, 5)
	for i := range rows {
		rows[i] = map[string]any{"i": i}
	}

	// limit=2, offset=0
	p := page(rows, 2, 0)
	if len(p.Items) != 2 || p.Total != 5 || p.Limit != 2 || p.Offset != 0 {
		t.Errorf("page(2,0): Items=%d Total=%d Limit=%d Offset=%d", len(p.Items), p.Total, p.Limit, p.Offset)
	}

	// limit=0 → clamp to total
	p = page(rows, 0, 0)
	if p.Limit != 5 || len(p.Items) != 5 {
		t.Errorf("page(0,0): Limit=%d Items=%d", p.Limit, len(p.Items))
	}

	// limit=10, offset=3 → limit clamps to 2
	p = page(rows, 10, 3)
	if p.Limit != 2 || len(p.Items) != 2 || p.Offset != 3 {
		t.Errorf("page(10,3): Limit=%d Items=%d Offset=%d", p.Limit, len(p.Items), p.Offset)
	}

	// offset past end
	p = page(rows, 2, 10)
	if len(p.Items) != 0 || p.Total != 5 || p.Offset != 10 {
		t.Errorf("page(2,10): Items=%d Total=%d Offset=%d", len(p.Items), p.Total, p.Offset)
	}

	// negative offset → clamps to 0
	p = page(rows, 2, -1)
	if p.Offset != 0 {
		t.Errorf("page(2,-1): Offset=%d, want 0", p.Offset)
	}
}

func TestDemoMergeStringSlices(t *testing.T) {
	got := mergeStringSlices([]string{"b", "a"}, []string{"a", "c"})
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("mergeStringSlices = %v, want %v", got, want)
	}
}

func TestDemoCollectEntities(t *testing.T) {
	results := []domain.RecallResult{
		{Entities: []string{"a", "b"}},
		{Entities: []string{"b", "c"}},
	}
	got := collectEntities(results)
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("collectEntities = %v, want %v", got, want)
	}
}

func TestDemoIncludeEntitiesAndFlag(t *testing.T) {
	if includeEntities(nil) {
		t.Error("nil include → false")
	}
	if !includeEntities(map[string]any{"entities": true}) {
		t.Error("entities key present → true")
	}

	// includeFlag
	if !includeFlag(map[string]any{"chunks": true}, "chunks") {
		t.Error("bool true → true")
	}
	if includeFlag(map[string]any{"chunks": map[string]any{"enabled": false}}, "chunks") {
		t.Error("nested enabled=false → false")
	}
	if !includeFlag(map[string]any{"chunks": map[string]any{}}, "chunks") {
		t.Error("nested map without enabled → true (default)")
	}
	if includeFlag(nil, "x") {
		t.Error("nil include → false")
	}
}

func TestDemoContainsAndAnyToStrings(t *testing.T) {
	if !contains([]string{"a", "b"}, "b") {
		t.Error("contains should find 'b'")
	}
	if contains([]string{"a", "b"}, "c") {
		t.Error("contains should not find 'c'")
	}

	got := anyToStrings([]string{"x"})
	if !reflect.DeepEqual(got, []string{"x"}) {
		t.Errorf("anyToStrings([]string{x}) = %v", got)
	}
	if got := anyToStrings("notaslice"); got != nil {
		t.Errorf("anyToStrings(string) = %v, want nil", got)
	}
	if got := anyToStrings([]any{"x"}); got != nil {
		t.Errorf("anyToStrings([]any) = %v, want nil", got)
	}
}

// ---------------------------------------------------------------------------
// Step 3 — Bank CRUD & config
// ---------------------------------------------------------------------------

func TestDemoCreateOrUpdateBankCreate(t *testing.T) {
	c := NewDemoClient()
	p, err := c.CreateOrUpdateBank(context.Background(), "newbank", domain.CreateBankRequest{
		Name:           stringPtr("My Bank"),
		ReflectMission: stringPtr("do things"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "My Bank" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.Mission != "do things" {
		t.Errorf("Mission = %q", p.Mission)
	}
	banks, _ := c.ListBanks(context.Background())
	if len(banks) != 3 {
		t.Errorf("ListBanks len = %d, want 3", len(banks))
	}
	found := false
	for _, b := range banks {
		if b.BankID == "newbank" {
			found = true
		}
	}
	if !found {
		t.Error("newbank not in ListBanks")
	}
}

func TestDemoCreateOrUpdateBankUpdate(t *testing.T) {
	c := NewDemoClient()
	_, err := c.CreateOrUpdateBank(context.Background(), "default", domain.CreateBankRequest{
		Name: stringPtr("Renamed"),
	})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := c.GetBank(context.Background(), "default")
	if p.Name != "Renamed" {
		t.Errorf("Name after update = %q", p.Name)
	}
	banks, _ := c.ListBanks(context.Background())
	if len(banks) != 2 {
		t.Errorf("ListBanks len = %d, want 2 (no duplicate)", len(banks))
	}
}

func TestDemoApplyBankRequestMissionPrecedence(t *testing.T) {
	bank := &demoBank{
		config: domain.BankConfig{Overrides: map[string]any{}},
	}
	// Both set → ReflectMission wins
	applyBankRequest(bank, domain.CreateBankRequest{
		ReflectMission: stringPtr("R"),
		RetainMission:  stringPtr("T"),
	})
	if bank.profile.Mission != "R" {
		t.Errorf("both set → Mission=%q, want R", bank.profile.Mission)
	}

	// Only RetainMission
	bank.profile.Mission = ""
	applyBankRequest(bank, domain.CreateBankRequest{
		RetainMission: stringPtr("T"),
	})
	if bank.profile.Mission != "T" {
		t.Errorf("only retain → Mission=%q, want T", bank.profile.Mission)
	}
}

func TestDemoApplyBankRequestOverrides(t *testing.T) {
	c := NewDemoClient()
	_, err := c.CreateOrUpdateBank(context.Background(), "newbank", domain.CreateBankRequest{
		EnableObservations:       boolPtr(true),
		RetainExtractionMode:     stringPtr("custom"),
		RetainCustomInstructions: stringPtr("inst"),
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := c.GetBankConfig(context.Background(), "newbank")
	if cfg.Overrides["enable_observations"] != true {
		t.Error("enable_observations not set")
	}
	if cfg.Overrides["retain_extraction_mode"] != "custom" {
		t.Error("retain_extraction_mode not set")
	}
	if cfg.Overrides["retain_custom_instructions"] != "inst" {
		t.Error("retain_custom_instructions not set")
	}
}

func TestDemoPatchBankNotFound(t *testing.T) {
	c := NewDemoClient()
	_, err := c.PatchBank(context.Background(), "ghost", domain.CreateBankRequest{})
	if err == nil {
		t.Fatal("expected error for missing bank")
	}
}

func TestDemoDeleteBank(t *testing.T) {
	c := NewDemoClient()
	if err := c.DeleteBank(context.Background(), "default"); err != nil {
		t.Fatal(err)
	}
	banks, _ := c.ListBanks(context.Background())
	if len(banks) != 1 || banks[0].BankID != "research" {
		t.Errorf("after delete: banks = %v", banks)
	}
	_, err := c.GetBank(context.Background(), "default")
	if err == nil {
		t.Error("expected error getting deleted bank")
	}
}

func TestDemoDeleteBankNotFound(t *testing.T) {
	c := NewDemoClient()
	if err := c.DeleteBank(context.Background(), "ghost"); err == nil {
		t.Fatal("expected error for missing bank")
	}
}

func TestDemoGetBankConfigCloneIsolation(t *testing.T) {
	c := NewDemoClient()
	cfg1, _ := c.GetBankConfig(context.Background(), "default")
	cfg1.Config["injected"] = "mutation"
	cfg2, _ := c.GetBankConfig(context.Background(), "default")
	if _, ok := cfg2.Config["injected"]; ok {
		t.Error("mutation leaked through clone")
	}
}

func TestDemoPatchBankConfigMerge(t *testing.T) {
	c := NewDemoClient()
	cfg, err := c.PatchBankConfig(context.Background(), "default", map[string]any{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Overrides["k"] != "v" {
		t.Error("patched key not present")
	}
	if cfg.Overrides["observations"] != true {
		t.Error("seeded override lost after patch")
	}
}

func TestDemoPatchBankConfigMissingBank(t *testing.T) {
	c := NewDemoClient()
	_, err := c.PatchBankConfig(context.Background(), "ghost", map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing bank")
	}
}

// ---------------------------------------------------------------------------
// Step 4 — Retain
// ---------------------------------------------------------------------------

func TestDemoRetainAddsMemoryAtFront(t *testing.T) {
	c := NewDemoClient()
	resp, err := c.Retain(context.Background(), "default", domain.RetainRequest{
		Items: []domain.MemoryItem{{Content: "New fact about widgets", Tags: []string{"x"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.ItemsCount != 1 || resp.Async {
		t.Errorf("response: %+v", resp)
	}
	page, _ := c.ListMemories(context.Background(), "default", "", "", 0, 0)
	if len(page.Items) != 4 {
		t.Fatalf("expected 4 memories, got %d", len(page.Items))
	}
	first := page.Items[0]
	if first["id"] != "mem-6" {
		t.Errorf("first id = %v, want mem-6", first["id"])
	}
	if first["text"] != "New fact about widgets" {
		t.Errorf("first text = %v", first["text"])
	}
}

func TestDemoRetainTypeInference(t *testing.T) {
	c := NewDemoClient()
	// Tag "observation" → type "observation"
	c.Retain(context.Background(), "default", domain.RetainRequest{
		Items: []domain.MemoryItem{{Content: "Observed something", Tags: []string{"observation"}}},
	})
	page, _ := c.ListMemories(context.Background(), "default", "", "", 0, 0)
	if page.Items[0]["type"] != "observation" {
		t.Errorf("type = %v, want observation", page.Items[0]["type"])
	}

	// Plain item → "experience"
	c2 := NewDemoClient()
	c2.Retain(context.Background(), "default", domain.RetainRequest{
		Items: []domain.MemoryItem{{Content: "Plain fact"}},
	})
	page2, _ := c2.ListMemories(context.Background(), "default", "", "", 0, 0)
	if page2.Items[0]["type"] != "experience" {
		t.Errorf("type = %v, want experience", page2.Items[0]["type"])
	}
}

func TestDemoRetainOperationAndAudit(t *testing.T) {
	c := NewDemoClient()
	c.Retain(context.Background(), "default", domain.RetainRequest{
		Items: []domain.MemoryItem{{Content: "Test"}},
	})
	ops, _ := c.ListOperations(context.Background(), "default", "", "", 0, 0)
	if len(ops.Items) == 0 {
		t.Fatal("no operations")
	}
	first := ops.Items[0]
	if first["id"] != "op-100" {
		t.Errorf("op id = %v, want op-100", first["id"])
	}
	if first["status"] != "completed" {
		t.Errorf("op status = %v, want completed", first["status"])
	}
	if first["type"] != "retain" {
		t.Errorf("op type = %v, want retain", first["type"])
	}

	audit, _ := c.ListAuditLogs(context.Background(), "default", TraceFilters{}, 0, 0)
	if len(audit.Items) == 0 {
		t.Fatal("no audit logs")
	}
	firstAudit := audit.Items[0]
	if firstAudit["action"] != "retain" || firstAudit["status"] != "completed" {
		t.Errorf("audit log: action=%v status=%v", firstAudit["action"], firstAudit["status"])
	}
}

func TestDemoRetainAsync(t *testing.T) {
	c := NewDemoClient()
	resp, err := c.Retain(context.Background(), "default", domain.RetainRequest{
		Items: []domain.MemoryItem{{Content: "Async item"}},
		Async: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Async {
		t.Error("Async should be true")
	}
	if resp.OperationID == nil {
		t.Fatal("OperationID should be non-nil")
	}
	if len(resp.OperationIDs) != 1 {
		t.Errorf("OperationIDs len = %d, want 1", len(resp.OperationIDs))
	}
	ops, _ := c.ListOperations(context.Background(), "default", "", "", 0, 0)
	if ops.Items[0]["status"] != "queued" {
		t.Errorf("async op status = %v, want queued", ops.Items[0]["status"])
	}
}

func TestDemoRetainNotFound(t *testing.T) {
	c := NewDemoClient()
	_, err := c.Retain(context.Background(), "ghost", domain.RetainRequest{
		Items: []domain.MemoryItem{{Content: "x"}},
	})
	if err == nil {
		t.Fatal("expected error for missing bank")
	}
}

// ---------------------------------------------------------------------------
// Step 5 — Recall
// ---------------------------------------------------------------------------

func TestDemoRecallEmptyQuery(t *testing.T) {
	c := NewDemoClient()
	resp, err := c.Recall(context.Background(), "default", domain.RecallRequest{Query: ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 3 {
		t.Fatalf("results len = %d, want 3", len(resp.Results))
	}
	// Ordered by CreatedAt desc (tiebreak for same score)
	if resp.Results[0].ID != "mem-3" {
		t.Errorf("results[0].ID = %q, want mem-3", resp.Results[0].ID)
	}
	if resp.Results[1].ID != "mem-2" {
		t.Errorf("results[1].ID = %q, want mem-2", resp.Results[1].ID)
	}
	if resp.Results[2].ID != "mem-1" {
		t.Errorf("results[2].ID = %q, want mem-1", resp.Results[2].ID)
	}
}

func TestDemoRecallQueryScoring(t *testing.T) {
	c := NewDemoClient()

	resp, _ := c.Recall(context.Background(), "default", domain.RecallRequest{Query: "dark"})
	if len(resp.Results) != 1 || resp.Results[0].ID != "mem-1" {
		t.Errorf("query 'dark': got %d results", len(resp.Results))
	}

	resp2, _ := c.Recall(context.Background(), "default", domain.RecallRequest{Query: "zzznomatch"})
	if len(resp2.Results) != 0 {
		t.Errorf("non-matching query: got %d results, want 0", len(resp2.Results))
	}
}

func TestDemoRecallTypeFilter(t *testing.T) {
	c := NewDemoClient()
	resp, _ := c.Recall(context.Background(), "default", domain.RecallRequest{Query: "", Types: []string{"world"}})
	if len(resp.Results) != 1 || resp.Results[0].ID != "mem-1" {
		t.Errorf("type 'world': got %d results", len(resp.Results))
	}
}

func TestDemoRecallTagFilter(t *testing.T) {
	c := NewDemoClient()
	resp, _ := c.Recall(context.Background(), "default", domain.RecallRequest{Query: "", Tags: []string{"ui"}})
	if len(resp.Results) != 1 || resp.Results[0].ID != "mem-1" {
		t.Errorf("tag 'ui': got %d results", len(resp.Results))
	}
	resp2, _ := c.Recall(context.Background(), "default", domain.RecallRequest{Query: "", Tags: []string{"nonexistent"}})
	if len(resp2.Results) != 0 {
		t.Errorf("non-matching tag: got %d results", len(resp2.Results))
	}
}

func TestDemoRecallResultFields(t *testing.T) {
	c := NewDemoClient()
	resp, _ := c.Recall(context.Background(), "default", domain.RecallRequest{Query: "dark"})
	r := resp.Results[0]
	if r.Type == nil || *r.Type != "world" {
		t.Errorf("Type = %v, want world", r.Type)
	}
	if !contains(r.Tags, "ui") {
		t.Error("Tags should contain 'ui'")
	}
	if r.DocumentID == nil || *r.DocumentID != "doc-1" {
		t.Errorf("DocumentID = %v, want doc-1", r.DocumentID)
	}
	if len(r.SourceFactIDs) != 1 || r.SourceFactIDs[0] != "mem-1" {
		t.Errorf("SourceFactIDs = %v", r.SourceFactIDs)
	}
	if r.ChunkID == nil || *r.ChunkID != "chunk-mem-1" {
		t.Errorf("ChunkID = %v, want chunk-mem-1", r.ChunkID)
	}
}

func TestDemoRecallIncludeEntitiesChunksSourceFacts(t *testing.T) {
	c := NewDemoClient()

	resp, _ := c.Recall(context.Background(), "default", domain.RecallRequest{
		Query:   "dark",
		Include: map[string]any{"entities": true},
	})
	if _, ok := resp.Entities["items"]; !ok {
		t.Error("entities.items missing")
	}
	if resp.Entities["max_tokens"] != 500 {
		t.Error("entities.max_tokens != 500")
	}

	resp2, _ := c.Recall(context.Background(), "default", domain.RecallRequest{
		Query:   "dark",
		Include: map[string]any{"chunks": true},
	})
	if resp2.Chunks["enabled"] != true {
		t.Error("chunks.enabled != true")
	}

	resp3, _ := c.Recall(context.Background(), "default", domain.RecallRequest{
		Query:   "dark",
		Include: map[string]any{"source_facts": true},
	})
	if _, ok := resp3.SourceFacts["mem-1"]; !ok {
		t.Error("source_facts missing mem-1")
	}
}

func TestDemoRecallTrace(t *testing.T) {
	c := NewDemoClient()
	resp, _ := c.Recall(context.Background(), "default", domain.RecallRequest{Query: "dark", Trace: true})
	if resp.Trace["provider"] != "demo" {
		t.Errorf("trace.provider = %v", resp.Trace["provider"])
	}
	if resp.Trace["matched_results"] != 1 {
		t.Errorf("trace.matched_results = %v", resp.Trace["matched_results"])
	}
	if resp.Trace["bank_id"] != "default" {
		t.Errorf("trace.bank_id = %v", resp.Trace["bank_id"])
	}
}

func TestDemoRecallNotFound(t *testing.T) {
	c := NewDemoClient()
	_, err := c.Recall(context.Background(), "ghost", domain.RecallRequest{Query: ""})
	if err == nil {
		t.Fatal("expected error for missing bank")
	}
}

// ---------------------------------------------------------------------------
// Step 6 — Reflect
// ---------------------------------------------------------------------------

func TestDemoReflectAnswerFromTopResult(t *testing.T) {
	c := NewDemoClient()
	resp, err := c.Reflect(context.Background(), "default", domain.ReflectRequest{Query: "dark"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(resp.Text, "Based on stored memory: ") {
		t.Errorf("Text prefix wrong: %q", resp.Text)
	}
	if !strings.Contains(resp.Text, "User prefers dark mode") {
		t.Error("Text should contain mem-1 content")
	}
	ids, ok := resp.BasedOn["memory_ids"].([]string)
	if !ok || !contains(ids, "mem-1") {
		t.Errorf("BasedOn.memory_ids = %v", resp.BasedOn["memory_ids"])
	}
	if resp.Usage == nil {
		t.Error("Usage should be non-nil")
	}
}

func TestDemoReflectNoMatchFallback(t *testing.T) {
	c := NewDemoClient()
	resp, _ := c.Reflect(context.Background(), "default", domain.ReflectRequest{Query: "zzznomatch"})
	if resp.Text != "No strongly matching memory yet. Retain a few more facts, then try again." {
		t.Errorf("fallback text = %q", resp.Text)
	}
	ids, _ := resp.BasedOn["memory_ids"].([]string)
	if len(ids) != 0 {
		t.Errorf("memory_ids should be empty, got %v", ids)
	}
}

func TestDemoReflectFactTypesTrace(t *testing.T) {
	c := NewDemoClient()
	resp, _ := c.Reflect(context.Background(), "default", domain.ReflectRequest{
		Query:     "dark",
		FactTypes: []string{"world"},
	})
	ft, ok := resp.Trace["fact_types"].([]string)
	if !ok || !reflect.DeepEqual(ft, []string{"world"}) {
		t.Errorf("trace.fact_types = %v", resp.Trace["fact_types"])
	}
}

func TestDemoReflectLLMRequestsPrepended(t *testing.T) {
	c := NewDemoClient()
	c.Reflect(context.Background(), "default", domain.ReflectRequest{Query: "dark"})
	llm, _ := c.ListLLMRequests(context.Background(), "default", TraceFilters{}, 0, 0)
	if len(llm.Items) == 0 {
		t.Fatal("no LLM requests")
	}
	first := llm.Items[0]
	if first["operation"] != "reflect" || first["provider"] != "demo" {
		t.Errorf("first LLM request: op=%v provider=%v", first["operation"], first["provider"])
	}
}

func TestDemoReflectNotFound(t *testing.T) {
	c := NewDemoClient()
	_, err := c.Reflect(context.Background(), "ghost", domain.ReflectRequest{Query: ""})
	if err == nil {
		t.Fatal("expected error for missing bank")
	}
}

// ---------------------------------------------------------------------------
// Step 7 — Listings, filters, pagination
// ---------------------------------------------------------------------------

func TestDemoListMemoriesTypeFilter(t *testing.T) {
	c := NewDemoClient()
	resp, _ := c.ListMemories(context.Background(), "default", "", "observation", 0, 0)
	if len(resp.Items) != 1 || resp.Items[0]["id"] != "mem-3" {
		t.Errorf("type filter: got %d items", len(resp.Items))
	}
}

func TestDemoListMemoriesQueryFilter(t *testing.T) {
	c := NewDemoClient()
	resp, _ := c.ListMemories(context.Background(), "default", "Bubble", "", 0, 0)
	if len(resp.Items) != 1 || resp.Items[0]["id"] != "mem-3" {
		t.Errorf("query filter: got %d items", len(resp.Items))
	}
}

func TestDemoListMemoriesPagination(t *testing.T) {
	c := NewDemoClient()
	resp, _ := c.ListMemories(context.Background(), "default", "", "", 2, 0)
	if len(resp.Items) != 2 {
		t.Errorf("paged items = %d, want 2", len(resp.Items))
	}
	if resp.Total != 3 {
		t.Errorf("total = %d, want 3", resp.Total)
	}
}

func TestDemoListDocumentsTagFilter(t *testing.T) {
	c := NewDemoClient()
	all, _ := c.ListDocuments(context.Background(), "default", "", nil, 0, 0)
	if len(all.Items) < 1 {
		t.Fatal("expected at least 1 document")
	}
	for _, row := range all.Items {
		if _, ok := row["document_id"]; !ok {
			t.Error("missing document_id")
		}
		if _, ok := row["title"]; !ok {
			t.Error("missing title")
		}
		if _, ok := row["memory_count"]; !ok {
			t.Error("missing memory_count")
		}
	}

	// Tag filter: only doc-1 has "ui" tag
	filtered, _ := c.ListDocuments(context.Background(), "default", "", []string{"ui"}, 0, 0)
	if len(filtered.Items) != 1 {
		t.Errorf("tag filter 'ui': got %d docs, want 1", len(filtered.Items))
	}
	if filtered.Items[0]["document_id"] != "doc-1" {
		t.Errorf("filtered doc = %v", filtered.Items[0]["document_id"])
	}

	empty, _ := c.ListDocuments(context.Background(), "default", "", []string{"nonexistent"}, 0, 0)
	if len(empty.Items) != 0 {
		t.Error("non-matching tag filter should return 0")
	}
}

func TestDemoListTagsQueryAndSort(t *testing.T) {
	c := NewDemoClient()
	resp, _ := c.ListTags(context.Background(), "default", "", 0, 0)
	if len(resp.Items) == 0 {
		t.Fatal("expected tags")
	}
	// Sorted by tag ascending
	if resp.Items[0]["tag"] != "architecture" {
		t.Errorf("first tag = %v, want architecture", resp.Items[0]["tag"])
	}
	for _, row := range resp.Items {
		if _, ok := row["count"]; !ok {
			t.Error("missing count")
		}
		if row["source"] != "memories" {
			t.Errorf("source = %v", row["source"])
		}
	}

	uiOnly, _ := c.ListTags(context.Background(), "default", "ui", 0, 0)
	if len(uiOnly.Items) != 1 || uiOnly.Items[0]["tag"] != "ui" {
		t.Errorf("tag 'ui' filter: got %v", uiOnly.Items)
	}
}

func TestDemoListEntitiesQuery(t *testing.T) {
	c := NewDemoClient()
	all, _ := c.ListEntities(context.Background(), "default", "", 0, 0)
	if len(all.Items) == 0 {
		t.Fatal("expected entities")
	}

	dark, _ := c.ListEntities(context.Background(), "default", "dark", 0, 0)
	if len(dark.Items) != 1 || dark.Items[0]["entity"] != "dark mode" {
		t.Errorf("entity 'dark': got %v", dark.Items)
	}
}

func TestDemoListOperationsFilter(t *testing.T) {
	c := NewDemoClient()

	completed, _ := c.ListOperations(context.Background(), "default", "completed", "", 0, 0)
	if len(completed.Items) != 1 {
		t.Errorf("completed: got %d, want 1", len(completed.Items))
	}

	queued, _ := c.ListOperations(context.Background(), "default", "queued", "", 0, 0)
	if len(queued.Items) != 0 {
		t.Error("queued should return 0")
	}

	retainOnly, _ := c.ListOperations(context.Background(), "default", "", "retain", 0, 0)
	if len(retainOnly.Items) != 1 {
		t.Error("retain type should return 1")
	}

	recallOnly, _ := c.ListOperations(context.Background(), "default", "", "recall", 0, 0)
	if len(recallOnly.Items) != 0 {
		t.Error("recall type should return 0")
	}
}

func TestDemoListAuditLogsFilter(t *testing.T) {
	c := NewDemoClient()

	retainLogs, _ := c.ListAuditLogs(context.Background(), "default", TraceFilters{Action: "retain"}, 0, 0)
	if len(retainLogs.Items) != 1 {
		t.Errorf("action 'retain': got %d, want 1", len(retainLogs.Items))
	}

	noAction, _ := c.ListAuditLogs(context.Background(), "default", TraceFilters{Action: "nope"}, 0, 0)
	if len(noAction.Items) != 0 {
		t.Error("action 'nope' should return 0")
	}

	demoTransport, _ := c.ListAuditLogs(context.Background(), "default", TraceFilters{Transport: "demo"}, 0, 0)
	if len(demoTransport.Items) != 1 {
		t.Errorf("transport 'demo': got %d, want 1", len(demoTransport.Items))
	}
}

func TestDemoListLLMRequestsFilter(t *testing.T) {
	c := NewDemoClient()

	reflectReqs, _ := c.ListLLMRequests(context.Background(), "default", TraceFilters{Operation: "reflect"}, 0, 0)
	if len(reflectReqs.Items) != 1 {
		t.Errorf("operation 'reflect': got %d, want 1", len(reflectReqs.Items))
	}

	recallReqs, _ := c.ListLLMRequests(context.Background(), "default", TraceFilters{Operation: "recall"}, 0, 0)
	if len(recallReqs.Items) != 0 {
		t.Error("operation 'recall' should return 0")
	}

	demoProvider, _ := c.ListLLMRequests(context.Background(), "default", TraceFilters{Provider: "demo"}, 0, 0)
	if len(demoProvider.Items) != 1 {
		t.Errorf("provider 'demo': got %d, want 1", len(demoProvider.Items))
	}

	completedReqs, _ := c.ListLLMRequests(context.Background(), "default", TraceFilters{Status: "completed"}, 0, 0)
	if len(completedReqs.Items) != 1 {
		t.Errorf("status 'completed': got %d, want 1", len(completedReqs.Items))
	}
}

func TestDemoListingsMissingBank(t *testing.T) {
	c := NewDemoClient()
	_, err1 := c.ListMemories(context.Background(), "ghost", "", "", 0, 0)
	if err1 == nil {
		t.Error("ListMemories on missing bank should error")
	}
	_, err2 := c.ListOperations(context.Background(), "ghost", "", "", 0, 0)
	if err2 == nil {
		t.Error("ListOperations on missing bank should error")
	}
}

// ---------------------------------------------------------------------------
// Step 8 — Graphs & templates
// ---------------------------------------------------------------------------

func TestDemoGetEntityGraph(t *testing.T) {
	c := NewDemoClient()
	raw, err := c.GetEntityGraph(context.Background(), "default", 0)
	if err != nil {
		t.Fatal(err)
	}
	var g map[string]any
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatal(err)
	}
	nodes, ok := g["nodes"].([]any)
	if !ok || len(nodes) == 0 {
		t.Error("nodes should be non-empty array")
	}
	if _, ok := g["edges"].([]any); !ok {
		t.Error("edges should be array")
	}

	// With limit=1
	raw2, _ := c.GetEntityGraph(context.Background(), "default", 1)
	var g2 map[string]any
	json.Unmarshal(raw2, &g2)
	if len(g2["nodes"].([]any)) != 1 {
		t.Error("limit=1 should return 1 node")
	}

	// Missing bank
	_, err = c.GetEntityGraph(context.Background(), "ghost", 0)
	if err == nil {
		t.Error("missing bank should error")
	}
}

func TestDemoGetMemoryGraph(t *testing.T) {
	c := NewDemoClient()
	raw, err := c.GetMemoryGraph(context.Background(), "default", "", "", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var g map[string]any
	json.Unmarshal(raw, &g)
	nodes := g["nodes"].([]any)
	if len(nodes) != 3 {
		t.Errorf("all memories → %d nodes, want 3", len(nodes))
	}
	// Each node has id, type, text, tags
	first := nodes[0].(map[string]any)
	for _, key := range []string{"id", "type", "text", "tags"} {
		if _, ok := first[key]; !ok {
			t.Errorf("node missing key %q", key)
		}
	}

	// memoryType filter
	raw2, _ := c.GetMemoryGraph(context.Background(), "default", "observation", "", nil, 0)
	var g2 map[string]any
	json.Unmarshal(raw2, &g2)
	if len(g2["nodes"].([]any)) != 1 {
		t.Error("observation filter should return 1 node")
	}

	// query filter
	raw3, _ := c.GetMemoryGraph(context.Background(), "default", "", "Bubble", nil, 0)
	var g3 map[string]any
	json.Unmarshal(raw3, &g3)
	if len(g3["nodes"].([]any)) != 1 {
		t.Error("query 'Bubble' should return 1 node")
	}

	// tag filter
	raw4, _ := c.GetMemoryGraph(context.Background(), "default", "", "", []string{"ui"}, 0)
	var g4 map[string]any
	json.Unmarshal(raw4, &g4)
	if len(g4["nodes"].([]any)) != 1 {
		t.Error("tag 'ui' should return 1 node")
	}

	// limit
	raw5, _ := c.GetMemoryGraph(context.Background(), "default", "", "", nil, 1)
	var g5 map[string]any
	json.Unmarshal(raw5, &g5)
	if len(g5["nodes"].([]any)) != 1 {
		t.Error("limit=1 should return 1 node")
	}

	// missing bank
	_, err = c.GetMemoryGraph(context.Background(), "ghost", "", "", nil, 0)
	if err == nil {
		t.Error("missing bank should error")
	}
}

func TestDemoExportBankTemplate(t *testing.T) {
	c := NewDemoClient()
	raw, err := c.ExportBankTemplate(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	var tmpl map[string]any
	if err := json.Unmarshal(raw, &tmpl); err != nil {
		t.Fatal(err)
	}
	if tmpl["bank_id"] != "default" {
		t.Errorf("bank_id = %v", tmpl["bank_id"])
	}
	if _, ok := tmpl["profile"]; !ok {
		t.Error("missing profile key")
	}
	if _, ok := tmpl["config"]; !ok {
		t.Error("missing config key")
	}

	// Missing bank
	_, err = c.ExportBankTemplate(context.Background(), "ghost")
	if err == nil {
		t.Error("missing bank should error")
	}
}

func TestDemoImportBankTemplate(t *testing.T) {
	c := NewDemoClient()

	// Dry run
	raw, err := c.ImportBankTemplate(context.Background(), "default", json.RawMessage(`{"x":1}`), true)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	json.Unmarshal(raw, &result)
	if result["dry_run"] != true {
		t.Error("dry_run should be true")
	}
	if result["imported"] != false {
		t.Error("imported should be false for dry run")
	}

	// Real import
	raw2, _ := c.ImportBankTemplate(context.Background(), "default", json.RawMessage(`{"x":1}`), false)
	var result2 map[string]any
	json.Unmarshal(raw2, &result2)
	if result2["imported"] != true {
		t.Error("imported should be true")
	}

	// Invalid JSON
	_, err = c.ImportBankTemplate(context.Background(), "default", json.RawMessage(`not json`), false)
	if err == nil {
		t.Error("invalid JSON should error")
	}
	if !strings.Contains(err.Error(), "template is not valid JSON") {
		t.Errorf("error = %q", err.Error())
	}

	// Missing bank (checked before JSON validity)
	_, err = c.ImportBankTemplate(context.Background(), "ghost", json.RawMessage(`{"x":1}`), false)
	if err == nil {
		t.Error("missing bank should error")
	}
}
