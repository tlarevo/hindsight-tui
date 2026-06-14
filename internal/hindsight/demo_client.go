package hindsight

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tlarevo/hindsight-tui/internal/domain"
)

type DemoClient struct {
	mu            sync.Mutex
	version       domain.VersionInfo
	banks         map[string]*demoBank
	bankOrder     []string
	nextMemoryID  int
	nextOperation int
}

type demoBank struct {
	profile     domain.BankProfile
	config      domain.BankConfig
	createdAt   string
	updatedAt   string
	lastDocAt   string
	memories    []demoMemory
	operations  []map[string]any
	auditLogs   []map[string]any
	llmRequests []map[string]any
}

type demoMemory struct {
	ID        string
	Type      string
	Item      domain.MemoryItem
	Entities  []string
	CreatedAt string
}

func NewDemoClient() *DemoClient {
	version := domain.VersionInfo{
		APIVersion: "0.8.1-demo",
		Features: domain.FeatureFlags{
			Observations:      true,
			MCP:               false,
			Worker:            false,
			BankConfigAPI:     true,
			FileUploadAPI:     false,
			DocumentExportAPI: true,
			DocumentImportAPI: true,
			AuditLog:          true,
			LLMTrace:          true,
			StoreDocumentText: true,
		},
	}

	client := &DemoClient{
		version: version,
		banks:   map[string]*demoBank{},
	}

	client.seedBank(
		"default",
		"Personal",
		"Personal preferences and operating habits.",
		[]demoMemory{
			{
				ID:   "mem-1",
				Type: "world",
				Item: domain.MemoryItem{
					Content:    "User prefers dark mode in terminal apps.",
					Tags:       []string{"preference", "ui"},
					Metadata:   map[string]string{"source": "demo"},
					DocumentID: stringPtr("doc-1"),
					Context:    stringPtr("Captured during initial setup."),
				},
				Entities:  []string{"user", "dark mode", "terminal apps"},
				CreatedAt: "2026-06-01T09:00:00Z",
			},
			{
				ID:   "mem-2",
				Type: "experience",
				Item: domain.MemoryItem{
					Content:    "Async retain jobs may take a few seconds before recall surfaces them.",
					Tags:       []string{"workflow", "operations"},
					Metadata:   map[string]string{"source": "demo"},
					DocumentID: stringPtr("doc-2"),
					Context:    stringPtr("Useful while testing the operations view."),
				},
				Entities:  []string{"async retain", "recall", "operations"},
				CreatedAt: "2026-06-02T10:30:00Z",
			},
			{
				ID:   "mem-3",
				Type: "observation",
				Item: domain.MemoryItem{
					Content:    "The project uses Bubble Tea for the TUI runtime.",
					Tags:       []string{"architecture", "observation"},
					Metadata:   map[string]string{"source": "demo"},
					DocumentID: stringPtr("doc-3"),
					Context:    stringPtr("Observed in the bootstrap spec."),
				},
				Entities:  []string{"Bubble Tea", "TUI runtime"},
				CreatedAt: "2026-06-03T08:15:00Z",
			},
		},
	)
	client.seedBank(
		"research",
		"Research",
		"Longer-form product and architecture notes.",
		[]demoMemory{
			{
				ID:   "mem-4",
				Type: "world",
				Item: domain.MemoryItem{
					Content:    "Tracing endpoints can show provider latency and token usage.",
					Tags:       []string{"traces", "llm"},
					Metadata:   map[string]string{"source": "demo"},
					DocumentID: stringPtr("doc-4"),
					Context:    stringPtr("Saved from an architecture review."),
				},
				Entities:  []string{"tracing", "provider latency", "token usage"},
				CreatedAt: "2026-06-04T14:00:00Z",
			},
			{
				ID:   "mem-5",
				Type: "experience",
				Item: domain.MemoryItem{
					Content:    "Keep dashboard cards resilient when optional endpoints return 404 or 405.",
					Tags:       []string{"dashboard", "resilience"},
					Metadata:   map[string]string{"source": "demo"},
					DocumentID: stringPtr("doc-5"),
					Context:    stringPtr("Derived from the approved bootstrap plan."),
				},
				Entities:  []string{"dashboard", "404", "405"},
				CreatedAt: "2026-06-05T16:45:00Z",
			},
		},
	)
	client.nextMemoryID = 6
	client.nextOperation = 100

	return client
}

