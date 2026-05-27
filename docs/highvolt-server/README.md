# highvolt-server

`highvolt-server` is the central analysis engine. It exposes a REST API over HTTP or HTTPS, accepts file submissions from clients, queues them for asynchronous LLM analysis, and stores results in OpenSearch.

## Source layout

```
cmd/highvolt-server/
├── main.go          — startup, HTTP routing, TLS, shutdown
├── submit.go        — POST /submit handler
├── query.go         — POST /query handler
├── jwt.go           — authentication token issuance and middleware
├── ratelimit.go     — per-IP rate limiting for /auth/token
├── haproxy.go       — optional HAProxy TCP health agent
├── reload.go        — config/debug reload monitor
├── signal.go        — OS signal handler
├── log.go           — HTTP request logger middleware
├── auth/
│   └── jwt_pat.go   — PAT → JWT exchange for JSONAir
├── config/
│   ├── config.go    — config fetch, parse, and hot-reload
│   └── env.go       — environment variable loading
├── db/
│   └── opensearch.go — OpenSearch index and search
├── debug/
│   └── debug.go     — debug flag management
├── helpers/
│   ├── helpers.go   — MIME type classification
│   └── sanity.go    — JSON payload validation
├── llm/
│   └── llm.go       — LLM submission (OpenAI-compatible)
├── models/
│   ├── config.go    — config structs
│   └── env.go       — environment structs
├── processors/
│   ├── submit_data.go — dispatch to per-type processor
│   ├── pdf.go         — PDF → PNG → LLM
│   ├── office.go      — Office → PDF → PNG → LLM
│   └── archive.go     — archive extraction and per-file analysis
└── queue/
    ├── queue.go     — in-memory bounded channel queue
    ├── worker.go    — goroutine worker pool
    └── dowork.go    — per-item work dispatch
```

## Startup sequence

1. Logger initialized to `local` syslog.
2. Environment variables loaded (needed to reach JSONAir).
3. PAT exchanged for a JSONAir JWT; configuration blob fetched.
4. Background goroutines started: config monitor, signal handler.
5. Logger reconfigured to user-specified syslog target.
6. Debug level fetched from JSONAir.
7. OpenSearch client initialized.
8. Worker pool started.
9. HAProxy TCP agent started (if enabled).
10. HTTP server bound (drops Unix privileges after bind).
