# AWS S3 — Cloud Scanner

The `aws-s3` client scans S3 buckets across an entire AWS Organization for files matching configured MIME types and submits them to `highvolt-server` for PII analysis.

## How it works

1. Authenticates to JSONAir and `highvolt-server`.
2. Lists all accounts in the AWS Organization using the Organizations API.
3. For each account (excluding any in the `s3.exclude_users` list), assumes the `OrganizationAccountAccessRole` IAM role via STS.
4. Lists all S3 buckets in that account.
5. For each bucket, paginates through all objects.
6. For each object, calls `Analyze()`:
   - Checks the local registry (GOB file keyed by `s3://<bucket>/<key>`) — skips if already seen.
   - Downloads the first 1 MB of the object to detect the MIME type via magic bytes.
   - If the MIME type is not in the configured list, records the path in the registry and moves on.
   - If the object exceeds `core.max_file_size`, it is recorded and skipped.
   - Queries `highvolt-server` to check if the SHA256 has already been analyzed.
   - If not analyzed, streams the full object while simultaneously computing SHA256/SHA1/MD5 and building a base64 payload in a single-pass `io.MultiWriter`.
   - Submits to `highvolt-server` and records the path in the registry.

## IAM requirements

The scanner requires an IAM role in the management (root) account with:

```json
{
  "Effect": "Allow",
  "Action": [
    "organizations:ListAccounts"
  ],
  "Resource": "*"
}
```

Each member account must have an `OrganizationAccountAccessRole` that trusts the management account and has at minimum:

```json
{
  "Effect": "Allow",
  "Action": [
    "s3:ListAllMyBuckets",
    "s3:ListBucket",
    "s3:GetObject"
  ],
  "Resource": "*"
}
```

## Configuration (via JSONAir)

```json
{
  "core": {
    "mime_types": ["application/pdf", "image/jpeg", "image/png"],
    "max_file_size": 104857600
  },
  "aws": {
    "region":            "us-east-1",
    "access_key_id":     "AKIA...",
    "secret_access_key": "..."
  },
  "s3": {
    "exclude_users": ["audit-account", "log-archive"]
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

## Local registry

The `aws-s3` client maintains a local GOB registry keyed by S3 path (`s3://<bucket>/<key>`). An object is recorded in the registry when:

- Its MIME type does not match the configured list.
- It exceeds the file size limit.
- It has already been analyzed by `highvolt-server`.
- It is successfully submitted for analysis.

Objects in the registry are never re-downloaded, making subsequent runs much faster.

## Submitted JSON structure

```json
{
  "mimetype": "application/pdf",
  "timestamp": "2026-05-19T12:00:00Z",
  "file_data": "<base64>",
  "sha256": "...",
  "sha1": "...",
  "md5": "...",
  "app": "aws-s3",
  "s3_bucket": "my-bucket",
  "s3_key": "documents/report.pdf",
  "s3_account_id": "123456789012",
  "device": { ... host information ... }
}
```

## Notes

- The scanner runs once and exits. Schedule it with cron or a workflow orchestrator for periodic scanning.
- The single-pass streaming approach means each S3 object is downloaded exactly once, even though four operations are performed (MIME detection uses a buffered header read, then the remainder is streamed).
- Member accounts that are inaccessible (permission denied, suspended) log an error and are skipped — the scan continues with the next account.
