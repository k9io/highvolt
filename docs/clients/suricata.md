# Suricata — Network Integration

The `suricata` client integrates Highvolt with [Suricata](https://suricata.io), an open-source network intrusion detection system. Suricata can extract files from network traffic; this client picks up those extractions and submits them for PII analysis.

## Architecture

```
Network Traffic
      │
      ▼
  Suricata IDS
      │  file extracted from traffic
      ├──► Stores file at: <suricata_path>/<sha256[0:2]>/<sha256>
      └──► Publishes JSON to Redis key
                    │
                    ▼
              suricata client
                    │  polls Redis LPOP
                    │  queries highvolt-server (dedup)
                    │  submits file for analysis
                    ▼
            highvolt-server → LLM → OpenSearch
```

## How it works

1. Suricata is configured to store extracted files and publish file metadata as JSON to a Redis list (pub/sub).
2. The `suricata` client polls Redis using `LPOP` on the configured key in a tight loop (1-second sleep between empty polls).
3. For each event, it extracts `fileinfo.sha256`, `fileinfo.md5`, `fileinfo.sha1`, `fileinfo.filename`, and `fileinfo.stored` from the Suricata JSON.
4. If `fileinfo.stored` is `false`, the file was not saved to disk by Suricata — the event is skipped.
5. The client queries `highvolt-server` to check if the SHA256 has already been analyzed.
6. If not analyzed, it reads the file from disk (path: `<suricata_path>/<sha256[0:2]>/<sha256>`), validates the MIME type against the configured list, base64-encodes the file, and submits it to `highvolt-server`.
7. After successful submission, the file is **deleted from disk** to avoid unbounded growth of the Suricata file store.

## Suricata configuration requirements

In `suricata.yaml`, enable file extraction and configure the Redis output:

```yaml
file-store:
  version: 2
  enabled: yes
  dir: /var/log/suricata/filestore

outputs:
  - redis:
      enabled: yes
      server: 127.0.0.1
      port: 6379
      async: true
      mode: list
      key: suricata
      pipelining:
        enabled: yes
```

Also ensure file hashing is enabled:

```yaml
file-store:
  force-hash: [md5, sha1, sha256]
```

## Configuration (via JSONAir)

```json
{
  "core": {
    "suricata_path": "/var/log/suricata/filestore",
    "mime_types": ["application/pdf", "image/jpeg", "image/png"]
  },
  "redis": {
    "server": "127.0.0.1",
    "port":   6379,
    "key":    "suricata"
  },
  "syslog": {
    "host": "local",
    "proto": "tcp"
  },
  "highvolt": {
    "url":    "https://highvolt.internal:8443",
    "pat":    "your-highvolt-pat",
    "query":  "https://highvolt.internal:8443/api/v1/highvolt/query",
    "submit": "https://highvolt.internal:8443/api/v1/highvolt/submit"
  }
}
```

## Debug flags

The `suricata` client supports granular debug logging that can be toggled at runtime via JSONAir:

| Flag | What it logs |
|------|-------------|
| `redis` | All Redis LPOP operations |
| `submit` | Raw JSON from Suricata and submission payloads |
| `http` | All HTTP requests/responses to `highvolt-server` |
| `sleep` | Loop sleep events |

## Submitted JSON structure

The submission to `highvolt-server` includes the original Suricata `fileinfo` JSON nested under a `suricata` field, along with device information from the host:

```json
{
  "mimetype": "image/jpeg",
  "timestamp": "2026-05-19T12:00:00Z",
  "file_data": "<base64>",
  "full_path": "/var/log/suricata/filestore/ab/abcdef...",
  "filename": "abcdef...",
  "app": "suricata",
  "sha256": "abcdef...",
  "sha1": "...",
  "md5": "...",
  "suricata": { ... original Suricata fileinfo JSON ... },
  "device": { ... host information ... }
}
```