func (c *DemoClient) Health(context.Context) (*domain.HealthStatus, error) {
	version := c.version
	return &domain.HealthStatus{OK: true, Version: &version, Detail: "demo backend ready"}, nil
}

func (c *DemoClient) Version(context.Context) (*domain.VersionInfo, error) {
	version := c.version
	return &version, nil
}

func (c *DemoClient) ListBanks(context.Context) ([]domain.BankSummary, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	banks := make([]domain.BankSummary, 0, len(c.bankOrder))
	for _, bankID := range c.bankOrder {
		banks = append(banks, c.banks[bankID].summary())
	}
	return banks, nil
}

func (c *DemoClient) GetBank(_ context.Context, bankID string) (*domain.BankProfile, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	profile := bank.profile
	return &profile, nil
}

func (c *DemoClient) CreateOrUpdateBank(_ context.Context, bankID string, req domain.CreateBankRequest) (*domain.BankProfile, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank := c.banks[bankID]
	if bank == nil {
		bank = &demoBank{
			profile: domain.BankProfile{
				BankID:      bankID,
				Name:        bankID,
				Mission:     "Demo bank",
				Disposition: domain.Disposition{Skepticism: 5, Literalism: 5, Empathy: 5},
			},
			config:    domain.BankConfig{BankID: bankID, Config: map[string]any{"theme": "auto"}, Overrides: map[string]any{}},
			createdAt: time.Now().UTC().Format(time.RFC3339),
			updatedAt: time.Now().UTC().Format(time.RFC3339),
			lastDocAt: time.Now().UTC().Format(time.RFC3339),
		}
		c.banks[bankID] = bank
		c.bankOrder = append(c.bankOrder, bankID)
	}
	applyBankRequest(bank, req)
	profile := bank.profile
	return &profile, nil
}

func (c *DemoClient) PatchBank(_ context.Context, bankID string, req domain.CreateBankRequest) (*domain.BankProfile, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	applyBankRequest(bank, req)
	profile := bank.profile
	return &profile, nil
}

func (c *DemoClient) DeleteBank(_ context.Context, bankID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := c.bank(bankID); err != nil {
		return err
	}
	delete(c.banks, bankID)
	for i, id := range c.bankOrder {
		if id == bankID {
			c.bankOrder = append(c.bankOrder[:i], c.bankOrder[i+1:]...)
			break
		}
	}
	return nil
}

func (c *DemoClient) GetBankConfig(_ context.Context, bankID string) (*domain.BankConfig, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	config := domain.BankConfig{BankID: bank.config.BankID, Config: cloneMapAny(bank.config.Config), Overrides: cloneMapAny(bank.config.Overrides)}
	return &config, nil
}

func (c *DemoClient) PatchBankConfig(_ context.Context, bankID string, updates map[string]any) (*domain.BankConfig, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	if bank.config.Overrides == nil {
		bank.config.Overrides = map[string]any{}
	}
	for key, value := range updates {
		bank.config.Overrides[key] = value
	}
	bank.updatedAt = time.Now().UTC().Format(time.RFC3339)
	config := domain.BankConfig{BankID: bank.config.BankID, Config: cloneMapAny(bank.config.Config), Overrides: cloneMapAny(bank.config.Overrides)}
	return &config, nil
}

