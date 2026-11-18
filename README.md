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
| Delivery receipts (success / simulated failure) | Implemented for registered deliveries |
| Multiple concurrent SMPP connections | Implemented |
| SQLite and PostgreSQL backends | Implemented |
| HTTP dashboard (`GET /`) | Implemented |
| JSON API (`GET /api/v1/messages`, last 250) | Implemented |
| Graceful shutdown on SIGINT/SIGTERM | Implemented |
| SMPP authentication / system ID validation | **Not implemented** (any bind is accepted) |
| `message_payload` TLV, optional TLVs, multipart concatenation | **Not implemented** |
| Server-initiated `deliver_sm`, inbound MO messages | **Not implemented** |
| HTTP authentication / authorization | **Not implemented** |
| Pagination, filtering, search in the API | **Not implemented** |
| Metrics / health endpoint | **Not implemented** |

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

- `GET /` — HTML dashboard listing captured messages.
- `GET /api/v1/messages` — JSON array of up to 250 most recent messages (newest first).
- Other methods on `/api/v1/messages` return `405 Method Not Allowed`.

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
- Graceful shutdown and concurrent connection handling.
- Race-clean test suite.

Known gaps and limitations (see table above):

- No authentication on either SMPP or HTTP.
- Incomplete SMPP feature set (no TLVs, no multipart, no MO delivery).
- No pagination or filtering in the API.
- No observability (metrics, health check, structured logging).
- `FailedSubmits` is not configurable.
- Storage timestamps are stored as driver-native values; very old SQLite files created by other tools may not sort correctly.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

Copyright (c) 2026 Antonio Ognio.
