package hindsight

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tlarevo/hindsight-tui/internal/domain"
)

var _ Client = (*HTTPClient)(nil)

func TestHTTPClientHealthAndVersion(t *testing.T) {
	t.Parallel()

	var healthAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			healthAuth = r.Header.Get("Authorization")
			writeJSON(t, w, map[string]any{
				"ok": true,
				"version": map[string]any{
					"api_version": "0.8.1",
					"features": map[string]any{
						"observations": true,
					},
				},
				"detail": "healthy",
			})
		case "/version":
			if got := r.Header.Get("Authorization"); got != "Bearer token" {
				t.Fatalf("version authorization header = %q", got)
			}
			writeJSON(t, w, map[string]any{
				"api_version": "0.8.1",
				"features": map[string]any{
					"observations": true,
					"llm_trace":    true,
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL+"/", 2*time.Second, "Bearer token")

	health, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health error: %v", err)
	}
	if !health.OK {
		t.Fatalf("Health OK = false")
	}
	if health.Version == nil || health.Version.APIVersion != "0.8.1" {
		t.Fatalf("Health version = %#v", health.Version)
	}
	if health.Detail != "healthy" {
		t.Fatalf("Health detail = %q", health.Detail)
	}
	if healthAuth != "Bearer token" {
		t.Fatalf("health authorization header = %q", healthAuth)
	}

	version, err := client.Version(context.Background())
	if err != nil {
		t.Fatalf("Version error: %v", err)
	}
	if version.APIVersion != "0.8.1" {
		t.Fatalf("Version APIVersion = %q", version.APIVersion)
	}
	if !version.Features.Observations || !version.Features.LLMTrace {
		t.Fatalf("Version features = %#v", version.Features)
	}
}

func TestHTTPClientRetainPostsOneItemWithoutDeprecatedFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/default/banks/default/memories" {
			t.Fatalf("path = %s", r.URL.Path)
		}

		var body map[string]any
		decodeJSON(t, r, &body)
		if _, ok := body["document_tags"]; ok {
			t.Fatalf("deprecated document_tags was sent: %#v", body)
		}

		items, ok := body["items"].([]any)
		if !ok {
			t.Fatalf("items = %#v", body["items"])
		}
		if len(items) != 1 {
			t.Fatalf("items len = %d", len(items))
		}

		item, ok := items[0].(map[string]any)
		if !ok {
			t.Fatalf("item = %#v", items[0])
		}
		if item["content"] != "stored memory" {
			t.Fatalf("content = %#v", item["content"])
		}

		writeJSON(t, w, map[string]any{
			"success":       true,
			"bank_id":       "default",
			"items_count":   1,
			"async":         false,
			"operation_ids": []string{},
		})
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, time.Second, "")
	response, err := client.Retain(context.Background(), "default", domain.RetainRequest{
		Items: []domain.MemoryItem{{Content: "stored memory"}},
		Async: false,
	})
	if err != nil {
		t.Fatalf("Retain error: %v", err)
	}
	if response.ItemsCount != 1 || response.BankID != "default" {
		t.Fatalf("Retain response = %#v", response)
	}
}

func TestHTTPClientRecallPostsExplicitTypes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/default/banks/default/memories/recall" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}

		var body map[string]any
		decodeJSON(t, r, &body)
		types, ok := body["types"].([]any)
		if !ok {
			t.Fatalf("types = %#v", body["types"])
		}
		if len(types) != 2 || types[0] != "world" || types[1] != "experience" {
			t.Fatalf("types = %#v", types)
		}

		writeJSON(t, w, map[string]any{
			"results":      []any{},
			"trace":        map[string]any{},
			"entities":     map[string]any{},
			"chunks":       map[string]any{},
			"source_facts": map[string]any{},
		})
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, time.Second, "")
	response, err := client.Recall(context.Background(), "default", domain.RecallRequest{
		Query:  "user preferences",
		Types:  []string{"world", "experience"},
		Budget: "mid",
	})
	if err != nil {
		t.Fatalf("Recall error: %v", err)
	}
	if len(response.Results) != 0 {
		t.Fatalf("Recall results = %#v", response.Results)
	}
}

