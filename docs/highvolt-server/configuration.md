# Configuration Reference

`highvolt-server` configuration is fetched from JSONAir at startup and refreshed periodically. The configuration is a single JSON document.

## core

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `api_key` | string | — | Internal API key (reserved) |
| `mime_types.image` | []string | — | MIME types routed to the image LLM path |
| `mime_types.pdf` | []string | — | MIME types routed to the PDF processor |
| `mime_types.office` | []string | — | MIME types routed to the Office processor |
| `mime_types.archive` | []string | — | MIME types routed to the archive processor |
| `mime_types.text` | []string | — | MIME types routed to the text LLM path |
| `minimum_image_size` | int | `5128` | Minimum decoded image size (bytes); smaller images are skipped |
| `max_pdf_pages` | int | `50` | Maximum PDF pages to analyze per document |
| `max_workers` | int | `2` | Number of parallel worker goroutines |
| `max_queue_size` | int | `50` | Submission queue depth; excess submissions receive 503 |
| `queue_directory` | string | — | Base directory for temp work files |
| `temp_file_mode` | string | `"0600"` | Octal file permissions for temp files |
| `max_archive_size` | int64 | `524288000` | Max total extracted bytes from an archive (500 MB) |
| `archive_extract_timeout` | int | `300` | Max seconds for archive extraction (5 minutes) |
| `max_body_size` | int64 | `1073741824` | Max HTTP request body size (1 GB) |
| `export_command_timeout` | int | `120` | Max seconds for PDF/Office conversion commands |

## http

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `listen` | string | — | TCP listen address (e.g. `:8443`) |
| `tls` | bool | `false` | Enable TLS |
| `cert` | string | — | Path to TLS certificate (required if `tls: true`) |
| `key` | string | — | Path to TLS private key (required if `tls: true`) |
| `mode` | string | `"production"` | Gin mode: `production`, `debug`, or `test`. In production, HTTP request logging is disabled. |

## syslog

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `host` | string | `"local"` | Syslog host, or `"local"` for stderr/stdout |
| `proto` | string | `"tcp"` | Transport protocol: `tcp` or `udp` |

## opensearch

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `url` | string | — | OpenSearch base URL |
| `username` | string | — | Basic auth username |
| `password` | string | — | Basic auth password |
| `index` | string | — | Index name for documents |
| `tls_skip_verify` | bool | `false` | Disable TLS cert verification (not for production) |

## llm

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `url` | string | — | LLM base URL (OpenAI-compatible). The path `/chat/completions` is appended automatically. |
| `api_key` | string | — | Bearer token for LLM API authentication |
| `model` | string | — | Model name (e.g. `gpt-4o`, `llava:latest`) |
| `timeout` | int | `120` | Request timeout in seconds |
| `system_prompt` | string | — | System prompt sent with every LLM request |
| `user_prompt` | string | — | User prompt sent with every LLM request |

### LLM prompt design

The LLM is expected to return a JSON object with this schema:

```json
{
  "has_sensitive_data": true,
  "confidence": "high",
  "reasoning": "...",
  "description": "...",
  "status": "success"
}
```

Your `system_prompt` should instruct the model to always respond with valid JSON in this format. Example:

```
You are a privacy compliance assistant. You analyze documents for Personally Identifiable Information (PII). Always respond with a single JSON object containing: has_sensitive_data (boolean), confidence (low/medium/high), reasoning (string), description (string).
```

## export_directories

| Key | Type | Description |
|-----|------|-------------|
| `work` | string | Base directory for temporary working files |
| `archive` | string | Directory for extracted archive content |

Both directories must be writable by the process user.

## export_commands

| Key | Type | Description |
|-----|------|-------------|
| `pdf` | string | Command to convert PDF to PNG images. Supports `%INFILE%`, `%WORKDIR%`, `%OUTFILE%`, `%RANGE%` placeholders. |
| `office` | string | Command to convert Office documents to PDF. Supports `%INFILE%`, `%WORKDIR%`, `%OUTFILE%` placeholders. |

## haproxy

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Enable HAProxy TCP health agent |
| `port` | int | — | TCP port for the HAProxy agent (required if `enabled: true`) |
