# Authentication

`highvolt-server` uses a two-stage authentication scheme:

1. A long-lived **Personal Access Token (PAT)** is exchanged for a short-lived **JWT**.
2. All protected endpoints require the JWT as a `Bearer` token in the `Authorization` header.

## Obtaining a token

```http
POST /api/v1/highvolt/auth/token
Content-Type: application/json

{ "token": "<your-PAT>" }
```

**Success response:**

```json
{
  "access_token": "<JWT>",
  "expires_in": 3600
}
```

`expires_in` is in seconds (the configured expiry in minutes × 60).

**Error responses:**

| HTTP Status | Meaning |
|-------------|---------|
| 400 | Missing `token` field in request body |
| 401 | PAT is invalid or expired |
| 500 | Internal error generating JWT |

## Using the token

Include the JWT in the `Authorization` header for all `/api/v1/highvolt/submit` and `/api/v1/highvolt/query` requests:

```http
Authorization: Bearer <JWT>
```

When the JWT expires the server returns `401`. Clients should re-authenticate with the PAT and retry.

## Rate limiting

The `/auth/token` endpoint is rate-limited per client IP to prevent brute-force attacks. Requests exceeding the limit receive HTTP `429 Too Many Requests`.

## Security headers

Every response includes:

```
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains  (TLS only)
```

## PAT validation

The server compares the submitted PAT against the value in `HIGHVOLT_PAT` using a **constant-time comparison** (`crypto/subtle.ConstantTimeCompare`) to prevent timing-based enumeration attacks.
