package errors

import (
	"context"
	"fmt"
	"testing"

	"hindsight-tui/internal/hindsight"
)

func TestFriendlyClassifiesStatusErrors(t *testing.T) {
	t.Parallel()

	cases := map[int]string{
		401: "Authorization failed",
		403: "Authorization failed",
		404: "Endpoint is unavailable",
		405: "Endpoint is unavailable",
		400: "Input needs attention",
		422: "Input needs attention",
	}
	for code, want := range cases {
		got := Friendly(&hindsight.StatusError{Code: code}).Title
		if got != want {
			t.Fatalf("status %d -> %q, want %q", code, got, want)
		}
	}
}

func TestFriendlyClassifiesTimeout(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("recall: %w", context.DeadlineExceeded)
	if got := Friendly(err).Title; got != "Request timed out" {
		t.Fatalf("timeout -> %q, want Request timed out", got)
	}
}
