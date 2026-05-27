# Highvolt

Highvolt is an AI-powered Personally Identifiable Information (PII) detection system built by [Key9, Inc](https://k9.io). It scans documents — PDFs, images, Office files, archives, text files, and more — and uses a Large Language Model (LLM) to determine whether each file contains sensitive data.

## What it does

Highvolt accepts files from multiple sources, routes them through an analysis pipeline, and records the results in OpenSearch. Each file is evaluated by an LLM that returns a structured JSON verdict:

```json
{
  "has_sensitive_data": true,
  "confidence": "high",
  "reasoning": "Document contains social security numbers and full names.",
  "description": "HR record with PII",
  "status": "success"
}
```

Results are deduplicated by SHA256 hash — a file already analyzed is never re-submitted.

## Components

| Component | Description |
|-----------|-------------|
| `highvolt-server` | Central REST API server. Receives file submissions, queues work, invokes the LLM, and stores results in OpenSearch. |
| `voltage` | Endpoint agent. Scans local filesystems (Linux, macOS, Windows) for files matching configured MIME types and submits them to `highvolt-server`. |
| `suricata` | Network integration. Listens on a Redis pub/sub queue populated by Suricata IDS and submits network-captured files for analysis. |
| `aws-s3` | Batch cloud scanner. Traverses all existing buckets across an AWS Organization and submits matching objects to `highvolt-server`. Run once to establish a baseline. |
| `aws-s3-lambda` | Real-time cloud scanner. An AWS Lambda triggered by S3 object-creation events across the entire organization. Analyzes new uploads within seconds of arrival. |

## License

Highvolt is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
