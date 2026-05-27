# Deployment Guide

## Prerequisites

| Dependency | Purpose | Notes |
|------------|---------|-------|
| OpenSearch | Result storage | Any version compatible with the Go opensearch-go v2 client |
| JSONAir | Configuration service | Provides runtime config to all components |
| LLM endpoint | PII analysis | Any OpenAI-compatible API (OpenAI, Ollama, vLLM, llama.cpp) |
| Redis | Suricata integration only | Used for Suricata pub/sub queue |
| LibreOffice | Office document conversion | Required only on the `highvolt-server` host |
| Ghostscript / pdftoppm | PDF-to-image conversion | Required only on the `highvolt-server` host |

## Environment Variables

### highvolt-server

| Variable | Required | Description |
|----------|----------|-------------|
| `JSONAIR_URL` | Yes | Base URL of the JSONAir configuration service |
| `JSONAIR_PAT` | Yes | Personal Access Token for JSONAir authentication |
| `JSONAIR_TYPE` | Yes | Config object type to fetch (e.g. `highvolt`) |
| `JSONAIR_NAME` | Yes | Config object name to fetch |
| `HIGHVOLT_PAT` | Yes | PAT clients must present to obtain a JWT |
| `JWT_SECRET` | Yes | Secret key for HMAC-SHA256 JWT signing |
| `JWT_EXPIRE` | Yes | JWT lifetime in minutes |
| `RUNAS` | No | Unix username to drop privileges to after binding |
| `CONFIG_SLEEP` | No | Interval for config polling (default 60s) |

### voltage / suricata / aws-s3

| Variable | Required | Description |
|----------|----------|-------------|
| `JSONAIR_URL` | Yes | Base URL of the JSONAir configuration service |
| `JSONAIR_PAT` | Yes | Personal Access Token for JSONAir |

## highvolt-server Configuration (via JSONAir)

```json
{
  "core": {
    "api_key": "...",
    "mime_types": {
      "image": ["image/jpeg", "image/png", "image/gif", "image/webp"],
      "pdf":   ["application/pdf"],
      "office": [
        "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        "application/vnd.ms-excel"
      ],
      "archive": ["application/zip", "application/x-tar"],
      "text": ["text/plain", "text/csv"]
    },
    "minimum_image_size": 5128,
    "max_pdf_pages": 50,
    "max_workers": 4,
    "max_queue_size": 50,
    "queue_directory": "/tmp",
    "temp_file_mode": "0600",
    "max_archive_size": 524288000,
    "archive_extract_timeout": 300,
    "max_body_size": 1073741824,
    "export_command_timeout": 120
  },
  "http": {
    "listen": ":8443",
    "tls": true,
    "cert": "/etc/highvolt/server.crt",
    "key":  "/etc/highvolt/server.key",
    "mode": "production"
  },
  "syslog": {
    "host": "localhost",
    "proto": "udp"
  },
  "opensearch": {
    "url":      "https://opensearch.internal:9200",
    "username": "highvolt",
    "password": "...",
    "index":    "highvolt",
    "tls_skip_verify": false
  },
  "llm": {
    "url":           "http://ollama.internal:11434",
    "api_key":       "...",
    "model":         "llava:latest",
    "timeout":       120,
    "system_prompt": "You are a privacy compliance assistant...",
    "user_prompt":   "Analyze this document for PII..."
  },
  "export_directories": {
    "work":    "/var/highvolt/work",
    "archive": "/var/highvolt/archive"
  },
  "export_commands": {
    "pdf":    "pdftoppm -png -r 150 %INFILE% %WORKDIR%/%OUTFILE%",
    "office": "libreoffice --headless --convert-to pdf --outdir %WORKDIR% %INFILE%"
  },
  "haproxy": {
    "enabled": false,
    "port": 9999
  }
}
```

### export_commands placeholders

| Placeholder | Replaced with |
|-------------|---------------|
| `%INFILE%` | Absolute path to the input file |
| `%WORKDIR%` | Absolute path to the temp working directory |
| `%OUTFILE%` | Output filename pattern |
| `%RANGE%` | Page range for PDF conversion (e.g. `[0-49]`) |

## Running highvolt-server

```bash
# Set environment variables
export JSONAIR_URL=https://jsonair.internal
export JSONAIR_PAT=pat_...
export JSONAIR_TYPE=highvolt
export JSONAIR_NAME=production
export HIGHVOLT_PAT=your-server-pat
export JWT_SECRET=your-jwt-secret
export JWT_EXPIRE=60
export RUNAS=highvolt

./highvolt-server
```

## Running voltage

### Environment variables

`voltage` requires `JSONAIR_URL`, `JSONAIR_PAT`, `JSONAIR_TYPE`, and `JSONAIR_NAME` at startup. When running as a service these are not inherited from a user shell, so place a `.env` file in the same directory as the `voltage` binary:

```
JSONAIR_URL=https://jsonair.internal
JSONAIR_PAT=pat_...
JSONAIR_TYPE=highvolt
JSONAIR_NAME=production
```

