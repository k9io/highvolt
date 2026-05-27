terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  required_version = ">= 1.3"
}

provider "aws" {
  region = var.aws_region
}

data "aws_caller_identity" "current" {}

# ── Central EventBridge event bus ─────────────────────────────────────────────
# All member accounts forward their S3 CloudTrail events here.

resource "aws_cloudwatch_event_bus" "highvolt" {
  name = "highvolt-org-s3-events"
}

# Resource policy: any principal inside the org may PutEvents onto this bus.
resource "aws_cloudwatch_event_bus_policy" "highvolt_org" {
  event_bus_name = aws_cloudwatch_event_bus.highvolt.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid    = "AllowOrgPutEvents"
      Effect = "Allow"
      Principal = { AWS = "*" }
      Action   = "events:PutEvents"
      Resource = aws_cloudwatch_event_bus.highvolt.arn
      Condition = {
        StringEquals = { "aws:PrincipalOrgID" = var.org_id }
      }
    }]
  })
}

# ── EventBridge rule on the central bus → Lambda ──────────────────────────────

resource "aws_cloudwatch_event_rule" "s3_events" {
  name           = "highvolt-s3-object-created"
  description    = "Trigger Highvolt Lambda for S3 object creation events forwarded from all org accounts"
  event_bus_name = aws_cloudwatch_event_bus.highvolt.name

  event_pattern = jsonencode({
    source      = ["aws.s3"]
    "detail-type" = ["AWS API Call via CloudTrail"]
    detail = {
      eventSource = ["s3.amazonaws.com"]
      eventName   = ["PutObject", "CompleteMultipartUpload"]
    }
  })
}

resource "aws_cloudwatch_event_target" "lambda" {
  rule           = aws_cloudwatch_event_rule.s3_events.name
  event_bus_name = aws_cloudwatch_event_bus.highvolt.name
  target_id      = "HighvoltLambda"
  arn            = aws_lambda_function.highvolt.arn
}

resource "aws_lambda_permission" "allow_eventbridge" {
  statement_id  = "AllowCentralBusInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.highvolt.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.s3_events.arn
}

# ── Lambda function ───────────────────────────────────────────────────────────

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${var.lambda_function_name}"
  retention_in_days = var.log_retention_days
}

resource "aws_lambda_function" "highvolt" {
  function_name = var.lambda_function_name
  role          = aws_iam_role.lambda_role.arn
  runtime       = "provided.al2023"
  handler       = "bootstrap"
  filename      = var.lambda_zip_path
  memory_size   = var.lambda_memory_size
  timeout       = var.lambda_timeout

  source_code_hash = filebase64sha256(var.lambda_zip_path)

  environment {
    variables = {
      JSONAIR_URL         = var.jsonair_url
      JSONAIR_NAME        = var.jsonair_name
      JSONAIR_TYPE        = var.jsonair_type
      JSONAIR_SECRET_NAME = var.jsonair_secret_name
      CROSS_ACCOUNT_ROLE  = var.cross_account_role_name
    }
  }

  depends_on = [aws_cloudwatch_log_group.lambda]
}
