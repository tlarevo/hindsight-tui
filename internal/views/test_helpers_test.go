package views

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/tlarevo/hindsight-tui/internal/config"
	"github.com/tlarevo/hindsight-tui/internal/domain"
	"github.com/tlarevo/hindsight-tui/internal/hindsight"
	appkeymap "github.com/tlarevo/hindsight-tui/internal/keymap"
	"github.com/tlarevo/hindsight-tui/internal/state"
)

func newTestShared() *Shared {
	cfg := config.Default()
	st := state.AppState{
		ActiveBank:  cfg.DefaultBank,
		Backend:     cfg.Backend,
		CurrentView: state.RouteDashboard,
	}
	version := &domain.VersionInfo{
		APIVersion: "0.8.1",
		Features: domain.FeatureFlags{
			Observations:  true,
			BankConfigAPI: true,
			AuditLog:      true,
			LLMTrace:      true,
		},
	}
	return &Shared{
		Config:  &cfg,
		State:   &st,
		Client:  hindsight.NewDemoClient(),
		Version: version,
		Health:  &domain.HealthStatus{OK: true, Version: version, Detail: "demo"},
		KeyMap:  appkeymap.Default(),
	}
}

type recordingClient struct {
	hindsight.Client
	recallRequest *domain.RecallRequest
	auditCalls    int
	llmCalls      int
}

func (c *recordingClient) Recall(ctx context.Context, bankID string, req domain.RecallRequest) (*domain.RecallResponse, error) {
	c.recallRequest = &req
	if c.Client != nil {
		return c.Client.Recall(ctx, bankID, req)
	}
	return &domain.RecallResponse{}, nil
}

func (c *recordingClient) ListAuditLogs(ctx context.Context, bankID string, filters hindsight.TraceFilters, limit, offset int) (*domain.Page[map[string]any], error) {
	c.auditCalls++
	if c.Client != nil {
		return c.Client.ListAuditLogs(ctx, bankID, filters, limit, offset)
	}
	return &domain.Page[map[string]any]{}, nil
}

func (c *recordingClient) ListLLMRequests(ctx context.Context, bankID string, filters hindsight.TraceFilters, limit, offset int) (*domain.Page[map[string]any], error) {
	c.llmCalls++
	if c.Client != nil {
		return c.Client.ListLLMRequests(ctx, bankID, filters, limit, offset)
	}
	return &domain.Page[map[string]any]{}, nil
}

func keyPress(text string, code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: text, Code: code})
}