Protect the file since it contains credentials:

```bash
chmod 600 .env
```

### Service behavior by platform

| Platform | Mechanism | Runs as | Sleep interval |
|----------|-----------|---------|----------------|
| macOS | LaunchDaemon | root | Configurable via `core.sleep_interval` in JSONAir config (default 86400 s) |
| Linux / Unix | systemd or cron | root (or dedicated user) | Configurable via `core.sleep_interval`, or controlled by cron schedule |
| Windows | Scheduled Task (`schtasks`) | Logged-in user (`InteractiveToken`) | Fixed daily at 02:00 — runs `--once` and exits; `core.sleep_interval` has no effect |

### macOS / Linux / Unix

Install and manage as a long-running background service:

```bash
sudo ./voltage install
sudo ./voltage start
sudo ./voltage stop
sudo ./voltage restart
```

> **macOS:** `install` writes to `/Library/LaunchDaemons/` and requires `sudo`. All service commands require `sudo`.

Alternatively, on any Unix-like system you can invoke `voltage --once` from **cron** instead of running it as a persistent service. This is simpler on hosts where a full service manager is unavailable or undesirable. Example crontab entry to run a scan every day at 02:00:

```cron
0 2 * * * /opt/voltage/voltage --once
```

When using cron, the `.env` file next to the binary is still used for credentials. `core.sleep_interval` has no effect in `--once` mode — the schedule is controlled entirely by the cron expression.

### Windows

Install as a Scheduled Task (run as Administrator):

```cmd
voltage.exe install
voltage.exe start
```

The task runs `voltage.exe --once` daily at 02:00 as the logged-in user. User-level environment variables are available as an alternative to `.env`, but on unattended machines with no logged-in user the `.env` file is required. `core.sleep_interval` has no effect on Windows.

### One-shot and maintenance commands

```bash
# Run a single scan and exit (all platforms)
./voltage --once

# Remove all local state and re-scan from scratch
./voltage --nuke
```

### JSONAir configuration (voltage-specific)

```json
"core": {
  "sleep_interval": 86400
}
```

`sleep_interval` controls how many seconds `voltage` waits between scan cycles when running as a persistent service on macOS/Linux. Omit the field to use the default of 86400 (1 day). This setting has no effect on Windows or when invoked via `--once` / cron.

## Running suricata

Suricata must be configured to write file extractions to disk and publish metadata to Redis. Set `JSONAIR_URL` and `JSONAIR_PAT` in the environment, then:

```bash
./suricata
```

## Running aws-s3

AWS credentials and org-level role access must be configured. The scanner uses `OrganizationAccountAccessRole` to assume roles in member accounts.

```bash
export JSONAIR_URL=https://jsonair.internal
export JSONAIR_PAT=pat_...

./aws-s3
```

## Deploying aws-s3-lambda

The Lambda client requires Terraform and must be deployed from the AWS management account (or a delegated CloudFormation StackSets administrator). Run the interactive deployment script:

```bash
cd cmd/clients/aws-s3-lambda
./deploy.sh
```

The script auto-detects your Organization ID and root ID, prompts for all other settings, builds the Lambda binary, and runs Terraform to deploy the full infrastructure.

### Environment variables (Lambda function — set by Terraform)

| Variable | Required | Description |
|---|---|---|
| `JSONAIR_SECRET_NAME` | Yes | AWS Secrets Manager secret name holding the JSONAIR PAT |
| `JSONAIR_URL` | Yes | Base URL of the JSONAir configuration service |
| `JSONAIR_NAME` | Yes | JSONAir config profile name |
| `JSONAIR_TYPE` | Yes | JSONAir config profile type |
| `CROSS_ACCOUNT_ROLE` | No | IAM role to assume in member accounts (default: `OrganizationAccountAccessRole`) |

### Secrets Manager

Create the JSONAIR PAT secret before (or immediately after) the first deployment:

```bash
aws secretsmanager create-secret \
  --name 'highvolt/jsonair-pat' \
  --region us-east-1 \
  --secret-string '{"JSONAIR_PAT":"pat_xxxxxxxxxxxxxxxxxxxxxxxx"}'
```

### Infrastructure created

| Resource | Account | Description |
|---|---|---|
| EventBridge custom bus `highvolt-org-s3-events` | Central | Receives S3 events from all org accounts |
| Lambda `highvolt-aws-s3-lambda` | Central | Analyzes new S3 objects for PII |
| Org CloudTrail (optional) | Central | Captures S3 write events across the org |
| CloudFormation StackSet | All org accounts × all monitored regions | EventBridge forwarding rule per account/region |

### Recommended deployment strategy

Run `aws-s3` (batch) once to process all existing objects, then deploy `aws-s3-lambda` to handle all new uploads going forward. Together they provide complete org-wide coverage.

## HAProxy Health Check (optional)

When `haproxy.enabled` is `true`, `highvolt-server` binds a TCP agent on the configured port. HAProxy can use this port for health checks and dynamic weight adjustment.
