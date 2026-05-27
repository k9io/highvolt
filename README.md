Join the Key9 Slack channel
---------------------------

[![Slack](./images/slack.png)](https://key9identity.slack.com/)


# Highvolt

**AI-powered Personally Identifiable Information (PII) detection for documents, endpoints, and cloud storage.**

Highvolt scans documents — PDFs, images, Office files, archives, text files, and more — and uses a Large Language Model (LLM) to determine whether each file contains sensitive data. It is built to keep your private data private: by supporting any OpenAI-compatible LLM backend (including fully local, open-weight models), you control where your documents go for analysis and who sees them.

Highvolt is developed and maintained by [Key9, Inc.](https://k9.io/)

---

## Table of Contents

- [How It Works](#how-it-works)
- [Architecture](#architecture)
- [Clients](#clients)
- [Supported File Formats](#supported-file-formats)
- [Output Format](#output-format)
- [Prerequisites](#prerequisites)
- [Building](#building)
- [Running the Server](#running-the-server)
- [JSONAir](#jsonair)
- [Configuration](#configuration)
- [Scalability with HAProxy](#scalability-with-haproxy)
- [Compatible LLM Backends](#compatible-llm-backends)
- [Documentation](#documentation)
- [License](#license)

---

## How It Works

Highvolt is an AI-powered Personally Identifiable Information (PII) detection system. It scans documents — PDFs, images, Office files, archives, text files, and more — and uses a Large Language Model (LLM) to determine whether each file contains sensitive data.

The purpose of Highvolt is to allow you to search for sensitive documents while keeping them private. Using open-weight visual models, Highvolt can scan documents without sending them to a third party for analysis.

Before submitting a file to the LLM, Highvolt determines the best method of sending it. Image files can be submitted directly. PDFs and Office documents are automatically converted to an image format the LLM can ingest. Archives are unzipped and their contents analyzed recursively. Highvolt also performs SHA256-based deduplication — identical files are never analyzed twice, saving time and LLM resources.

When sensitive documents are discovered, Highvolt produces a structured JSON result containing file hashes (MD5, SHA1, SHA256), the filename, and the full LLM response. By default this data is stored in an OpenSearch index, where it can be queried by log analysis engines, access-control systems, or any other downstream tool.

---

## Architecture

Highvolt has two main components:

**`highvolt-server`** does the heavy lifting: it receives document submissions from clients, queues them for asynchronous processing, communicates with the LLM, and stores results in OpenSearch. The server exposes a REST API and handles authentication via JWT.

**Clients** run anywhere and submit documents to the server. The server does not care where the data came from — only the analysis matters.

---

## Clients

| Client | Description |
|---|---|
| **Voltage** | Scans traditional file systems on endpoints. Can run as a persistent agent (system service on Linux, macOS, and Windows) or agent-less via a scheduled task using the `--once` flag. |
| **Suricata** | Integrates with the Suricata IDS to capture and analyze files observed on the network. Uses Redis for pub/sub queuing. |
| **aws-s3** | Batch scans every S3 bucket across an entire AWS Organization for sensitive data. Runs once and exits. |
| **aws-s3-lambda** | Real-time S3 monitoring via AWS Lambda. Deployed with Terraform; fires on S3 object-created events across all member accounts. |

---

## Supported File Formats

| Type | Formats | How It's Analyzed |
|---|---|---|
| Images | JPEG, PNG, GIF, WebP | Submitted directly to the vision LLM |
| PDFs | PDF | Converted to PNG (one page at a time); stops at first PII detection |
| Office documents | DOCX, XLSX, PPTX | Converted to PDF via LibreOffice, then to images |
| Archives | ZIP, tar.gz, and more | Extracted recursively; each file analyzed by its type |
| Text | TXT, CSV | Submitted as plain text |

---

## Output Format

When a file is analyzed, Highvolt stores a record in OpenSearch with the following structure:

```json
{
  "filename": "example.pdf",
  "md5": "...",
  "sha1": "...",
  "sha256": "...",
  "llm_response": {
    "has_sensitive_data": true,
    "confidence": "high",
    "reasoning": "Document contains full name, date of birth, and SSN.",
    "description": "Patient intake form with PII.",
    "status": "success"
  }
}
```

---

## Prerequisites

The following must be available before running `highvolt-server`:

| Dependency | Purpose |
|---|---|
| [OpenSearch](https://opensearch.org/) | Stores analysis results |
| [JSONAir](https://github.com/k9io/jsonair/) | Centralized configuration service (see [JSONAir](#jsonair)) |
| OpenAI-compatible LLM endpoint | Performs PII analysis (see [Compatible LLM Backends](#compatible-llm-backends)) |
| [LibreOffice](https://www.libreoffice.org/) | Converts Office documents to PDF. Runs in a "headless" configuration |
| `pdftoppm` or Ghostscript | Converts PDFs to images |

For the **Suricata** client only: a running Redis instance is also required.

For the **aws-s3-lambda** client: Terraform and AWS credentials with sufficient permissions to deploy Lambda functions and IAM roles.

---

## Building

```bash
# Build everything
make build

# Build individual components
make highvolt-server
make voltscan          # Voltage endpoint scanner
make aws-s3            # AWS S3 batch scanner
make aws-s3-lambda     # AWS Lambda function (cross-compiled for Linux/amd64)
```

Go 1.25.2 or later is required.

---

## Running the Server

Set the required environment variables, then run the binary:

```bash
export JSONAIR_URL=https://jsonair.internal
export JSONAIR_PAT=pat_...
export JSONAIR_TYPE=highvolt
export JSONAIR_NAME=production
export HIGHVOLT_PAT=your-server-pat
export JWT_SECRET=your-jwt-secret
export JWT_EXPIRE=60        # JWT lifetime in minutes
export RUNAS=highvolt       # Optional: drop privileges to this user after startup

./highvolt-server
```

### REST API

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/highvolt/auth/token` | Obtain a JWT using the server PAT |
| `POST` | `/api/v1/highvolt/submit` | Submit a file for analysis |
| `POST` | `/api/v1/highvolt/query` | Check if a file (by SHA256) has already been analyzed |

All `/submit` and `/query` requests require a valid JWT in the `Authorization: Bearer` header.

---

## JSONAir

All Highvolt components — the server and every client — fetch their runtime configuration from **JSONAir**, a configuration service built by [Key9, Inc.](https://k9.io/) JSONAir separates configuration from deployment: you store your config in JSONAir and components pull it at startup, with automatic hot-reload so changes take effect without a restart.

You will need a running JSONAir instance and a Personal Access Token (PAT) before deploying any part of Highvolt. Visit [k9.io](https://k9.io/) to get started.

Each component identifies which config to fetch using three environment variables:

| Variable | Description |
|---|---|
| `JSONAIR_URL` | Base URL of your JSONAir instance |
| `JSONAIR_PAT` | Personal Access Token for authentication |
| `JSONAIR_TYPE` | Config object type (e.g. `highvolt`) |
| `JSONAIR_NAME` | Config object name (e.g. `production`) |

---

## Configuration

Highvolt server and client configuration is stored in JSONAir and fetched at startup. Configuration changes are applied without restarting.

Key server configuration options include:

```json
{
  "core": {
    "max_workers": 2,
    "max_queue_size": 50,
    "max_pdf_pages": 50,
    "max_archive_size": 524288000
  },
  "http": {
    "listen": ":8443",
    "tls": true,
    "cert": "/etc/highvolt/server.crt",
    "key": "/etc/highvolt/server.key"
  },
  "opensearch": {
    "url": "https://opensearch.internal:9200",
    "username": "...",
    "password": "...",
    "index": "highvolt"
  },
  "llm": {
    "url": "http://ollama.internal:11434/v1",
    "api_key": "...",
    "model": "Ministral-3-3B-Instruct-2512",
    "timeout": 120,
    "system_prompt": "...",
    "user_prompt": "..."
  }
}
```

Full configuration references for the server and each client are in the [Highvolt documentation](https://docs.k9.io/key9-identity/highvolt).

---

## Scalability with HAProxy

Because Highvolt works well with small, open-weight models in the 3–8B parameter range, you do not need cluster inference frameworks (such as [exo](https://github.com/exo-explore/exo)) that shard a single large model across multiple machines. Instead, you can run one `highvolt-server` instance per piece of hardware — each talking to its own local LLM — and use [HAProxy](https://www.haproxy.org/) to load-balance across them. This gives you both horizontal scale and a resilient cluster with no additional infrastructure.

Each `highvolt-server` instance includes a built-in HAProxy agent. When enabled, the server listens on a configurable TCP port and responds to HAProxy agent-checks with its current load as a percentage:

| Response | Meaning |
|---|---|
| `100%` | Node is idle — fully available |
| `75%` | Node is 75% available (queue 25% full) |
| `drain` | Queue is full — stop sending new work |

HAProxy uses these values for dynamic weight adjustment, automatically routing new document submissions to the least-busy node and stopping traffic to any node whose queue is saturated. If a node goes offline entirely, HAProxy routes around it.

To enable the agent, set the following in your JSONAir config for each `highvolt-server` instance:

```json
"haproxy": {
  "enabled": true,
  "port": 9999
}
```

Then configure HAProxy with an `agent-check` pointing at that port. Full details are in the [Highvolt documentation](https://docs.k9.io/key9-identity/highvolt).

---

## Compatible LLM Backends

Highvolt communicates with LLMs using the OpenAI API specification. This makes it compatible with any backend that speaks that API:

- [Ollama](https://ollama.com/)
- [llama.cpp](https://github.com/ggerganov/llama.cpp)
- [LM Studio](https://lmstudio.ai/)
- [vLLM](https://github.com/vllm-project/vllm)
- [omlx](https://github.com/jundot/omlx)
- OpenAI (cloud)
- Any other OpenAI-compatible endpoint

**Important:** Highvolt requires a **vision-capable** model. Document analysis involves submitting images to the LLM. Text-only models will not work for PDF and image scanning.

For document analysis, large frontier models are not required. Highvolt has shown strong results with models in the 3–8B parameter range. `Ministral-3-3B-Instruct-2512` has performed well on Apple hardware and is a good starting point.

---

## Documentation

Full documentation for Highvolt — including deployment guides, API references, configuration schemas, and client-specific instructions — is available at:

**[https://docs.k9.io/key9-identity/highvolt](https://docs.k9.io/key9-identity/highvolt)**

---

## License

Copyright (C) 2026 Key9, Inc. and Champ Clark III

Highvolt is licensed under the [GNU Affero General Public License v3.0](LICENSE) (AGPL-3.0).
