# ── AWS / Organization ────────────────────────────────────────────────────────

variable "aws_region" {
  description = "AWS region where the Lambda and central event bus are deployed (typically your management or security account's primary region)"
  type        = string
}

variable "org_id" {
  description = "AWS Organization ID (e.g. o-ab12cd34ef). Used to scope the central event bus resource policy so only org accounts can put events."
  type        = string
}

variable "org_root_id" {
  description = "AWS Organization root ID (e.g. r-ab12). The CloudFormation StackSet is deployed to every account under this root."
  type        = string
}

variable "monitored_regions" {
  description = "List of AWS regions to deploy the member-account EventBridge forwarding rules into. Include every region where org S3 buckets exist."
  type        = list(string)
  default     = ["us-east-1"]
}

variable "cross_account_role_name" {
  description = "IAM role name in each member account that the Lambda assumes to read S3 objects. AWS Organizations creates this automatically as 'OrganizationAccountAccessRole'."
  type        = string
  default     = "OrganizationAccountAccessRole"
}

# ── JSONAIR / Highvolt ────────────────────────────────────────────────────────

variable "jsonair_url" {
  description = "JSONAIR service base URL (e.g. https://jsonair.example.com)"
  type        = string
}

variable "jsonair_name" {
  description = "JSONAIR config profile name"
  type        = string
}

variable "jsonair_type" {
  description = "JSONAIR config profile type"
  type        = string
}

variable "jsonair_secret_name" {
  description = "AWS Secrets Manager secret name that holds the JSONAIR PAT (e.g. highvolt/jsonair-pat)"
  type        = string
}

# ── Lambda ────────────────────────────────────────────────────────────────────

variable "lambda_function_name" {
  description = "Name for the Lambda function"
  type        = string
  default     = "highvolt-aws-s3-lambda"
}

variable "lambda_memory_size" {
  description = "Lambda memory allocation in MB. Increase for very large files."
  type        = number
  default     = 256
}

variable "lambda_timeout" {
  description = "Lambda timeout in seconds (max 900). Allow enough time for large file downloads."
  type        = number
  default     = 300
}

variable "log_retention_days" {
  description = "CloudWatch Logs retention period in days"
  type        = number
  default     = 30
}

variable "lambda_zip_path" {
  description = "Path to the Lambda deployment zip built by deploy.sh"
  type        = string
  default     = "../lambda.zip"
}

# ── CloudTrail ────────────────────────────────────────────────────────────────

variable "create_cloudtrail" {
  description = "Set to true to create an org-level CloudTrail trail with S3 data events. Set to false if your org already has one — you only need one org trail."
  type        = bool
  default     = true
}

variable "cloudtrail_bucket_name" {
  description = "Name of the S3 bucket to store CloudTrail logs. Created automatically when create_cloudtrail=true."
  type        = string
  default     = ""
}
