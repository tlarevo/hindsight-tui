package hindsight

import (
	"context"
	"net/http"

	"github.com/tlarevo/hindsight-tui/internal/domain"
)

// ensureTransport guarantees the embed daemon is running before each request is
// dispatched. EnsureRunning is memoized, so the common case is a cheap mutex
// check rather than a subprocess fork.
type ensureTransport struct {
	manager *EmbedManager
	base    http.RoundTripper
}

func (t *ensureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.manager.EnsureRunning(req.Context()); err != nil {
		return nil, err
	}
	return t.base.RoundTrip(req)
}

// EmbedClient is an HTTPClient whose transport starts the embed daemon on
// demand. All API methods are inherited from the embedded *HTTPClient; only
// Health is overridden to degrade gracefully when the daemon cannot be started.
type EmbedClient struct {
	*HTTPClient
	manager *EmbedManager
}

func NewEmbedClient(manager *EmbedManager, httpClient *HTTPClient) *EmbedClient {
	httpClient.http.Transport = &ensureTransport{manager: manager, base: http.DefaultTransport}
	return &EmbedClient{HTTPClient: httpClient, manager: manager}
}

// Health never fails: a failure to reach (or start) the daemon is reported as a
// not-OK status carrying the underlying error detail, so the dashboard can show
// the problem instead of aborting startup.
func (c *EmbedClient) Health(ctx context.Context) (*domain.HealthStatus, error) {
	health, err := c.HTTPClient.Health(ctx)
	if err != nil {
		return &domain.HealthStatus{OK: false, Detail: err.Error()}, nil
	}
	return health, nil
}

var _ Client = (*EmbedClient)(nil)