func (c *DemoClient) BankStats(_ context.Context, bankID string) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	stats := map[string]any{
		"bank_id":          bankID,
		"fact_count":       len(bank.memories),
		"document_count":   len(c.documentRows(bank)),
		"tag_count":        len(c.tagRows(bank, "")),
		"entity_count":     len(c.entityRows(bank, "")),
		"last_document_at": bank.lastDocAt,
	}
	if len(bank.operations) > 0 {
		stats["latest_operation"] = cloneMapAny(bank.operations[0])
	}
	return stats, nil
}

func (c *DemoClient) Retain(_ context.Context, bankID string, req domain.RetainRequest) (*domain.RetainResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	for _, item := range req.Items {
		memoryType := inferMemoryType(item)
		memory := demoMemory{
			ID:        fmt.Sprintf("mem-%d", c.nextMemoryID),
			Type:      memoryType,
			Item:      copyMemoryItem(item),
			Entities:  inferEntities(item.Content),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		c.nextMemoryID++
		bank.memories = append([]demoMemory{memory}, bank.memories...)
		if item.DocumentID != nil {
			bank.lastDocAt = memory.CreatedAt
		}
	}

	operationID := fmt.Sprintf("op-%d", c.nextOperation)
	c.nextOperation++
	operationStatus := "completed"
	if req.Async {
		operationStatus = "queued"
	}
	bank.operations = append([]map[string]any{map[string]any{
		"id":          operationID,
		"status":      operationStatus,
		"type":        "retain",
		"bank_id":     bankID,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
		"items_count": len(req.Items),
	}}, bank.operations...)
	bank.auditLogs = append([]map[string]any{map[string]any{
		"id":         fmt.Sprintf("audit-%d", c.nextOperation),
		"action":     "retain",
		"transport":  "demo",
		"status":     operationStatus,
		"bank_id":    bankID,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}}, bank.auditLogs...)

	response := &domain.RetainResponse{
		Success:    true,
		BankID:     bankID,
		ItemsCount: len(req.Items),
		Async:      req.Async,
		Usage:      &domain.TokenUsage{InputTokens: 12, OutputTokens: 6, TotalTokens: 18},
	}
	if req.Async {
		response.OperationID = &operationID
		response.OperationIDs = []string{operationID}
	}
	return response, nil
}

func (c *DemoClient) Recall(_ context.Context, bankID string, req domain.RecallRequest) (*domain.RecallResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	results := c.recallResults(bank, req)
	response := &domain.RecallResponse{
		Results:     results,
		Trace:       map[string]any{},
		Entities:    map[string]any{},
		Chunks:      map[string]any{},
		SourceFacts: map[string]domain.RecallResult{},
	}
	if includeEntities(req.Include) {
		response.Entities["items"] = collectEntities(results)
		response.Entities["max_tokens"] = 500
	}
	if includeFlag(req.Include, "chunks") {
		response.Chunks["enabled"] = true
		response.Chunks["items"] = []map[string]any{{"chunk_id": "chunk-demo-1", "summary": "Demo chunk for recalled text."}}
	}
	if includeFlag(req.Include, "source_facts") {
		for _, result := range results {
			response.SourceFacts[result.ID] = result
		}
	}
	if req.Trace {
		response.Trace["provider"] = "demo"
		response.Trace["matched_results"] = len(results)
		response.Trace["bank_id"] = bankID
	}
	return response, nil
}

func (c *DemoClient) Reflect(_ context.Context, bankID string, req domain.ReflectRequest) (*domain.ReflectResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	recall := c.recallResults(bank, domain.RecallRequest{Query: req.Query, Tags: req.Tags})
	text := "No strongly matching memory yet. Retain a few more facts, then try again."
	basedOn := map[string]any{"bank_id": bankID, "memory_ids": []string{}}
	if len(recall) > 0 {
		ids := make([]string, 0, len(recall))
		snippets := make([]string, 0, len(recall))
		for i, result := range recall {
			if i == 0 {
				text = "Based on stored memory: " + result.Text
			}
			ids = append(ids, result.ID)
			snippets = append(snippets, result.Text)
		}
		basedOn["memory_ids"] = ids
		basedOn["snippets"] = snippets
	}
	trace := map[string]any{"provider": "demo", "bank_id": bankID}
	if len(req.FactTypes) > 0 {
		trace["fact_types"] = append([]string(nil), req.FactTypes...)
	}
	response := &domain.ReflectResponse{
		Text:             text,
		BasedOn:          basedOn,
		StructuredOutput: map[string]any{"bank_id": bankID, "answer_length": len(text)},
		Usage:            &domain.TokenUsage{InputTokens: 24, OutputTokens: 20, TotalTokens: 44},
		Trace:            trace,
	}
	bank.llmRequests = append([]map[string]any{map[string]any{
		"id":          fmt.Sprintf("llm-%d", c.nextOperation),
		"status":      "completed",
		"operation":   "reflect",
		"scope":       "demo",
		"provider":    "demo",
		"trace_id":    fmt.Sprintf("trace-%d", c.nextOperation),
		"document_id": "",
		"memory_id":   firstMemoryID(recall),
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	}}, bank.llmRequests...)
	return response, nil
}

func (c *DemoClient) ListMemories(_ context.Context, bankID string, q string, memoryType string, limit, offset int) (*domain.Page[map[string]any], error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(bank.memories))
	for _, memory := range bank.memories {
		if memoryType != "" && memory.Type != memoryType {
			continue
		}
		if !matchesQuery(memory.Item.Content+" "+valueOrEmpty(memory.Item.Context), q) {
			continue
		}
		rows = append(rows, memoryRow(memory))
	}
	return page(rows, limit, offset), nil
}

