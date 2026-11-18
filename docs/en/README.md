# SmsVoid

SmsVoid is an SMPP message capture service. SMPP submissions are captured in SQLite or PostgreSQL and can be inspected through an HTTP dashboard and JSON API.

## Quick start

```sh
go build -o bin/smsvoid ./cmd/smsvoid
./bin/smsvoid
```

Defaults: SMPP listens on `:2775`, HTTP on `:8025`, and SQLite uses `smsvoid.db` in the working directory. Configure an SMPP client to bind to `localhost:2775`, then open <http://localhost:8025>.

## Configuration

| Flag | Environment variable | Default | Description |
|---|---|---|---|
| `-smpp-addr` | `SMSVOID_SMPP_ADDR` | `:2775` | SMPP TCP listen address |
| `-http-addr` | `SMSVOID_HTTP_ADDR` | `:8025` | HTTP listen address |
| `-db-driver` | `SMSVOID_DB_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `-db-dsn` | `SMSVOID_DB_DSN` | `smsvoid.db` | SQLite file path or PostgreSQL DSN |

Flags override environment variables. PostgreSQL example:

```sh
SMSVOID_DB_DRIVER=postgres SMSVOID_DB_DSN='postgres://smsvoid:secret@localhost:5432/smsvoid?sslmode=require' ./bin/smsvoid
```

SmsVoid creates the message table at startup. PostgreSQL database must already exist and the configured user must be allowed to create tables.

## SMPP behavior

Implemented PDUs: `bind_receiver`, `bind_transmitter`, `bind_transceiver`, `submit_sm`, `enquire_link`, and `unbind`. Any bind system ID is currently accepted; credentials are not authenticated. A successful `submit_sm` gets a generated ID and persists source, destination, body, data coding, sequence number, receive time, and peer address.

The current parser handles mandatory `short_message` content. It does not process `message_payload` or optional TLVs, multipart concatenation, delivery receipts, or server initiated `deliver_sm`. This is a development capture tool, not a production SMSC.

## HTTP

- `GET /`: HTML capture dashboard.
- `GET /api/v1/messages`: JSON array of at most 250 latest messages.

The HTTP interface has no authentication. Bind it to a trusted network or put it behind an authenticated proxy.

## Architecture

- `internal/smpp`: SMPP transport adapter and PDU handling.
- `internal/app`: message ID assignment and submission use case.
- `internal/storage`: SQLite/PostgreSQL persistence.
- `internal/web`: dashboard and JSON adapter.
- `internal/config`: environment and flag configuration.
- `cmd/smsvoid`: process startup and graceful shutdown.

## Development

```sh
go test ./...
go test -race ./...
go test -cover ./...
go build ./cmd/smsvoid
```

SQLite tests create disposable databases. PostgreSQL integration should be tested against an explicitly provisioned database.
