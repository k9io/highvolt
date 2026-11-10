# Architecture Overview

## System Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                              Data Sources                                │
│                                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────────────────┐    │
│  │ voltage  │  │ suricata │  │  aws-s3  │  │    aws-s3-lambda       │    │
│  │(endpoint)│  │(network) │  │  (batch) │  │  (real-time / Lambda)  │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────────┬─────────────┘    │
└───────┼─────────────┼─────────────┼───────────────────┼──────────────────┘
        │             │             │                   │
        │        ┌────┴─────┐       │           S3 event → CloudTrail
        │        │  Redis   │       │           → EventBridge (StackSet)
        │        │ Pub/Sub  │       │           → Central event bus
        │        └────┬─────┘       │                   │
        │             │             │          STS AssumeRole (member acct)
        │             │             │          S3 GetObject (cross-account)
        └─────────────┴─────────────┴───────────────────┘
                               │
                               │  HTTPS (JWT)
                               ▼
                   ┌────────────────────────────────┐
                   │      highvolt-server           │
                   │                                │
                   │  POST /submit  ──►  Queue      │
                   │  POST /query   ──►  Storage    │
                   │                                │
                   │       Workers                  │
                   │     (goroutines)               │
                   └──────────┬─────────────────────┘
                              │
             ┌────────────────┼────────────────┐
             │                │                │
             ▼                ▼                ▼
       ┌──────────┐    ┌─────────────┐  ┌──────────────┐
       │   LLM    │    │   Storage   │  │  JSONAir     │
       │(OpenAI   │    │ OpenObserve │  │(config store)│
       │ compat.) │    │ and/or      │  │              │
       │          │    │ OpenSearch  │  │              │
       └──────────┘    └─────────────┘  └──────────────┘
```

## Storage Backends

Results can be stored in **OpenObserve** (preferred) and/or **OpenSearch**. Two independent settings control this (see [Configuration](highvolt-server/configuration.md#core)):

- `core.write_backends` — every backend listed receives every analyzed document. Listing both enables dual-writing, e.g. during a migration.
- `core.query_backend` — the single backend that answers `/query` (deduplication) lookups.

See [OpenObserve Integration](highvolt-server/openobserve.md) and [OpenSearch Integration](highvolt-server/opensearch.md) for backend-specific behavior.

## Configuration Management (JSONAir)

All components fetch their runtime configuration from **JSONAir**, an external configuration service. This separates configuration from deployment. On startup each component:

1. Authenticates to JSONAir using a Personal Access Token (PAT) from the environment.
2. Downloads its JSON configuration blob.
3. Starts a background goroutine (`Monitor_Config`) that polls JSONAir on a configurable interval and hot-reloads if the config changes.

## Request Lifecycle

1. A client (Voltage, Suricata, AWS-S3) discovers a file matching a configured MIME type.
2. The client computes MD5, SHA1, and SHA256 hashes of the file.
3. The client **queries** `highvolt-server` with the SHA256 — if the file has already been analyzed, it is skipped.
4. The client base64-encodes the file and **submits** it to `highvolt-server` as a JSON payload.
5. `highvolt-server` validates the payload, places it on an internal in-memory queue, and returns immediately (async).
6. A worker goroutine dequeues the work, classifies the MIME type, and routes it to the appropriate processor (IMAGE, TEXT, PDF, OFFICE, or ARCHIVE).
7. The processor sends the file to the LLM via an OpenAI-compatible `POST /chat/completions` request.
8. The LLM returns a JSON verdict. The worker merges the verdict into the original submission JSON and indexes it to every backend in `write_backends` (OpenObserve and/or OpenSearch). OpenSearch uses the SHA256 as the document ID; OpenObserve appends it as a new record in the configured stream.
9. The raw `file_data` field is stripped before storage — only metadata and LLM results are saved.

## Deduplication

Most clients maintain a local **GOB binary registry** (a `map[string]bool` serialized to disk) keyed by SHA256 hash (Voltage) or S3 path (aws-s3). Before submitting a file, the client checks this local cache first, then confirms with the server via the `/query` endpoint, which is answered by the configured `query_backend`. This two-layer approach avoids re-downloading large files unnecessarily.

The `aws-s3-lambda` client is stateless and has no local registry. It relies solely on the server-side SHA256 query for deduplication. Because the Lambda is only invoked for newly created objects, re-processing existing objects is not a concern.

## Concurrency Model

`highvolt-server` uses a bounded worker pool. The pool size (`max_workers`) and queue depth (`max_queue_size`) are both configurable. Submissions that arrive when the queue is full are rejected with HTTP 503.

## Authentication Flow

```
Client                     highvolt-server
  │                               │
  │  POST /auth/token {token:PAT} │
  ├──────────────────────────────►│
  │                               │  constant-time compare
  │  {access_token: JWT, ...}     │  JWT signed w/ HMAC-SHA256
  │◄──────────────────────────────│
  │                               │
  │  POST /submit  Bearer <JWT>   │
  ├──────────────────────────────►│
  │                               │  jwtMiddleware validates
  │  200 OK                       │
  │◄──────────────────────────────│
```

JWT tokens are short-lived (configurable expiry). Clients detect 401 responses and automatically re-authenticate.
