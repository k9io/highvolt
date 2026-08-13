# OpenObserve Integration

OpenObserve is an alternative storage backend to OpenSearch. It can be used for **indexing** analysis results, for **querying** by SHA256 to detect duplicates, or both — controlled independently via `write_backends` and `query_backend` (see [Configuration](configuration.md#core)).

## Connection

The OpenObserve client is initialized once at startup with credentials from the server configuration:

```json
{
  "openobserve": {
    "url":             "https://openobserve.internal:5080",
    "username":        "highvolt",
    "password":        "...",
    "organization":    "default",
    "stream":          "highvolt",
    "tls_skip_verify": false
  }
}
```

`tls_skip_verify: true` disables TLS certificate validation. A warning is logged at startup if this is set. Do not use in production.

## Indexing a document

After LLM analysis, the worker calls `db.Index_Document(sha256, jsondata)`. Before indexing, the `file_data` field is **deleted** from the JSON — raw file contents are never stored in OpenObserve.

The record is appended to the configured stream via OpenObserve's native JSON ingestion API (`POST /api/{organization}/{stream}/_json`). Unlike OpenSearch, OpenObserve has **no upsert-by-ID concept** — every submission is appended as a new record, so re-submitting the same SHA256 produces multiple records rather than overwriting one. The SHA256 is passed as `document_id` only for interface parity with the OpenSearch backend; it is not sent to OpenObserve and is not used as a record key.

## Querying by SHA256

The `/query` endpoint calls `db.SearchBySHA256(sha256)`, which issues a SQL search against the configured stream:

```sql
SELECT sha256 FROM "<stream>" WHERE sha256='<sha256>' LIMIT 1
```

OpenObserve is time-partitioned and every search requires a time range, unlike OpenSearch's timeless term query. The search scans from a fixed floor (2020-01-01, predating OpenObserve support in Highvolt) through now. At this deployment's scale, OpenObserve's per-file min/max and bloom-filter stats let it prune partitions that can't contain a match rather than scanning every row.

A total of `0` means the file has never been analyzed. Any other value means it has.

A `400` response is treated the same as OpenSearch's 404-on-missing-index — it usually means the stream doesn't exist yet (e.g. before the first ingest) — and is treated as "not found" rather than a hard failure. Because a `400` can also indicate a genuinely malformed query, the response body is still logged at `WARN` so a real bug stays visible.

## Choosing backends

`write_backends` and `query_backend` are independent, which allows dual-writing to both OpenSearch and OpenObserve during a migration while still reading from the trusted backend, then cutting reads over once the new backend is validated:

```json
{
  "core": {
    "write_backends": ["opensearch", "openobserve"],
    "query_backend": "opensearch"
  }
}
```
