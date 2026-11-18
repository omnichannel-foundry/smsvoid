# SmsVoid

SmsVoid is a small, development-oriented SMPP message capture service. It accepts SMPP submissions from any client, persists them to SQLite or PostgreSQL, and exposes them through an HTTP dashboard and a JSON API. It is useful for inspecting what an application or gateway actually sends over SMPP without standing up a full SMSC.

- Language: Go 1.24+
- Protocols: SMPP 3.4 subset (`bind_*`, `submit_sm`, `enquire_link`, `unbind`)
- Storage: SQLite (default) or PostgreSQL
- Interfaces: SMPP TCP listener + HTTP dashboard/API

See the detailed guides: [English documentation](docs/en/README.md) · [documentación en español](docs/es/README.md).

## Functionality at a glance

| Capability | Status |
|---|---|
| Accept `bind_receiver`, `bind_transmitter`, `bind_transceiver` | Implemented |
| Accept `submit_sm` and persist the submission | Implemented |
| `enquire_link` / `unbind` handling | Implemented |
| Generated message IDs (24 hex chars) | Implemented |
| Configurable simulated delivery failure | Implemented (`-fail-deliveries`) |
| Delivery receipts (success / simulated failure) | Implemented for registered deliveries |
| Multiple concurrent SMPP connections | Implemented |
| SQLite and PostgreSQL backends | Implemented |
| HTTP dashboard (`GET /`) | Implemented |
| JSON API (`GET /api/v1/messages`, last 250) | Implemented |
| Graceful shutdown on SIGINT/SIGTERM | Implemented |
| SMPP authentication / system ID validation | Implemented (optional, via `-smpp-user`/`-smpp-pass`) |
| `message_payload` TLV and optional TLVs | Implemented (payload used as body when present) |
| Multipart / concatenated messages (UDH) | **Not implemented** |
| Server-initiated `deliver_sm` (delivery receipts) | Implemented for registered deliveries |
| Inbound MO messages | **Not implemented** |
| HTTP basic authentication | Implemented (optional, via `-http-user`/`-http-pass`) |
| Pagination, filtering, search in the API | Implemented (`limit`, `offset`, `source`, `destination`, `q`) |
| Health and metrics endpoints | Implemented (`/healthz`, `/metrics`) |

SmsVoid is a capture and inspection tool for development and testing. It is **not** a production SMSC and must not be exposed to untrusted networks.

## Quick start

```sh
# Build
make build
# or: go build -o bin/smsvoid ./cmd/smsvoid

# Run with defaults (SMPP :2775, HTTP :8025, SQLite smsvoid.db)
./bin/smsvoid
```

Then point an SMPP client at `localhost:2775` and open <http://localhost:8025>.

You can also run without building:

```sh
go run ./cmd/smsvoid
```

## Configuration

Settings are read from environment variables first, then overridden by command-line flags.

| Flag | Environment variable | Default | Description |
|---|---|---|---|
| `-smpp-addr` | `SMSVOID_SMPP_ADDR` | `:2775` | SMPP TCP listen address |
| `-http-addr` | `SMSVOID_HTTP_ADDR` | `:8025` | HTTP listen address |
| `-db-driver` | `SMSVOID_DB_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `-db-dsn` | `SMSVOID_DB_DSN` | `smsvoid.db` | SQLite file path or PostgreSQL DSN |
| `-smpp-user` | `SMSVOID_SMPP_USER` | _(empty)_ | SMPP bind system id; empty disables auth |
| `-smpp-pass` | `SMSVOID_SMPP_PASS` | _(empty)_ | SMPP bind password |
| `-http-user` | `SMSVOID_HTTP_USER` | _(empty)_ | HTTP basic auth user; empty disables auth |
| `-http-pass` | `SMSVOID_HTTP_PASS` | _(empty)_ | HTTP basic auth password |
| `-fail-deliveries` | `SMSVOID_FAIL_DELIVERIES` | `false` | Simulate failed delivery receipts |

Examples:

```sh
# Custom ports
./bin/smsvoid -smpp-addr 127.0.0.1:2776 -http-addr :9000