func (c *DemoClient) ListDocuments(_ context.Context, bankID string, q string, tags []string, limit, offset int) (*domain.Page[map[string]any], error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0)
	for _, row := range c.documentRows(bank) {
		if !matchesQuery(stringify(row), q) || !matchesTagFilter(anyToStrings(row["tags"]), tags) {
			continue
		}
		rows = append(rows, cloneMapAny(row))
	}
	return page(rows, limit, offset), nil
}

func (c *DemoClient) ListTags(_ context.Context, bankID string, q string, limit, offset int) (*domain.Page[map[string]any], error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	return page(c.tagRows(bank, q), limit, offset), nil
}

func (c *DemoClient) ListEntities(_ context.Context, bankID string, q string, limit, offset int) (*domain.Page[map[string]any], error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	return page(c.entityRows(bank, q), limit, offset), nil
}

func (c *DemoClient) GetEntityGraph(_ context.Context, bankID string, limit int) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	nodes := c.entityRows(bank, "")
	if limit > 0 && len(nodes) > limit {
		nodes = nodes[:limit]
	}
	edges := make([]map[string]any, 0)
	for _, memory := range bank.memories {
		for i := 0; i+1 < len(memory.Entities); i++ {
			edges = append(edges, map[string]any{"source": memory.Entities[i], "target": memory.Entities[i+1], "memory_id": memory.ID})
		}
	}
	payload, err := json.Marshal(map[string]any{"nodes": nodes, "edges": edges})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func (c *DemoClient) GetMemoryGraph(_ context.Context, bankID string, memoryType string, q string, tags []string, limit int) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	nodes := make([]map[string]any, 0)
	edges := make([]map[string]any, 0)
	for _, memory := range bank.memories {
		if memoryType != "" && memory.Type != memoryType {
			continue
		}
		if !matchesQuery(memory.Item.Content+" "+valueOrEmpty(memory.Item.Context), q) || !matchesTagFilter(memory.Item.Tags, tags) {
			continue
		}
		nodes = append(nodes, map[string]any{"id": memory.ID, "type": memory.Type, "text": memory.Item.Content, "tags": append([]string(nil), memory.Item.Tags...)})
		for _, entity := range memory.Entities {
			edges = append(edges, map[string]any{"source": memory.ID, "target": entity, "relationship": "mentions"})
		}
		if limit > 0 && len(nodes) >= limit {
			break
		}
	}
	payload, err := json.Marshal(map[string]any{"nodes": nodes, "edges": edges})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func (c *DemoClient) ListOperations(_ context.Context, bankID string, status string, opType string, limit, offset int) (*domain.Page[map[string]any], error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(bank.operations))
	for _, row := range bank.operations {
		if status != "" && row["status"] != status {
			continue
		}
		if opType != "" && row["type"] != opType {
			continue
		}
		rows = append(rows, cloneMapAny(row))
	}
	return page(rows, limit, offset), nil
}

