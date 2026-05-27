# REST API Reference

Base path: `/api/v1/highvolt`

All endpoints except `/auth/token` require a valid JWT in the `Authorization: Bearer <token>` header.

---

## POST /auth/token

Exchange a Personal Access Token for a short-lived JWT.

**Request body:**

```json
{ "token": "your-personal-access-token" }
```

**Success (200):**

```json
{
  "access_token": "eyJhbGci...",
  "expires_in": 3600
}
```

**Errors:** 400, 401, 500

---

## POST /submit

Submit a file for PII analysis. The request is queued and processed asynchronously.

**Required headers:**

```
Authorization: Bearer <JWT>
Content-Type: application/json
```

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mimetype` | string | Yes | MIME type of the file (e.g. `application/pdf`) |
| `file_data` | string | Yes | Base64-encoded file contents |
| `sha256` | string | Yes | SHA256 hex digest of the file |
| `sha1` | string | Yes | SHA1 hex digest |
| `md5` | string | Yes | MD5 hex digest |
| `filename` | string | Yes | Base filename (no path) |
| `full_path` | string | Yes | Full path or source URI of the file |
| `timestamp` | string | Yes | RFC3339 timestamp of when the file was found |
| `app` | string | Yes | Identifying name of the submitting client |

Clients may include additional fields (e.g. `suricata`, `device`) which are preserved in the stored document.

**Success (200):** Body contains a status string.

**Errors:**

| HTTP Status | Meaning |
|-------------|---------|
| 400 | Payload failed sanity check (missing required fields) |
| 401 | Invalid or expired JWT |
| 500 | Internal queue error |
| 503 | Queue is full |

---

## POST /query

Check whether a file has already been analyzed.

**Request body:**

```json
{ "sha256": "abc123..." }
```

**Responses:**

```json
{ "status": "found", "code": 200 }
```

```json
{ "status": "not found", "code": 404 }
```

**Errors:** 400 (missing sha256), 401, 500

---

## Stored document schema

After analysis, the document stored in OpenSearch contains the original submission fields (minus `file_data`) plus an `llm` field:

```json
{
  "mimetype": "application/pdf",
  "sha256": "abc123...",
  "sha1": "...",
  "md5": "...",
  "filename": "employee_records.pdf",
  "full_path": "/home/user/docs/employee_records.pdf",
  "timestamp": "2026-05-19T12:00:00Z",
  "app": "voltscan",
  "llm": {
    "has_sensitive_data": true,
    "confidence": "high",
    "reasoning": "Document contains names, SSNs, and dates of birth.",
    "description": "HR records with PII",
    "status": "success",
    "page_number": 2
  }
}
```

The `page_number` field is only present for PDFs and archives where the sensitive data was located on a specific page or within a specific nested file (`file_in_zip`).