# PostgreSQL
SMSVOID_DB_DRIVER=postgres \
SMSVOID_DB_DSN='postgres://smsvoid:secret@localhost:5432/smsvoid?sslmode=require' \
./bin/smsvoid
```

The message table is created automatically at startup. For PostgreSQL the database must already exist and the configured user must be allowed to create tables.

## SMPP behavior

Implemented PDUs:

- `bind_receiver` (1), `bind_transmitter` (2), `bind_transceiver` (9)
- `submit_sm` (4)
- `enquire_link` (0x15)
- `unbind` (6)

Any bind system ID is accepted; credentials are **not** authenticated. A successful `submit_sm` receives a generated 24-character hex ID and persists:

- message ID
- receive timestamp (UTC)
- peer (remote) address
- source address
- destination address
- body (`short_message`)
- data coding
- SMPP sequence number

### Parser notes

The parser handles the mandatory `short_message` content of `submit_sm`. It does not process:

- `message_payload` (TLV 0x0424) or other optional TLVs
- multipart / concatenated messages (UDH segmentation)
- delivery receipts initiated by the client
- server-initiated `deliver_sm` other than the simulated receipt below

### Delivery receipts

When a `submit_sm` has `registered_delivery != 0`, SmsVoid spawns a goroutine that waits 2 seconds and then writes a `deliver_sm` (command ID 5) back to the same connection containing a delivery receipt. The receipt state is:

- `DELIVRD` when `FailedSubmits` is false (default)
- `UNDELIV` with error `0x00000069` when `FailedSubmits` is true

`FailedSubmits` is an exported field on `smpp.Server`; it is intended for tests and experiments and is not wired to configuration.

## HTTP interface

- `GET /` — HTML dashboard listing captured messages. Accepts the same query filters as the API.
- `GET /api/v1/messages` — JSON array of up to 250 most recent messages (newest first).
- `GET /healthz` — JSON health check (`{"status":"ok"}`).
- `GET /metrics` — Prometheus-style text metrics (`smsvoid_messages_total`).
- Other methods on `/api/v1/messages` return `405 Method Not Allowed`.

Query parameters (all optional):

- `limit` — page size (default 250, max 1000)
- `offset` — page offset
- `source` — exact source address filter
- `destination` — exact destination address filter
- `q` — case-insensitive substring search in the message body

Example:

```sh
curl -s http://localhost:8025/api/v1/messages | jq .
```

```json
[
  {
    "ID": "a1b2c3...",
    "ReceivedAt": "2026-01-01T12:00:00Z",
    "RemoteAddr": "127.0.0.1:54321",
    "Source": "100",
    "Destination": "200",
    "Body": "hello",
    "DataCoding": 0,
    "Sequence": 7
  }
]
```

The HTTP interface has no authentication. Bind it to a trusted network or place it behind an authenticated proxy.

## Architecture

```
cmd/smsvoid          process startup, wiring, graceful shutdown
internal/config      flags + environment loading and validation
internal/smpp        SMPP transport adapter and PDU parsing
internal/app         message ID assignment and submit use case
internal/storage     SQLite/PostgreSQL persistence
internal/web         HTTP dashboard and JSON adapter
```

Dependency direction: `cmd` → `config`, `storage`, `smpp`, `web`, `app`; `smpp` → `app` (via `Handler` interface); `app` → `storage`; `web` → `storage`. Storage and SMPP do not depend on each other.

## Data model

Table `messages`:

| Column | Type (SQLite) | Type (PostgreSQL) | Notes |
|---|---|---|---|
| `id` | TEXT PK | TEXT PK | 24 hex chars |
| `received_at` | TEXT | TIMESTAMP | UTC |
| `remote_addr` | TEXT | TEXT | peer address |
| `source` | TEXT | TEXT | source_addr |
| `destination` | TEXT | TEXT | destination_addr |
| `body` | BLOB | BYTEA | raw `short_message` bytes |
| `data_coding` | INTEGER | INTEGER | SMPP data coding |
| `sequence` | INTEGER | INTEGER | SMPP sequence number |

The API exposes at most the 250 newest rows, ordered by `received_at` descending.

## Development

Build and test with `make` (see `make help` for the full list):

```sh
make build          # build into bin/smsvoid
make test           # run tests
make test-race      # run tests with the race detector
make cover          # coverage report
make vet            # go vet
make fmt            # format
make fmt-check      # fail if unformatted
make check          # fmt-check + vet + test
make tidy           # go mod tidy
make tidy-check     # fail if go.mod/go.sum are untidy
make clean          # remove build artifacts
```

Run with authentication and filtered queries:

```sh
make run ARGS='-smpp-user demo -smpp-pass secret -http-user admin -http-pass admin'
curl -u admin:admin 'http://localhost:8025/api/v1/messages?source=100&q=hello&limit=50'
curl -u admin:admin http://localhost:8025/healthz
curl -u admin:admin http://localhost:8025/metrics
```

Equivalently, without make:

```sh
go test ./...
go test -race ./...
go test -cover ./...
go build ./cmd/smsvoid
gofmt -l .
```

SQLite tests create disposable databases in temporary directories. PostgreSQL integration should be tested against an explicitly provisioned database; no PostgreSQL instance is required for the default test suite.

### Install

```sh
make install PREFIX=$HOME/.local   # installs to $HOME/.local/bin/smsvoid
make uninstall
```

## Security and operational notes

- SMPP binds are unauthenticated. Do not expose the SMPP port to untrusted networks.
- The HTTP dashboard and API have no authentication. Bind to localhost or protect with a proxy.
- Connection reads have no idle timeout; very slow or stuck clients can hold goroutines. Suitable for development use.
- SQLite writes are serialized by the driver's locking; under heavy concurrent load you may see "database is locked" errors.
- Receipt goroutines write to the originating connection; if the client disconnects before the 2-second delay, the write fails silently.

## Functionality assessment

Strengths:

- Small, readable codebase with clear package boundaries.
- Clean SMPP PDU handling with table-driven tests for parsing.
- Dual backend support with automatic schema creation.
- Optional SMPP and HTTP authentication.
- Pagination, filtering, and body search in the API.
- Health and metrics endpoints.
- Graceful shutdown and concurrent connection handling.
- Race-clean test suite.

Remaining gaps and limitations (see table above):

- Incomplete SMPP feature set (no multipart/UDH, no inbound MO delivery).
- No structured logging.
- Storage timestamps are stored as driver-native values; very old SQLite files created by other tools may not sort correctly.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

Copyright (c) 2026 Antonio Ognio.

## Runtime discovery and filesystem layout

SmsVoid resolves its installation mode from the executable location (via `os.Executable()` with symlink resolution), never from the current working directory.

Recognized locations:

- System: `/usr/bin`, `/usr/sbin`, `/usr/local/bin`, `/usr/local/sbin`
- User: `~/.local/bin`, `~/bin`
- Anything else: development mode

Conventional paths:

- System: config `/etc/smsvoid/config.toml`, data `/var/lib/smsvoid`, cache `/var/cache/smsvoid`, runtime `/run/smsvoid`, logs `/var/log/smsvoid`
- User: XDG base directories (`~/.config/smsvoid`, `~/.local/share/smsvoid`, `~/.local/state/smsvoid`, `~/.cache/smsvoid`, `$XDG_RUNTIME_DIR/smsvoid`)
- Development: config next to the executable, data/state/cache/runtime/logs under `<project>/.local/`

Configuration precedence (highest to lowest): explicit `--config` path, existing environment variables, conventional TOML file, built-in defaults. An explicit path is authoritative and errors if missing or invalid. Conventional TOML uses `config.toml`.

CLI diagnostics:

- `--info` — print effective paths (read-only)
- `--doctor` — check paths, files, and permissions (read-only)
- `--init` — create a default config file without overwriting
- `--version` — print version
- `--config /path/to/config.toml` — explicit configuration override
