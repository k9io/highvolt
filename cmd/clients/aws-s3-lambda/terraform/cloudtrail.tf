# ── Org-level CloudTrail with S3 data events ─────────────────────────────────
#
# Only created when var.create_cloudtrail = true. If your organization already
# has an org-level trail, set create_cloudtrail = false — you can only have one
# org trail per organization.
#
# The trail enables S3 data events so CloudTrail records every PutObject and
# CompleteMultipartUpload across all member accounts. EventBridge rules in each
# member account (deployed via the StackSet in stackset.tf) then forward those
# events to the central Highvolt event bus.

locals {
  # Default log bucket name if none provided
  ct_bucket_name = var.cloudtrail_bucket_name != "" ? var.cloudtrail_bucket_name : "highvolt-cloudtrail-logs-${data.aws_caller_identity.current.account_id}"
}

resource "aws_s3_bucket" "cloudtrail" {
  count         = var.create_cloudtrail ? 1 : 0
  bucket        = local.ct_bucket_name
  force_destroy = false

  lifecycle {
    prevent_destroy = false
  }
}

resource "aws_s3_bucket_versioning" "cloudtrail" {
  count  = var.create_cloudtrail ? 1 : 0
  bucket = aws_s3_bucket.cloudtrail[0].id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "cloudtrail" {
  count  = var.create_cloudtrail ? 1 : 0
  bucket = aws_s3_bucket.cloudtrail[0].id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "cloudtrail" {
  count  = var.create_cloudtrail ? 1 : 0
  bucket = aws_s3_bucket.cloudtrail[0].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# CloudTrail requires a specific bucket policy to allow delivery.
resource "aws_s3_bucket_policy" "cloudtrail" {
  count  = var.create_cloudtrail ? 1 : 0
  bucket = aws_s3_bucket.cloudtrail[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AWSCloudTrailAclCheck"
        Effect = "Allow"
        Principal = { Service = "cloudtrail.amazonaws.com" }
        Action   = "s3:GetBucketAcl"
        Resource = aws_s3_bucket.cloudtrail[0].arn
      },
      {
        Sid    = "AWSCloudTrailWrite"
        Effect = "Allow"
        Principal = { Service = "cloudtrail.amazonaws.com" }
        Action   = "s3:PutObject"
        Resource = "${aws_s3_bucket.cloudtrail[0].arn}/AWSLogs/*"
        Condition = {
          StringEquals = { "s3:x-amz-acl" = "bucket-owner-full-control" }
        }
      },
    ]
  })

  depends_on = [aws_s3_bucket_public_access_block.cloudtrail]
}

resource "aws_cloudtrail" "org" {
  count = var.create_cloudtrail ? 1 : 0

  name                          = "highvolt-org-trail"
  s3_bucket_name                = aws_s3_bucket.cloudtrail[0].id
  include_global_service_events = true
  is_multi_region_trail         = true
  is_organization_trail         = true
  enable_log_file_validation    = true

  # S3 data events: record PutObject and CompleteMultipartUpload for all buckets.
  event_selector {
    read_write_type           = "WriteOnly"
    include_management_events = true

    data_resource {
      type   = "AWS::S3::Object"
      values = ["arn:aws:s3:::"] # all S3 objects in all accounts
    }
  }

  depends_on = [aws_s3_bucket_policy.cloudtrail]
}
