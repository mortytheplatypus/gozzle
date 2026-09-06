# gozzle

HTTP log pipe in Go. Register a `service_id`, append NDJSON (Newline-Delimited JSON) lines to a per-service WAL, read back by inclusive ID offset. Ingest is async per service (bounded queue + worker); durable before `200`.

```
POST /api/services  →  services.json + {id}.log
POST /api/logs      →  per-service queue → worker → {id}.log
GET  /api/logs      →  scan {id}.log (ID >= offset)
```

Module: `github.com/mortytheplatypus/gozzle`

## Status

Done through **ingest backpressure** (per-service workers, `503` when that service’s queue is full).

**Next:** graceful shutdown (SIGTERM → drain queues → close FDs). Auth not implemented.

## Run

```bash
cp .env.example .env
go run ./cmd/gozzle
```

## Config

| Env | Default | Notes |
|-----|---------|--------|
| `GOZZLE_LISTEN` | `:8080` | `.env.example` uses `:9090` |
| `GOZZLE_WAL_DIR` | `./data/wal` | WAL + registry directory |
| `GOZZLE_SERVICES_FILE` | `services.json` | Filename only; joined under `WAL_DIR` |
| `GOZZLE_INGEST_QUEUE` | `256` | Per-`service_id` channel capacity |

## API

| Method | Path | Notes |
|--------|------|--------|
| `GET` | `/api/health` | `{"status":"ok"}` |
| `POST` | `/api/services` | Body `{"service_id":"..."}`; idempotent |
| `POST` | `/api/logs` | Body `{"service_id","message","level"}`; `200` after WAL write |
| `GET` | `/api/logs` | Query `service_id` (required), `offset` (default `0`, inclusive) |

**`service_id`:** `^[a-z][a-z0-9_-]{0,63}$`. Reserved: `services` → `400`.

| Condition | Status |
|-----------|--------|
| Bad JSON / bad `service_id` / bad `offset` / missing `service_id` on GET | `400` |
| Not registered | `404` |
| Ingest queue full (that service) | `503` |
| Store/internal failure | `500` |

### curl

```bash
curl -X POST localhost:9090/api/services \
  -H 'Content-Type: application/json' \
  -d '{"service_id":"foo"}'

curl -X POST localhost:9090/api/logs \
  -H 'Content-Type: application/json' \
  -d '{"service_id":"foo","level":"INFO","message":"hi"}'

curl 'localhost:9090/api/logs?service_id=foo&offset=1'
```

## Storage

Under `GOZZLE_WAL_DIR`:

```
services.json          # registry + per-service next_id
{service_id}.log       # NDJSON LogEntry lines
```

Example `services.json`:

```json
{"services":{"foo":{"next_id":4},"bar":{"next_id":2}}}
```

### `next_id`

- Scoped **per `service_id`** (foo and bar both can have `ID: 1`).
- Value in the registry = **next ID to assign** (after writing IDs 1..3, `next_id` is `4`).
- On append: assign `ID = next_id`, increment, **persist `services.json`, then** append the NDJSON line.
- Crash between persist and WAL line → **ID gap** (OK). IDs are never reused.
- Read `offset` is inclusive: keep lines with `ID >= offset`.

### Ingest workers

- On **register**, gozzle starts **one goroutine + bounded channel** for that `service_id` (capacity `GOZZLE_INGEST_QUEUE`).
- `POST /api/logs` does a **non-blocking** enqueue to that service’s channel:
  - full → `503` (`ErrBusy`) — backpressure is **per service**, not global
  - accepted → HTTP goroutine **blocks on a per-job result chan** until the worker finishes `Store.Append` (`200` ⇒ durable)
- Worker loop: receive job → `Store.Append` → send error/nil on `result`.
- Different services = different workers (can progress in parallel). Same service = serial writes.
- `Store` mutex protects the registry map / `services.json` when workers run concurrently.
- Write FDs for `{id}.log` are lazy-opened on first append; closed when all workers drain on `Close`.

## Layout

```
cmd/gozzle/            # main
internal/config/       # env
internal/log/          # handler → service (per-service queues/workers) → store
internal/server/       # mux + wiring
```
