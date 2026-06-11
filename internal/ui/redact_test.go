package ui

import (
	"strings"
	"testing"
)

func TestRedactEnvValueMasksSensitiveValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		key   string
		value string
		want  string
	}{
		{key: "HINDSIGHT_EMBED_LLM_API_KEY", value: "secret-value", want: "set"},
		{key: "ACCESS_TOKEN", value: "token", want: "set"},
		{key: "PASSWORD_HINT", value: "present", want: "set"},
		{key: "HINDSIGHT_API_LLM_API_KEY", value: "", want: "unset"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.key, func(t *testing.T) {
			t.Parallel()

			got := RedactEnvValue(tc.key, tc.value)
			if got != tc.want {
				t.Fatalf("RedactEnvValue(%q, %q) = %q, want %q", tc.key, tc.value, got, tc.want)
			}
			if strings.Contains(got, tc.value) && tc.value != "" {
				t.Fatalf("RedactEnvValue(%q, %q) leaked input value", tc.key, tc.value)
			}
		})
	}
}