func (c *DemoClient) ListAuditLogs(_ context.Context, bankID string, filters TraceFilters, limit, offset int) (*domain.Page[map[string]any], error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(bank.auditLogs))
	for _, row := range bank.auditLogs {
		if filters.Action != "" && row["action"] != filters.Action {
			continue
		}
		if filters.Transport != "" && row["transport"] != filters.Transport {
			continue
		}
		rows = append(rows, cloneMapAny(row))
	}
	return page(rows, limit, offset), nil
}

func (c *DemoClient) ListLLMRequests(_ context.Context, bankID string, filters TraceFilters, limit, offset int) (*domain.Page[map[string]any], error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(bank.llmRequests))
	for _, row := range bank.llmRequests {
		if filters.Status != "" && row["status"] != filters.Status {
			continue
		}
		if filters.Operation != "" && row["operation"] != filters.Operation {
			continue
		}
		if filters.Scope != "" && row["scope"] != filters.Scope {
			continue
		}
		if filters.Provider != "" && row["provider"] != filters.Provider {
			continue
		}
		if filters.TraceID != "" && row["trace_id"] != filters.TraceID {
			continue
		}
		rows = append(rows, cloneMapAny(row))
	}
	return page(rows, limit, offset), nil
}

func (c *DemoClient) ExportBankTemplate(_ context.Context, bankID string) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bank, err := c.bank(bankID)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]any{
		"bank_id": bankID,
		"profile": bank.profile,
		"config":  bank.config,
	})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func (c *DemoClient) ImportBankTemplate(_ context.Context, bankID string, template json.RawMessage, dryRun bool) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := c.bank(bankID); err != nil {
		return nil, err
	}
	if !json.Valid(template) {
		return nil, fmt.Errorf("template is not valid JSON")
	}
	payload, err := json.Marshal(map[string]any{
		"bank_id":  bankID,
		"dry_run":  dryRun,
		"imported": !dryRun,
		"message":  "Demo import completed locally.",
	})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func (c *DemoClient) seedBank(bankID string, name string, mission string, memories []demoMemory) {
	createdAt := memories[len(memories)-1].CreatedAt
	updatedAt := memories[0].CreatedAt
	lastDocAt := updatedAt
	bank := &demoBank{
		profile: domain.BankProfile{
			BankID:      bankID,
			Name:        name,
			Mission:     mission,
			Background:  stringPtr("Demo seed data for UI development."),
			Disposition: domain.Disposition{Skepticism: 6, Literalism: 5, Empathy: 7},
		},
		config: domain.BankConfig{
			BankID:    bankID,
			Config:    map[string]any{"theme": "auto", "notes": "demo"},
			Overrides: map[string]any{"observations": true},
		},
		createdAt: createdAt,
		updatedAt: updatedAt,
		lastDocAt: lastDocAt,
		memories:  append([]demoMemory(nil), memories...),
		operations: []map[string]any{
			{
				"id":         fmt.Sprintf("op-seed-%s", bankID),
				"status":     "completed",
				"type":       "retain",
				"bank_id":    bankID,
				"created_at": updatedAt,
			},
		},
		auditLogs: []map[string]any{
			{
				"id":         fmt.Sprintf("audit-seed-%s", bankID),
				"action":     "retain",
				"transport":  "demo",
				"status":     "completed",
				"bank_id":    bankID,
				"created_at": updatedAt,
			},
		},
		llmRequests: []map[string]any{
			{
				"id":          fmt.Sprintf("llm-seed-%s", bankID),
				"status":      "completed",
				"operation":   "reflect",
				"scope":       "demo",
				"provider":    "demo",
				"trace_id":    fmt.Sprintf("trace-seed-%s", bankID),
				"document_id": valueOrEmpty(memories[0].Item.DocumentID),
				"memory_id":   memories[0].ID,
				"created_at":  updatedAt,
			},
		},
	}
	c.banks[bankID] = bank
	c.bankOrder = append(c.bankOrder, bankID)
}

