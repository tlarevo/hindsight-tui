package views

import (
	"testing"

	"hindsight-tui/internal/domain"
)

func TestTracesViewSkipsEndpointsWhenFeaturesDisabled(t *testing.T) {
	t.Parallel()

	shared := newTestShared()
	client := &recordingClient{Client: shared.Client}
	shared.Client = client
	shared.Version = &domain.VersionInfo{APIVersion: "0.8.1"}

	view := NewTracesView(shared)
	if cmd := view.loadGateCmd(); cmd != nil {
		t.Fatal("loadGateCmd() returned command for disabled audit logs")
	}
	if client.auditCalls != 0 || client.llmCalls != 0 {
		t.Fatalf("calls = audit:%d llm:%d, want 0/0", client.auditCalls, client.llmCalls)
	}
	if view.notice != "Audit logging is disabled on this Hindsight server." {
		t.Fatalf("notice = %q", view.notice)
	}

	view.tab = tracesLLM
	if cmd := view.loadGateCmd(); cmd != nil {
		t.Fatal("loadGateCmd() returned command for disabled llm tracing")
	}
	if client.auditCalls != 0 || client.llmCalls != 0 {
		t.Fatalf("calls = audit:%d llm:%d, want 0/0", client.auditCalls, client.llmCalls)
	}
	if view.notice != "LLM tracing is disabled on this Hindsight server." {
		t.Fatalf("notice = %q", view.notice)
	}
}
