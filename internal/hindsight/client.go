package hindsight

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tlarevo/hindsight-tui/internal/config"
	"github.com/tlarevo/hindsight-tui/internal/domain"
)

type TraceFilters struct {
	Action     string
	Transport  string
	Status     string
	Operation  string
	Scope      string
	Provider   string
	TraceID    string
	DocumentID string
	MemoryID   string
	StartDate  string
	EndDate    string
}

type Client interface {
	Health(ctx context.Context) (*domain.HealthStatus, error)
	Version(ctx context.Context) (*domain.VersionInfo, error)

	ListBanks(ctx context.Context) ([]domain.BankSummary, error)
	GetBank(ctx context.Context, bankID string) (*domain.BankProfile, error)
	CreateOrUpdateBank(ctx context.Context, bankID string, req domain.CreateBankRequest) (*domain.BankProfile, error)
	PatchBank(ctx context.Context, bankID string, req domain.CreateBankRequest) (*domain.BankProfile, error)
	DeleteBank(ctx context.Context, bankID string) error
	GetBankConfig(ctx context.Context, bankID string) (*domain.BankConfig, error)
	PatchBankConfig(ctx context.Context, bankID string, updates map[string]any) (*domain.BankConfig, error)
	BankStats(ctx context.Context, bankID string) (map[string]any, error)

	Retain(ctx context.Context, bankID string, req domain.RetainRequest) (*domain.RetainResponse, error)
	Recall(ctx context.Context, bankID string, req domain.RecallRequest) (*domain.RecallResponse, error)
	Reflect(ctx context.Context, bankID string, req domain.ReflectRequest) (*domain.ReflectResponse, error)

	ListMemories(ctx context.Context, bankID string, q string, memoryType string, limit, offset int) (*domain.Page[map[string]any], error)
	ListDocuments(ctx context.Context, bankID string, q string, tags []string, limit, offset int) (*domain.Page[map[string]any], error)
	ListTags(ctx context.Context, bankID string, q string, limit, offset int) (*domain.Page[map[string]any], error)
	ListEntities(ctx context.Context, bankID string, q string, limit, offset int) (*domain.Page[map[string]any], error)
	GetEntityGraph(ctx context.Context, bankID string, limit int) (json.RawMessage, error)
	GetMemoryGraph(ctx context.Context, bankID string, memoryType string, q string, tags []string, limit int) (json.RawMessage, error)
	ListOperations(ctx context.Context, bankID string, status string, opType string, limit, offset int) (*domain.Page[map[string]any], error)
	ListAuditLogs(ctx context.Context, bankID string, filters TraceFilters, limit, offset int) (*domain.Page[map[string]any], error)
	ListLLMRequests(ctx context.Context, bankID string, filters TraceFilters, limit, offset int) (*domain.Page[map[string]any], error)
	ExportBankTemplate(ctx context.Context, bankID string) (json.RawMessage, error)
	ImportBankTemplate(ctx context.Context, bankID string, template json.RawMessage, dryRun bool) (json.RawMessage, error)
}

func NewFromConfig(cfg config.Config) (Client, *EmbedManager) {
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	httpClient := NewHTTPClient(cfg.APIURL, timeout, cfg.AuthToken)

	switch cfg.Backend {
	case config.BackendDemo:
		return NewDemoClient(), nil
	case config.BackendHTTP:
		return httpClient, nil
	case config.BackendEmbed:
		fallthrough
	default:
		manager := &EmbedManager{APIURL: cfg.APIURL, Timeout: timeout}
		return NewEmbedClient(manager, httpClient), manager
	}
}