func (c *DemoClient) bank(bankID string) (*demoBank, error) {
	bank := c.banks[bankID]
	if bank == nil {
		return nil, fmt.Errorf("bank %q not found", bankID)
	}
	return bank, nil
}

func (c *DemoClient) recallResults(bank *demoBank, req domain.RecallRequest) []domain.RecallResult {
	type ranked struct {
		memory demoMemory
		score  int
	}
	rankedMemories := make([]ranked, 0, len(bank.memories))
	for _, memory := range bank.memories {
		if len(req.Types) > 0 && !contains(req.Types, memory.Type) {
			continue
		}
		if !matchesTagFilter(memory.Item.Tags, req.Tags) {
			continue
		}
		score := matchScore(memory.Item.Content+" "+valueOrEmpty(memory.Item.Context)+" "+strings.Join(memory.Item.Tags, " "), req.Query)
		if score == 0 && strings.TrimSpace(req.Query) != "" {
			continue
		}
		rankedMemories = append(rankedMemories, ranked{memory: memory, score: score})
	}
	sort.SliceStable(rankedMemories, func(i, j int) bool {
		if rankedMemories[i].score == rankedMemories[j].score {
			return rankedMemories[i].memory.CreatedAt > rankedMemories[j].memory.CreatedAt
		}
		return rankedMemories[i].score > rankedMemories[j].score
	})

	results := make([]domain.RecallResult, 0, len(rankedMemories))
	for _, item := range rankedMemories {
		memory := item.memory
		memoryType := memory.Type
		result := domain.RecallResult{
			ID:            memory.ID,
			Text:          memory.Item.Content,
			Type:          &memoryType,
			Entities:      append([]string(nil), memory.Entities...),
			Context:       cloneStringPtr(memory.Item.Context),
			OccurredStart: cloneStringPtr(memory.Item.Timestamp),
			OccurredEnd:   cloneStringPtr(memory.Item.Timestamp),
			MentionedAt:   stringPtr(memory.CreatedAt),
			DocumentID:    cloneStringPtr(memory.Item.DocumentID),
			Metadata:      cloneStringMap(memory.Item.Metadata),
			ChunkID:       stringPtr("chunk-" + memory.ID),
			Tags:          append([]string(nil), memory.Item.Tags...),
			SourceFactIDs: []string{memory.ID},
		}
		results = append(results, result)
	}
	return results
}

func (c *DemoClient) documentRows(bank *demoBank) []map[string]any {
	docs := map[string]map[string]any{}
	for _, memory := range bank.memories {
		documentID := valueOrEmpty(memory.Item.DocumentID)
		if documentID == "" {
			documentID = "doc-" + memory.ID
		}
		row := docs[documentID]
		if row == nil {
			row = map[string]any{
				"document_id":  documentID,
				"title":        "Document " + documentID,
				"tags":         []string{},
				"memory_count": 0,
				"updated_at":   memory.CreatedAt,
			}
			docs[documentID] = row
		}
		row["memory_count"] = row["memory_count"].(int) + 1
		row["tags"] = mergeStringSlices(anyToStrings(row["tags"]), memory.Item.Tags)
		if memory.CreatedAt > row["updated_at"].(string) {
			row["updated_at"] = memory.CreatedAt
		}
	}
	rows := make([]map[string]any, 0, len(docs))
	for _, row := range docs {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i]["document_id"].(string) < rows[j]["document_id"].(string)
	})
	return rows
}