func TestHTTPClientListAndGraphEndpointsEscapeValues(t *testing.T) {
	t.Parallel()

	bankID := "bank/one"
	entityQuery := url.Values{}
	entityQuery.Set("q", "name/with spaces & symbols?")
	entityQuery.Set("limit", "25")
	entityQuery.Set("offset", "5")

	memoryQuery := url.Values{}
	memoryQuery.Set("type", "world/state")
	memoryQuery.Set("q", "alpha & beta/?")
	memoryQuery.Add("tags", "tag one")
	memoryQuery.Add("tags", "tag/two")
	memoryQuery.Set("tags_match", "all_strict")
	memoryQuery.Set("limit", "9")

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch requestCount {
		case 0:
			if r.Method != http.MethodGet {
				t.Fatalf("ListEntities method = %s", r.Method)
			}
			if r.URL.EscapedPath() != "/v1/default/banks/bank%2Fone/entities" {
				t.Fatalf("ListEntities path = %s", r.URL.EscapedPath())
			}
			if r.URL.RawQuery != entityQuery.Encode() {
				t.Fatalf("ListEntities raw query = %q", r.URL.RawQuery)
			}
			writeJSON(t, w, map[string]any{
				"items":  []any{map[string]any{"id": "entity-1"}},
				"total":  1,
				"limit":  25,
				"offset": 5,
			})
		case 1:
			if r.URL.EscapedPath() != "/v1/default/banks/bank%2Fone/entities/graph" {
				t.Fatalf("GetEntityGraph path = %s", r.URL.EscapedPath())
			}
			if r.URL.RawQuery != "limit=7" {
				t.Fatalf("GetEntityGraph raw query = %q", r.URL.RawQuery)
			}
			writeRawJSON(t, w, `{"nodes":["entity-1"]}`)
		case 2:
			if r.URL.EscapedPath() != "/v1/default/banks/bank%2Fone/graph" {
				t.Fatalf("GetMemoryGraph path = %s", r.URL.EscapedPath())
			}
			if r.URL.RawQuery != memoryQuery.Encode() {
				t.Fatalf("GetMemoryGraph raw query = %q", r.URL.RawQuery)
			}
			writeRawJSON(t, w, `{"edges":["memory-1"]}`)
		default:
			t.Fatalf("unexpected request %d: %s", requestCount, r.URL.String())
		}
		requestCount++
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, time.Second, "")

	entities, err := client.ListEntities(context.Background(), bankID, "name/with spaces & symbols?", 25, 5)
	if err != nil {
		t.Fatalf("ListEntities error: %v", err)
	}
	if entities.Total != 1 || len(entities.Items) != 1 {
		t.Fatalf("ListEntities response = %#v", entities)
	}

	entityGraph, err := client.GetEntityGraph(context.Background(), bankID, 7)
	if err != nil {
		t.Fatalf("GetEntityGraph error: %v", err)
	}
	if string(entityGraph) != `{"nodes":["entity-1"]}` {
		t.Fatalf("GetEntityGraph response = %s", string(entityGraph))
	}

	memoryGraph, err := client.GetMemoryGraph(context.Background(), bankID, "world/state", "alpha & beta/?", []string{"tag one", "tag/two"}, 9)
	if err != nil {
		t.Fatalf("GetMemoryGraph error: %v", err)
	}
	if string(memoryGraph) != `{"edges":["memory-1"]}` {
		t.Fatalf("GetMemoryGraph response = %s", string(memoryGraph))
	}

	if requestCount != 3 {
		t.Fatalf("requestCount = %d", requestCount)
	}
}

func TestHTTPClientReflectOmitsDeprecatedContextField(t *testing.T) {
	t.Parallel()

	query := "Question:\nWhat should I do?\n\nContext:\nUse the recalled memory."
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/default/banks/default/reflect" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}

		var body map[string]any
		decodeJSON(t, r, &body)
		if body["query"] != query {
			t.Fatalf("query = %#v", body["query"])
		}
		if _, ok := body["context"]; ok {
			t.Fatalf("deprecated context field was sent: %#v", body)
		}

		writeJSON(t, w, map[string]any{
			"text":              "Grounded answer",
			"based_on":          map[string]any{},
			"structured_output": map[string]any{},
			"trace":             map[string]any{},
		})
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, time.Second, "")
	response, err := client.Reflect(context.Background(), "default", domain.ReflectRequest{Query: query})
	if err != nil {
		t.Fatalf("Reflect error: %v", err)
	}
	if response.Text != "Grounded answer" {
		t.Fatalf("Reflect response = %#v", response)
	}
}

func TestHTTPClientNon2xxIncludesMethodPathStatusAndBodyPrefix(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("x", 5000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, time.Second, "")
	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	errText := err.Error()
	if !strings.Contains(errText, "GET /health") {
		t.Fatalf("error missing method/path: %q", errText)
	}
	if !strings.Contains(errText, "status 502") {
		t.Fatalf("error missing status: %q", errText)
	}
	prefix := body[:4096]
	if !strings.Contains(errText, prefix) {
		t.Fatalf("error missing body prefix")
	}
	if strings.Contains(errText, body) {
		t.Fatalf("error included more than 4KiB of body")
	}
}

func decodeJSON(t *testing.T, r *http.Request, dst any) {
	t.Helper()
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		t.Fatalf("decode request: %v", err)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func writeRawJSON(t *testing.T, w http.ResponseWriter, value string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write([]byte(value))
	if err != nil {
		t.Fatalf("write response: %v", err)
	}
}
