package hindsight

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatusErrorIsUnsupportedOn404(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, time.Second, "")
	_, err := client.ListBanks(context.Background())
	if err == nil {
		t.Fatal("expected error from 404")
	}

	var se *StatusError
	if !stderrors.As(err, &se) {
		t.Fatalf("error %v is not a *StatusError", err)
	}
	if se.Code != 404 {
		t.Fatalf("StatusError.Code = %d, want 404", se.Code)
	}
	if !IsUnsupported(err) {
		t.Fatal("IsUnsupported(err) = false, want true")
	}
}

func TestImportBankTemplatePostsTemplateAndDryRun(t *testing.T) {
	t.Parallel()

	var gotMethod, gotDryRun string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotDryRun = r.URL.Query().Get("dry_run")
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"imported":true}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, time.Second, "")
	template := json.RawMessage(`{"name":"x"}`)
	if _, err := client.ImportBankTemplate(context.Background(), "default", template, true); err != nil {
		t.Fatalf("ImportBankTemplate error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if string(gotBody) != string(template) {
		t.Fatalf("body = %q, want %q", gotBody, template)
	}
	if gotDryRun != "true" {
		t.Fatalf("dry_run = %q, want true", gotDryRun)
	}
}

func TestImportBankTemplateRejectsEmptyTemplateWithoutRequest(t *testing.T) {
	t.Parallel()

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, time.Second, "")
	if _, err := client.ImportBankTemplate(context.Background(), "default", nil, false); err == nil {
		t.Fatal("expected error for empty template")
	}
	if calls != 0 {
		t.Fatalf("server received %d requests, want 0", calls)
	}
}

func TestAuthHeaderBearerPrefixing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		token string
		want  string
	}{
		{token: "abc", want: "Bearer abc"},
		{token: "Bearer xyz", want: "Bearer xyz"},
	}
	for _, tc := range cases {
		var gotAuth string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		}))
		client := NewHTTPClient(server.URL, time.Second, tc.token)
		if _, err := client.ListBanks(context.Background()); err != nil {
			server.Close()
			t.Fatalf("token %q: ListBanks error = %v", tc.token, err)
		}
		server.Close()
		if gotAuth != tc.want {
			t.Fatalf("token %q -> Authorization %q, want %q", tc.token, gotAuth, tc.want)
		}
	}
}
