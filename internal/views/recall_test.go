package views

import "testing"

func TestRecallSubmitBuildsExplicitTypesAndIncludeFields(t *testing.T) {
	t.Parallel()

	shared := newTestShared()
	client := &recordingClient{Client: shared.Client}
	shared.Client = client

	view := NewRecallView(shared)
	view.query.SetValue("user preferences")
	view.bank.SetValue("default")
	view.observation = true
	view.includeChunks = true
	view.includeSources = true
	view.trace = true

	cmd := view.submit()
	if cmd == nil {
		t.Fatal("submit() returned nil command")
	}
	if _, ok := cmd().(recallSubmittedMsg); !ok {
		t.Fatal("submit() command did not return recallSubmittedMsg")
	}
	if client.recallRequest == nil {
		t.Fatal("recall request was not captured")
	}

	got := client.recallRequest
	if len(got.Types) != 3 || got.Types[0] != "world" || got.Types[1] != "experience" || got.Types[2] != "observation" {
		t.Fatalf("types = %#v, want [world experience observation]", got.Types)
	}
	if got.Budget != "mid" || got.MaxTokens != 4096 || !got.Trace {
		t.Fatalf("request = %#v", got)
	}
	entities, ok := got.Include["entities"].(map[string]any)
	if !ok || entities["max_tokens"] != 500 {
		t.Fatalf("entities include = %#v", got.Include["entities"])
	}
	if got.Include["chunks"].(map[string]any)["enabled"] != true {
		t.Fatalf("chunks include = %#v", got.Include["chunks"])
	}
	if got.Include["source_facts"].(map[string]any)["enabled"] != true {
		t.Fatalf("source_facts include = %#v", got.Include["source_facts"])
	}
}
