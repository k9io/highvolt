# AWS S3 Lambda — Real-Time Cloud Scanner

The `aws-s3-lambda` client is an AWS Lambda function that detects new S3 objects across an entire AWS Organization in real time and submits them to `highvolt-server` for PII analysis. Unlike the batch `aws-s3` scanner, this client is event-driven: it is invoked within seconds of a file being uploaded to any bucket in any member account.

## How it works

```
New S3 object in any org account / any region
         │
         │  CloudTrail data event (PutObject / CompleteMultipartUpload)
         ▼
EventBridge rule in source account
  (deployed via CloudFormation StackSet)
         │
         │  events:PutEvents (cross-account, org-scoped)
         ▼
Central EventBridge event bus
  (management / security account)
         │
         │  Lambda invocation
         ▼
highvolt-aws-s3-lambda
  1. Parse CloudTrail event → bucket, key, region, account ID
  2. STS AssumeRole → OrganizationAccountAccessRole in source account
  3. S3 GetObject (cross-account, cross-region)
  4. MIME detection (magic bytes, first 1 MB)
  5. SHA256 / SHA1 / MD5 hashing + base64 encode (single-pass stream)
  6. Query highvolt-server by SHA256 — skip if already analyzed
  7. Submit to highvolt-server
```

### Startup (cold start)

On each Lambda cold start, `init()`:

1. Loads required environment variables.
2. Fetches the JSONAIR PAT from **AWS Secrets Manager** (secret name from `JSONAIR_SECRET_NAME`).
3. Authenticates to JSONAir and downloads the runtime configuration (MIME type list, max file size, Highvolt URL and PAT).
4. Authenticates to `highvolt-server` and obtains a JWT bearer token.
5. Determines the Lambda's own AWS account ID via `STS GetCallerIdentity`.
6. Initializes a base AWS SDK config and STS client for subsequent role assumptions.

The JWT bearer token is cached in the Lambda container across warm invocations. Expired tokens (HTTP 401 from `highvolt-server`) are transparently refreshed during the handler.

### Per-invocation handler

Each invocation receives one CloudTrail S3 event from EventBridge. The handler:

1. Parses the event to extract bucket name, object key, source region, and source account ID. The object key is URL-decoded (CloudTrail encodes special characters).
2. If the source account matches the Lambda's own account, uses the Lambda execution role's credentials directly. Otherwise assumes `OrganizationAccountAccessRole` (or the role configured in `CROSS_ACCOUNT_ROLE`) in the source account via STS.
3. Creates a region-scoped S3 client for the source bucket's region.
4. Calls `Analyze()`.

### Deduplication

There is no local registry file (the Lambda is stateless). Deduplication is handled entirely by querying `highvolt-server` with the object's SHA256 before submitting. Objects already analyzed by the server are silently skipped.

## AWS Infrastructure

The full infrastructure is managed by Terraform and deployed with `deploy.sh`.

### Central account resources

| Resource | Description |
|---|---|
| EventBridge custom event bus `highvolt-org-s3-events` | Receives forwarded S3 events from all org accounts. Resource policy restricts puts to principals inside the org (`aws:PrincipalOrgID`). |
| EventBridge rule `highvolt-s3-object-created` | Matches `PutObject` and `CompleteMultipartUpload` events on the central bus and invokes the Lambda. |
| Lambda function `highvolt-aws-s3-lambda` | The scanner. Runtime `provided.al2023` (Go custom runtime). |
| CloudWatch log group `/aws/lambda/highvolt-aws-s3-lambda` | Lambda output. Retention configurable (default 30 days). |
| IAM role `highvolt-aws-s3-lambda-role` | Lambda execution role. See IAM requirements below. |

### Member account resources (deployed via CloudFormation StackSet)

A CloudFormation StackSet (`highvolt-s3-event-forwarding`) deploys the following into **every account × region** combination in the organization:

| Resource | Description |
|---|---|
| IAM role `highvolt-eventbridge-forwarder` | Allows EventBridge to call `events:PutEvents` on the central bus. |
| EventBridge rule `highvolt-s3-object-created-forward` | Matches `PutObject` / `CompleteMultipartUpload` CloudTrail events and forwards them to the central bus. |

The StackSet uses `auto_deployment`, so accounts that join the organization in the future receive the rules automatically.

### Org-level CloudTrail (optional)

When `create_cloudtrail = true`, Terraform creates an org-level multi-region CloudTrail trail with **S3 write-only data events** enabled for all buckets (`arn:aws:s3:::`). This generates the CloudTrail events that the member-account EventBridge rules forward.

> You can only have one org-level CloudTrail trail per organization. If your org already has one, set `create_cloudtrail = false` and ensure S3 data events (write-only or all) are enabled on the existing trail.

## IAM requirements

### Lambda execution role

```json
{
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"],
      "Resource": "arn:aws:logs:<region>:<account>:log-group:/aws/lambda/highvolt-aws-s3-lambda:*"
    },
    {
      "Effect": "Allow",
      "Action": ["s3:GetObject"],
      "Resource": "arn:aws:s3:::*"
    },
    {
      "Effect": "Allow",
      "Action": ["sts:AssumeRole"],
      "Resource": "arn:aws:iam::*:role/OrganizationAccountAccessRole"
    },
    {
      "Effect": "Allow",
      "Action": ["sts:GetCallerIdentity"],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": ["secretsmanager:GetSecretValue"],
      "Resource": "arn:aws:secretsmanager:<region>:<account>:secret:highvolt/jsonair-pat*"
    }
  ]
}
```

### Member accounts

Each member account must have an `OrganizationAccountAccessRole` that trusts the management (central) account. AWS Organizations creates this role automatically for accounts created or invited via the console or API. The role needs at minimum:

