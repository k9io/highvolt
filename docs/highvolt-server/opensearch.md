# OpenSearch Integration

Highvolt uses OpenSearch for two operations: **indexing** analysis results and **querying** by SHA256 to detect duplicates.

## Connection

The OpenSearch client is initialized once at startup with credentials from the server configuration:

```json
{
  "opensearch": {
    "url":      "https://opensearch.internal:9200",
    "username": "highvolt",
    "password": "...",
    "index":    "highvolt",
    "tls_skip_verify": false
  }
}
```

`tls_skip_verify: true` disables TLS certificate validation. A warning is logged at startup if this is set. Do not use in production.

## Indexing a document

After LLM analysis, the worker calls `db.Index_Document(sha256, jsondata)`. Before indexing:

- The `file_data` field is **deleted** from the JSON — raw file contents are never stored in OpenSearch.
- The SHA256 hash is used as the **document ID**, making every submission naturally idempotent.

## Querying by SHA256

The `/query` endpoint calls `db.SearchBySHA256(sha256)`, which issues a `term` query against the configured index:

```json
{
  "query": {
    "term": {
      "sha256": { "value": "<sha256>" }
    }
  }
}
```

A `hits.total.value` of `0` means the file has never been analyzed. Any other value means it has.

## Index mapping recommendation

No explicit mapping is required, but setting `sha256` as a `keyword` field prevents tokenization and ensures exact-match term queries work correctly:

```json
{
  "mappings": {
    "properties": {
      "sha256": { "type": "keyword" },
      "md5":    { "type": "keyword" },
      "sha1":   { "type": "keyword" },
      "mimetype": { "type": "keyword" },
      "app":    { "type": "keyword" },
      "timestamp": { "type": "date" },
      "llm": {
        "properties": {
          "has_sensitive_data": { "type": "boolean" },
          "confidence": { "type": "keyword" }
        }
      }
    }
  }
}
```
