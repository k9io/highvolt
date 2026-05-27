output "lambda_function_arn" {
  description = "ARN of the deployed Highvolt Lambda function"
  value       = aws_lambda_function.highvolt.arn
}

output "lambda_function_name" {
  description = "Name of the deployed Lambda function"
  value       = aws_lambda_function.highvolt.function_name
}

output "lambda_role_arn" {
  description = "ARN of the Lambda execution IAM role"
  value       = aws_iam_role.lambda_role.arn
}

output "central_event_bus_arn" {
  description = "ARN of the central Highvolt EventBridge event bus (org accounts forward events here)"
  value       = aws_cloudwatch_event_bus.highvolt.arn
}

output "cloudwatch_log_group" {
  description = "CloudWatch log group for Lambda output"
  value       = aws_cloudwatch_log_group.lambda.name
}

output "cloudtrail_bucket" {
  description = "S3 bucket storing CloudTrail logs (only set when create_cloudtrail=true)"
  value       = var.create_cloudtrail ? aws_s3_bucket.cloudtrail[0].id : "n/a (existing trail)"
}

output "stackset_name" {
  description = "CloudFormation StackSet deploying EventBridge forwarding rules to all org accounts"
  value       = aws_cloudformation_stack_set.member_eventbridge_rules.name
}
