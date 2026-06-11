package hindsight

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"hindsight-tui/internal/domain"
)

const (
	defaultHTTPBaseURL = "http://127.0.0.1:8888"
	maxErrorBodyBytes  = 4 * 1024
)

type HTTPClient struct {
	baseURL   string
	authToken string
	http      *http.Client
}

func NewHTTPClient(baseURL string, timeout time.Duration, authToken string) *HTTPClient {
	trimmedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmedBaseURL == "" {
		trimmedBaseURL = defaultHTTPBaseURL
	}

	return &HTTPClient{
		baseURL:   trimmedBaseURL,
		authToken: authToken,
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *HTTPClient) Health(ctx context.Context) (*domain.HealthStatus, error) {
	var response struct {
		OK      bool                `json:"ok"`
		Version *domain.VersionInfo `json:"version"`
		Detail  string              `json:"detail"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/health", nil, nil, &response); err != nil {
		return nil, err
	}

	return &domain.HealthStatus{
		OK:      response.OK,
		Version: response.Version,
		Detail:  response.Detail,
	}, nil
}

func (c *HTTPClient) Version(ctx context.Context) (*domain.VersionInfo, error) {
	var response domain.VersionInfo
	if err := c.doJSON(ctx, http.MethodGet, "/version", nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *HTTPClient) ListBanks(ctx context.Context) ([]domain.BankSummary, error) {
	var response []domain.BankSummary
	if err := c.doJSON(ctx, http.MethodGet, "/v1/default/banks", nil, nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *HTTPClient) GetBank(ctx context.Context, bankID string) (*domain.BankProfile, error) {
	var response domain.BankProfile
	if err := c.doJSON(ctx, http.MethodGet, bankPath(bankID, "/profile"), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *HTTPClient) CreateOrUpdateBank(ctx context.Context, bankID string, req domain.CreateBankRequest) (*domain.BankProfile, error) {
	var response domain.BankProfile
	if err := c.doJSON(ctx, http.MethodPut, bankPath(bankID, ""), nil, req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *HTTPClient) PatchBank(ctx context.Context, bankID string, req domain.CreateBankRequest) (*domain.BankProfile, error) {
	var response domain.BankProfile
	if err := c.doJSON(ctx, http.MethodPatch, bankPath(bankID, ""), nil, req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *HTTPClient) DeleteBank(ctx context.Context, bankID string) error {
	return c.doJSON(ctx, http.MethodDelete, bankPath(bankID, ""), nil, nil, nil)
}

func (c *HTTPClient) GetBankConfig(ctx context.Context, bankID string) (*domain.BankConfig, error) {
	var response domain.BankConfig
	if err := c.doJSON(ctx, http.MethodGet, bankPath(bankID, "/config"), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *HTTPClient) PatchBankConfig(ctx context.Context, bankID string, updates map[string]any) (*domain.BankConfig, error) {
	var response domain.BankConfig
	body := struct {
		Updates map[string]any `json:"updates"`
	}{
		Updates: updates,
	}
	if err := c.doJSON(ctx, http.MethodPatch, bankPath(bankID, "/config"), nil, body, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *HTTPClient) BankStats(ctx context.Context, bankID string) (map[string]any, error) {
	var response map[string]any
	if err := c.doJSON(ctx, http.MethodGet, bankPath(bankID, "/stats"), nil, nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *HTTPClient) Retain(ctx context.Context, bankID string, req domain.RetainRequest) (*domain.RetainResponse, error) {
	var response domain.RetainResponse
	if err := c.doJSON(ctx, http.MethodPost, bankPath(bankID, "/memories"), nil, req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *HTTPClient) Recall(ctx context.Context, bankID string, req domain.RecallRequest) (*domain.RecallResponse, error) {
	var response domain.RecallResponse
	if err := c.doJSON(ctx, http.MethodPost, bankPath(bankID, "/memories/recall"), nil, req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *HTTPClient) Reflect(ctx context.Context, bankID string, req domain.ReflectRequest) (*domain.ReflectResponse, error) {
	var response domain.ReflectResponse
	if err := c.doJSON(ctx, http.MethodPost, bankPath(bankID, "/reflect"), nil, req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *HTTPClient) ListMemories(ctx context.Context, bankID string, q string, memoryType string, limit, offset int) (*domain.Page[map[string]any], error) {
	query := url.Values{}
	query.Set("q", q)
	query.Set("type", memoryType)
	query.Set("limit", strconv.Itoa(limit))
	query.Set("offset", strconv.Itoa(offset))
	return c.listPage(ctx, bankPath(bankID, "/memories/list"), query)
}

func (c *HTTPClient) ListDocuments(ctx context.Context, bankID string, q string, tags []string, limit, offset int) (*domain.Page[map[string]any], error) {
	query := url.Values{}
	query.Set("q", q)
	for _, tag := range tags {
		query.Add("tags", tag)
	}
	query.Set("tags_match", "any_strict")
	query.Set("limit", strconv.Itoa(limit))
	query.Set("offset", strconv.Itoa(offset))
	return c.listPage(ctx, bankPath(bankID, "/documents"), query)
}

func (c *HTTPClient) ListTags(ctx context.Context, bankID string, q string, limit, offset int) (*domain.Page[map[string]any], error) {
	query := url.Values{}
	query.Set("q", q)
	query.Set("source", "memories")
	query.Set("limit", strconv.Itoa(limit))
	query.Set("offset", strconv.Itoa(offset))
	return c.listPage(ctx, bankPath(bankID, "/tags"), query)
}

func (c *HTTPClient) ListEntities(ctx context.Context, bankID string, q string, limit, offset int) (*domain.Page[map[string]any], error) {
	query := url.Values{}
	query.Set("q", q)
	query.Set("limit", strconv.Itoa(limit))
	query.Set("offset", strconv.Itoa(offset))
	return c.listPage(ctx, bankPath(bankID, "/entities"), query)
}

func (c *HTTPClient) GetEntityGraph(ctx context.Context, bankID string, limit int) (json.RawMessage, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))
	return c.rawJSON(ctx, http.MethodGet, bankPath(bankID, "/entities/graph"), query, nil)
}

func (c *HTTPClient) GetMemoryGraph(ctx context.Context, bankID string, memoryType string, q string, tags []string, limit int) (json.RawMessage, error) {
	query := url.Values{}
	query.Set("type", memoryType)
	query.Set("q", q)
	for _, tag := range tags {
		query.Add("tags", tag)
	}
	query.Set("tags_match", "all_strict")
	query.Set("limit", strconv.Itoa(limit))
	return c.rawJSON(ctx, http.MethodGet, bankPath(bankID, "/graph"), query, nil)
}

func (c *HTTPClient) ListOperations(ctx context.Context, bankID string, status string, opType string, limit, offset int) (*domain.Page[map[string]any], error) {
	query := url.Values{}
	query.Set("status", status)
	query.Set("type", opType)
	query.Set("limit", strconv.Itoa(limit))
	query.Set("offset", strconv.Itoa(offset))
	query.Set("exclude_parents", "false")

	var response struct {
		Operations []map[string]any `json:"operations"`
		Total      int              `json:"total"`
		Limit      int              `json:"limit"`
		Offset     int              `json:"offset"`
	}
	if err := c.doJSON(ctx, http.MethodGet, bankPath(bankID, "/operations"), query, nil, &response); err != nil {
		return nil, err
	}

	return &domain.Page[map[string]any]{
		Items:  response.Operations,
		Total:  response.Total,
		Limit:  response.Limit,
		Offset: response.Offset,
	}, nil
}

func (c *HTTPClient) ListAuditLogs(ctx context.Context, bankID string, filters TraceFilters, limit, offset int) (*domain.Page[map[string]any], error) {
	query := url.Values{}
	query.Set("action", filters.Action)
	query.Set("transport", filters.Transport)
	query.Set("start_date", filters.StartDate)
	query.Set("end_date", filters.EndDate)
	query.Set("limit", strconv.Itoa(limit))
	query.Set("offset", strconv.Itoa(offset))
	return c.listPage(ctx, bankPath(bankID, "/audit-logs"), query)
}

func (c *HTTPClient) ListLLMRequests(ctx context.Context, bankID string, filters TraceFilters, limit, offset int) (*domain.Page[map[string]any], error) {
	query := url.Values{}
	query.Set("status", filters.Status)
	query.Set("operation", filters.Operation)
	query.Set("scope", filters.Scope)
	query.Set("provider", filters.Provider)
	query.Set("trace_id", filters.TraceID)
	query.Set("document_id", filters.DocumentID)
	query.Set("memory_id", filters.MemoryID)
	query.Set("group", "false")
	query.Set("start_date", filters.StartDate)
	query.Set("end_date", filters.EndDate)
	query.Set("limit", strconv.Itoa(limit))
	query.Set("offset", strconv.Itoa(offset))
	return c.listPage(ctx, bankPath(bankID, "/llm-requests"), query)
}

func (c *HTTPClient) ExportBankTemplate(ctx context.Context, bankID string) (json.RawMessage, error) {
	return c.rawJSON(ctx, http.MethodGet, bankPath(bankID, "/export"), nil, nil)
}

func (c *HTTPClient) ImportBankTemplate(ctx context.Context, bankID string, template json.RawMessage, dryRun bool) (json.RawMessage, error) {
	if len(template) == 0 {
		return nil, fmt.Errorf("import template is required")
	}
	query := url.Values{}
	query.Set("dry_run", strconv.FormatBool(dryRun))
	return c.rawJSON(ctx, http.MethodPost, bankPath(bankID, "/import"), query, template)
}

func (c *HTTPClient) listPage(ctx context.Context, path string, query url.Values) (*domain.Page[map[string]any], error) {
	var response domain.Page[map[string]any]
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *HTTPClient) rawJSON(ctx context.Context, method string, path string, query url.Values, body any) (json.RawMessage, error) {
	response, err := c.doRequest(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(response), nil
}

func (c *HTTPClient) doJSON(ctx context.Context, method string, path string, query url.Values, body any, dst any) error {
	response, err := c.doRequest(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	if dst == nil || len(response) == 0 {
		return nil
	}
	if err := json.Unmarshal(response, dst); err != nil {
		return fmt.Errorf("%s %s: decode response: %w", method, path, err)
	}
	return nil
}

func (c *HTTPClient) doRequest(ctx context.Context, method string, path string, query url.Values, body any) ([]byte, error) {
	requestURL := c.baseURL + path
	if encodedQuery := query.Encode(); encodedQuery != "" {
		requestURL += "?" + encodedQuery
	}

	var requestBody io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("%s %s: encode request: %w", method, path, err)
		}
		requestBody = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("%s %s: build request: %w", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.authToken != "" {
		if strings.Contains(c.authToken, " ") {
			req.Header.Set("Authorization", c.authToken)
		} else {
			req.Header.Set("Authorization", "Bearer "+c.authToken)
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyPrefix, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		if readErr != nil {
			return nil, fmt.Errorf("%s %s: status %d (reading body: %v)", method, path, resp.StatusCode, readErr)
		}
		return nil, &StatusError{Method: method, Path: path, Code: resp.StatusCode, Body: string(bodyPrefix)}
	}

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s %s: read response: %w", method, path, err)
	}
	return response, nil
}

func bankPath(bankID string, suffix string) string {
	return "/v1/default/banks/" + url.PathEscape(bankID) + suffix
}