func (c *DemoClient) tagRows(bank *demoBank, q string) []map[string]any {
	counts := map[string]int{}
	for _, memory := range bank.memories {
		for _, tag := range memory.Item.Tags {
			counts[tag]++
		}
	}
	rows := make([]map[string]any, 0, len(counts))
	for tag, count := range counts {
		if !matchesQuery(tag, q) {
			continue
		}
		rows = append(rows, map[string]any{"tag": tag, "count": count, "source": "memories"})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i]["tag"].(string) < rows[j]["tag"].(string)
	})
	return rows
}

func (c *DemoClient) entityRows(bank *demoBank, q string) []map[string]any {
	counts := map[string]int{}
	for _, memory := range bank.memories {
		for _, entity := range memory.Entities {
			counts[entity]++
		}
	}
	rows := make([]map[string]any, 0, len(counts))
	for entity, count := range counts {
		if !matchesQuery(entity, q) {
			continue
		}
		rows = append(rows, map[string]any{"entity": entity, "count": count})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i]["entity"].(string) < rows[j]["entity"].(string)
	})
	return rows
}

func (b *demoBank) summary() domain.BankSummary {
	name := b.profile.Name
	mission := b.profile.Mission
	createdAt := b.createdAt
	updatedAt := b.updatedAt
	lastDocAt := b.lastDocAt
	return domain.BankSummary{
		BankID:         b.profile.BankID,
		Name:           &name,
		Mission:        &mission,
		Disposition:    b.profile.Disposition,
		CreatedAt:      &createdAt,
		UpdatedAt:      &updatedAt,
		FactCount:      len(b.memories),
		LastDocumentAt: &lastDocAt,
	}
}

func applyBankRequest(bank *demoBank, req domain.CreateBankRequest) {
	if req.Name != nil {
		bank.profile.Name = *req.Name
	}
	if req.ReflectMission != nil {
		bank.profile.Mission = *req.ReflectMission
	} else if req.RetainMission != nil {
		bank.profile.Mission = *req.RetainMission
	}
	if bank.config.Overrides == nil {
		bank.config.Overrides = map[string]any{}
	}
	if req.EnableObservations != nil {
		bank.config.Overrides["enable_observations"] = *req.EnableObservations
	}
	if req.RetainExtractionMode != nil {
		bank.config.Overrides["retain_extraction_mode"] = *req.RetainExtractionMode
	}
	if req.RetainCustomInstructions != nil {
		bank.config.Overrides["retain_custom_instructions"] = *req.RetainCustomInstructions
	}
	bank.updatedAt = time.Now().UTC().Format(time.RFC3339)
}

func includeEntities(include map[string]any) bool {
	if include == nil {
		return false
	}
	_, ok := include["entities"]
	return ok
}

func includeFlag(include map[string]any, key string) bool {
	if include == nil {
		return false
	}
	value, ok := include[key]
	if !ok {
		return false
	}
	if nested, ok := value.(map[string]any); ok {
		enabled, ok := nested["enabled"].(bool)
		return !ok || enabled
	}
	flag, ok := value.(bool)
	return ok && flag
}

func inferMemoryType(item domain.MemoryItem) string {
	for _, tag := range item.Tags {
		if tag == "observation" {
			return "observation"
		}
	}
	if item.Context != nil && strings.Contains(strings.ToLower(*item.Context), "observ") {
		return "observation"
	}
	return "experience"
}

func inferEntities(content string) []string {
	words := strings.Fields(content)
	entities := make([]string, 0, 3)
	seen := map[string]struct{}{}
	for _, word := range words {
		clean := strings.Trim(word, ".,:;!?()[]{}\"'")
		if len(clean) < 4 {
			continue
		}
		clean = strings.ToLower(clean)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		entities = append(entities, clean)
		if len(entities) == 3 {
			break
		}
	}
	return entities
}

