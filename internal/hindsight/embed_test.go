package hindsight

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	stdexec "os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeRunner struct {
	mu      sync.Mutex
	calls   []runnerCall
	output  []byte
	err     error
	handler func(name string, args ...string) ([]byte, error)
}

type runnerCall struct {
	name string
	args []string
}

func (r *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.mu.Lock()
	r.calls = append(r.calls, runnerCall{name: name, args: append([]string(nil), args...)})
	handler := r.handler
	output := append([]byte(nil), r.output...)
	err := r.err
	r.mu.Unlock()

	if handler != nil {
		return handler(name, args...)
	}
	return output, err
}

func (r *fakeRunner) snapshot() []runnerCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	calls := make([]runnerCall, len(r.calls))
	copy(calls, r.calls)
	return calls
}

func TestCheckInstalledMissingExecutableWrapsInstallHint(t *testing.T) {
	t.Parallel()

	manager := &EmbedManager{
		Runner: &fakeRunner{err: &stdexec.Error{Name: "hindsight-embed", Err: stdexec.ErrNotFound}},
	}

	err := manager.CheckInstalled(context.Background())
	if err == nil {
		t.Fatal("CheckInstalled() error = nil, want install hint")
	}
	if !stderrors.Is(err, ErrEmbedNotInstalled) {
		t.Fatalf("CheckInstalled() error = %v, want wrapping ErrEmbedNotInstalled", err)
	}
	if !strings.Contains(err.Error(), EmbedInstallHint()) {
		t.Fatalf("CheckInstalled() error = %q, want it to contain the install hint", err.Error())
	}
}

func TestEnsureRunningStartsDaemonThenPollsHealth(t *testing.T) {
	t.Parallel()

	var healthCalls int
	var healthMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		healthMu.Lock()
		defer healthMu.Unlock()
		healthCalls++
		if healthCalls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("warming up"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	runner := &fakeRunner{output: []byte("daemon status ok")}
	manager := &EmbedManager{Runner: runner, APIURL: server.URL, Timeout: 2 * time.Second}

	if err := manager.EnsureRunning(context.Background()); err != nil {
		t.Fatalf("EnsureRunning() error = %v", err)
	}

	calls := runner.snapshot()
	wantCalls := []runnerCall{
		{name: "hindsight-embed", args: []string{"daemon", "status"}},
		{name: "hindsight-embed", args: []string{"daemon", "start"}},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("runner calls = %#v, want %#v", calls, wantCalls)
	}

	healthMu.Lock()
	gotHealthCalls := healthCalls
	healthMu.Unlock()
	if gotHealthCalls < 3 {
		t.Fatalf("health calls = %d, want at least 3", gotHealthCalls)
	}
}

func TestEnsureRunningMemoizesHealth(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	runner := &fakeRunner{output: []byte("daemon status ok")}
	manager := &EmbedManager{Runner: runner, APIURL: server.URL, Timeout: time.Second}

	if err := manager.EnsureRunning(context.Background()); err != nil {
		t.Fatalf("EnsureRunning() first error = %v", err)
	}
	first := len(runner.snapshot())
	if err := manager.EnsureRunning(context.Background()); err != nil {
		t.Fatalf("EnsureRunning() second error = %v", err)
	}
	if second := len(runner.snapshot()); second != first {
		t.Fatalf("second EnsureRunning ran %d extra runner calls, want 0 (memoized)", second-first)
	}
	for _, c := range runner.snapshot() {
		if len(c.args) >= 2 && c.args[1] == "start" {
			t.Fatalf("unexpected daemon start call when health was immediately OK: %#v", c)
		}
	}
}

func TestEmbedClientDelegatesAfterEnsureRunning(t *testing.T) {
	t.Parallel()

	var healthCalls int
	var listBanksCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			healthCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/v1/default/banks":
			listBanksCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"bank_id":"default","name":"Default","mission":"Demo","disposition":{"skepticism":1,"literalism":2,"empathy":3},"fact_count":2}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	runner := &fakeRunner{output: []byte("running")}
	manager := &EmbedManager{Runner: runner, APIURL: server.URL, Timeout: time.Second}
	client := NewEmbedClient(manager, NewHTTPClient(server.URL, time.Second, ""))

	banks, err := client.ListBanks(context.Background())
	if err != nil {
		t.Fatalf("ListBanks() error = %v", err)
	}
	if len(banks) != 1 || banks[0].BankID != "default" {
		t.Fatalf("ListBanks() = %#v, want default bank", banks)
	}

	calls := runner.snapshot()
	wantCalls := []runnerCall{{name: "hindsight-embed", args: []string{"daemon", "status"}}}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("runner calls = %#v, want %#v", calls, wantCalls)
	}
	if healthCalls == 0 {
		t.Fatal("health endpoint was not polled")
	}
	if listBanksCalls != 1 {
		t.Fatalf("ListBanks endpoint calls = %d, want 1", listBanksCalls)
	}
}

func TestEmbedClientHealthReturnsDegradedStatusOnStartupFailure(t *testing.T) {
	t.Parallel()

	manager := &EmbedManager{
		Runner: &fakeRunner{err: &stdexec.Error{Name: "hindsight-embed", Err: stdexec.ErrNotFound}},
	}
	client := NewEmbedClient(manager, NewHTTPClient("http://127.0.0.1:8888", time.Second, ""))

	health, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if health.OK {
		t.Fatal("Health().OK = true, want false")
	}
	if !strings.Contains(health.Detail, EmbedInstallHint()) {
		t.Fatalf("Health().Detail = %q, want it to contain the install hint", health.Detail)
	}
}