```json
{
  "Effect": "Allow",
  "Action": ["s3:GetObject"],
  "Resource": "arn:aws:s3:::*"
}
```

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `JSONAIR_SECRET_NAME` | Yes | AWS Secrets Manager secret name containing the JSONAIR PAT (e.g. `highvolt/jsonair-pat`) |
| `JSONAIR_URL` | Yes | Base URL of the JSONAir configuration service |
| `JSONAIR_NAME` | Yes | JSONAir config profile name |
| `JSONAIR_TYPE` | Yes | JSONAir config profile type |
| `CROSS_ACCOUNT_ROLE` | No | IAM role name to assume in member accounts. Defaults to `OrganizationAccountAccessRole`. |
| `TLS_SKIP_VERIFY` | No | Set `true` to disable TLS verification. Do not use in production. |

## Secrets Manager secret format

The JSONAIR PAT can be stored in Secrets Manager as either a JSON object or a plain string:

```json
{ "JSONAIR_PAT": "pat_xxxxxxxxxxxxxxxxxxxxxxxx" }
```

```bash
aws secretsmanager create-secret \
  --name 'highvolt/jsonair-pat' \
  --region us-east-1 \
  --secret-string '{"JSONAIR_PAT":"pat_xxxxxxxxxxxxxxxxxxxxxxxx"}'
```

## Configuration (via JSONAir)

The Lambda fetches the same JSONAir config structure as the batch `aws-s3` client, minus the `aws` and `s3` blocks (credentials and account exclusions are not needed — IAM roles and the event trigger handle those concerns):

```json
{
  "core": {
    "mime_types": ["application/pdf", "image/jpeg", "image/png"],
    "max_file_size": 104857600
  },
  "highvolt": {
    "url": "https://highvolt.internal:8443",
    "pat": "your-highvolt-pat"
  }
}
```

## Deployment

```bash
cd cmd/clients/aws-s3-lambda
./deploy.sh
```

`deploy.sh` performs the following steps:

1. Auto-detects `org_id` and `org_root_id` from the AWS CLI (prompts if unavailable).
2. Prompts for monitored regions, JSONAIR settings, Lambda sizing, and CloudTrail preference.
3. Enables CloudFormation trusted access with AWS Organizations (`stacksets.cloudformation.amazonaws.com`).
4. Cross-compiles the Lambda binary (`GOOS=linux GOARCH=amd64 CGO_ENABLED=0`) and packages it as `lambda.zip`.
5. Writes `terraform/terraform.tfvars`.
6. Runs `terraform init`, `terraform plan`, and (on confirmation) `terraform apply`.

### Prerequisites

| Requirement | Notes |
|---|---|
| Run from the management account (or delegated admin) | Required for SERVICE_MANAGED CloudFormation StackSets |
| `go`, `zip`, `terraform`, `aws` CLI installed | Checked by `deploy.sh` at startup |
| CloudFormation trusted access enabled | `deploy.sh` enables this automatically |
| `OrganizationAccountAccessRole` in member accounts | Created automatically by AWS Organizations for most accounts |

### Post-deployment verification

```bash
# Confirm StackSet deployed successfully to all accounts
aws cloudformation list-stack-set-operations \
  --stack-set-name highvolt-s3-event-forwarding \
  --region us-east-1

# Stream Lambda logs
aws logs tail /aws/lambda/highvolt-aws-s3-lambda --follow --region us-east-1

# Test by uploading a file to any org bucket
aws s3 cp test.pdf s3://your-bucket/test.pdf --profile member-account-profile
```

## Submitted JSON structure

```json
{
  "mimetype":   "application/pdf",
  "timestamp":  "2026-05-23T14:00:00Z",
  "file_data":  "<base64>",
  "sha256":     "...",
  "sha1":       "...",
  "md5":        "...",
  "app":        "aws-s3-lambda",
  "full_path":  "s3://my-bucket/documents/report.pdf",
  "filename":   "report.pdf",
  "data_origin": {
    "os":      { "hostname": "...", "os": "linux", ... },
    "cpu":     { "model": "...", "cores": 2, ... },
    "memory":  { "total_mem": 0 },
    "network": { "interface": "eth0", "address": "..." }
  }
}
```

The `data_origin` block reflects the Lambda execution environment rather than an end-user machine.

## Comparison with aws-s3 (batch scanner)

| | `aws-s3` | `aws-s3-lambda` |
|---|---|---|
| **Trigger** | Manual / scheduled (cron) | Real-time (S3 event → EventBridge → Lambda) |
| **Scope** | All existing objects in all org buckets | New objects only, from the moment of deployment |
| **Deduplication** | Local GOB registry + server SHA256 query | Server SHA256 query only (stateless) |
| **Cross-account access** | STS AssumeRole per account | STS AssumeRole per invocation |
| **Configuration** | JSONAir + `.env` file | JSONAir + Lambda env vars + Secrets Manager |
| **Deployment** | Single binary | Terraform (Lambda + CloudTrail + StackSet) |

For complete org coverage, run the batch `aws-s3` scanner once to process existing objects, then deploy `aws-s3-lambda` to catch all new uploads going forward.

## Notes

- The StackSet deploys one EventBridge rule per account per monitored region. Add regions to `monitored_regions` in `terraform.tfvars` and re-run `terraform apply` to extend coverage.
- CloudTrail S3 data events are priced per-event (~$0.10 per 100,000 events). For high-volume orgs, consider enabling `WriteOnly` (already the default in the Terraform) to avoid charging for read operations.
- Lambda invocations are priced per-request and per GB-second. At 256 MB and a 5-second average runtime, analyzing 1 million files costs approximately $2.50 in compute plus data transfer.
- If a member account is inaccessible (suspended, role missing), the Lambda logs the error and returns without retrying — EventBridge does not re-deliver the event by default.