func memoryRow(memory demoMemory) map[string]any {
	return map[string]any{
		"id":              memory.ID,
		"text":            memory.Item.Content,
		"type":            memory.Type,
		"context":         valueOrEmpty(memory.Item.Context),
		"document_id":     valueOrEmpty(memory.Item.DocumentID),
		"timestamp":       valueOrEmpty(memory.Item.Timestamp),
		"metadata":        cloneStringMap(memory.Item.Metadata),
		"tags":            append([]string(nil), memory.Item.Tags...),
		"entities":        append([]string(nil), memory.Entities...),
		"mentioned_at":    memory.CreatedAt,
		"source_fact_ids": []string{memory.ID},
	}
}

func page(items []map[string]any, limit int, offset int) *domain.Page[map[string]any] {
	if offset < 0 {
		offset = 0
	}
	total := len(items)
	if limit <= 0 || offset+limit > total {
		limit = total - offset
		if limit < 0 {
			limit = 0
		}
	}
	paged := make([]map[string]any, 0, limit)
	for i := offset; i < offset+limit && i < total; i++ {
		paged = append(paged, cloneMapAny(items[i]))
	}
	return &domain.Page[map[string]any]{Items: paged, Total: total, Limit: limit, Offset: offset}
}

func matchScore(text string, query string) int {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return 1
	}
	text = strings.ToLower(text)
	score := 0
	for _, token := range strings.Fields(query) {
		if strings.Contains(text, token) {
			score++
		}
	}
	return score
}

func matchesQuery(text string, query string) bool {
	return matchScore(text, query) > 0
}

func matchesTagFilter(have []string, want []string) bool {
	if len(want) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, tag := range have {
		set[strings.ToLower(strings.TrimSpace(tag))] = struct{}{}
	}
	for _, tag := range want {
		if _, ok := set[strings.ToLower(strings.TrimSpace(tag))]; ok {
			return true
		}
	}
	return false
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func stringify(value any) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func collectEntities(results []domain.RecallResult) []string {
	set := map[string]struct{}{}
	entities := make([]string, 0)
	for _, result := range results {
		for _, entity := range result.Entities {
			if _, ok := set[entity]; ok {
				continue
			}
			set[entity] = struct{}{}
			entities = append(entities, entity)
		}
	}
	sort.Strings(entities)
	return entities
}

func firstMemoryID(results []domain.RecallResult) string {
	if len(results) == 0 {
		return ""
	}
	return results[0].ID
}

func copyMemoryItem(item domain.MemoryItem) domain.MemoryItem {
	copied := item
	copied.Context = cloneStringPtr(item.Context)
	copied.Timestamp = cloneStringPtr(item.Timestamp)
	copied.DocumentID = cloneStringPtr(item.DocumentID)
	copied.Metadata = cloneStringMap(item.Metadata)
	copied.Tags = append([]string(nil), item.Tags...)
	return copied
}

func cloneMapAny(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		switch typed := value.(type) {
		case []string:
			cloned[key] = append([]string(nil), typed...)
		case map[string]any:
			cloned[key] = cloneMapAny(typed)
		default:
			cloned[key] = typed
		}
	}
	return cloned
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func mergeStringSlices(base []string, add []string) []string {
	set := map[string]struct{}{}
	merged := make([]string, 0, len(base)+len(add))
	for _, value := range base {
		if _, ok := set[value]; ok {
			continue
		}
		set[value] = struct{}{}
		merged = append(merged, value)
	}
	for _, value := range add {
		if _, ok := set[value]; ok {
			continue
		}
		set[value] = struct{}{}
		merged = append(merged, value)
	}
	sort.Strings(merged)
	return merged
}

func anyToStrings(value any) []string {
	items, ok := value.([]string)
	if !ok {
		return nil
	}
	return append([]string(nil), items...)
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func stringPtr(value string) *string {
	return &value
}

var _ Client = (*DemoClient)(nil)
