# hindsight-tui

`hindsight-tui` is a local-first terminal UI for Hindsight.

It exposes Hindsight as memory banks, retain, recall, reflect, explorer, operations, traces, and configuration instead of raw REST calls or CLI plumbing.

## Why embed is the default backend

Default backend: `hindsight-embed`.

The Hindsight embed docs describe it as a local daemon for development, prototyping, and single-user use. It auto-starts a daemon, uses embedded pg0 storage, and binds `127.0.0.1:8888`. `hindsight-tui` keeps HTTP mode first-class for bare-metal `hindsight-api`, but defaults to the local-first path.

## Prerequisites

- Go
- One Hindsight backend:
  - Embed:
    - `uvx hindsight-embed@latest configure`
    - or `pipx install hindsight-embed && hindsight-embed configure`
  - Bare metal API:
    - `pip install hindsight-api && hindsight-api`

## LLM environment variables

### Embed backend

- `HINDSIGHT_EMBED_LLM_PROVIDER`
- `HINDSIGHT_EMBED_LLM_API_KEY`
- `HINDSIGHT_EMBED_LLM_MODEL`

### HTTP API backend

- `HINDSIGHT_API_LLM_PROVIDER`
- `HINDSIGHT_API_LLM_API_KEY`
- `HINDSIGHT_API_LLM_MODEL`

## Run

```sh
go run ./cmd/hindsight-tui
go run ./cmd/hindsight-tui --demo
go run ./cmd/hindsight-tui --backend http --api-url http://localhost:8888
go run ./cmd/hindsight-tui --doctor
go run ./cmd/hindsight-tui --auth-token "$HINDSIGHT_TUI_AUTH_TOKEN"
```

## Configuration

Default config path: `~/.config/hindsight-tui/config.yaml`

See `example.config.yaml`.

The TUI stores backend selection, API URL, default bank, theme, compact mode, and timeout. It does not write provider secrets into app config. Sensitive environment values are redacted in the Config view.

## Workflows

- Create or select a bank from **Banks**. Import a bank template by setting the **Import file path** field and running the import action.
- Store new memory items from **Retain**. When a retain returns async operations, use the **View Operations** action to jump straight to **Operations**.
- Search ranked memory results from **Recall**.
- Ask grounded questions in **Reflect**.
- Inspect facts, entities, relationships, documents, and tags in **Explorer**.
- Check async indexing jobs in **Operations**.
- Inspect audit logs and LLM requests in **Traces** when the server enables them.

## Troubleshooting

### `hindsight-embed` command not found

Install and configure it:

```sh
uvx hindsight-embed@latest configure
pipx install hindsight-embed
hindsight-embed configure
```

### API is not running

- Embed mode expects `http://127.0.0.1:8888`.
- HTTP mode expects the explicit `--api-url` or configured `api_url` to be live.
- If embed startup fails, check `hindsight-embed daemon logs`.

### Missing provider key

Common embed failure: missing `HINDSIGHT_EMBED_LLM_API_KEY`.

For HTTP mode, check `HINDSIGHT_API_LLM_API_KEY` and related provider/model env vars.

### Retain succeeded but recall is empty

Indexing can complete asynchronously. Check **Operations** or retry recall after a few seconds.

### Tracing or audit tabs are empty

Those features are server-gated. If the Hindsight version reports tracing or audit disabled, the TUI will keep the app alive and show the disabled message instead of calling unsupported endpoints.

## Secret policy

- The TUI redacts API keys, tokens, passwords, secrets, and access values in its Config view.
- App config is written with `0600` permissions.
- Provider secrets stay in environment variables, not in `config.yaml`.
- Pass an API authorization token with `--auth-token` or the `HINDSIGHT_TUI_AUTH_TOKEN` environment variable. A bare token is sent as `Authorization: Bearer <token>`; a value containing a space (e.g. `Basic abc`) is sent verbatim. The token is runtime-only and is never written to `config.yaml`.
