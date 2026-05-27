# ── CloudFormation StackSet: EventBridge forwarding rules ─────────────────────
#
# Deploys an EventBridge rule and a cross-account forwarder IAM role into every
# AWS account in the organization (under org_root_id) for every region listed in
# monitored_regions. Each rule matches S3 CloudTrail events and forwards them to
# the central Highvolt event bus in this account.
#
# Prerequisites:
#   1. This Terraform must run from the management account (or a delegated
#      CloudFormation StackSets administrator account).
#   2. CloudFormation trusted access with AWS Organizations must be enabled:
#        aws organizations enable-aws-service-access \
#          --service-principal stacksets.cloudformation.amazonaws.com
#   3. The org-level CloudTrail (cloudtrail.tf or an existing trail) must have
#      S3 data events enabled so CloudTrail generates the events that these
#      rules will match.

resource "aws_cloudformation_stack_set" "member_eventbridge_rules" {
  name             = "highvolt-s3-event-forwarding"
  description      = "Highvolt: forward S3 object-creation CloudTrail events to the central event bus"
  permission_model = "SERVICE_MANAGED"
  template_body    = file("${path.module}/templates/member-eventbridge-rule.yaml")

  capabilities = ["CAPABILITY_NAMED_IAM"]

  parameters = {
    CentralEventBusArn = aws_cloudwatch_event_bus.highvolt.arn
  }

  # Automatically apply the StackSet to accounts that join the org in future,
  # and remove stacks if accounts leave.
  auto_deployment {
    enabled                          = true
    retain_stacks_on_account_removal = false
  }

  operation_preferences {
    failure_tolerance_percentage = 20
    max_concurrent_percentage    = 50
    region_concurrency_type      = "PARALLEL"
  }

  depends_on = [aws_cloudwatch_event_bus_policy.highvolt_org]
}

# Deploy one StackSet instance per monitored region, targeting the entire org root.
# This creates a stack in every account × region combination.
resource "aws_cloudformation_stack_set_instance" "org_regions" {
  for_each = toset(var.monitored_regions)

  stack_set_name = aws_cloudformation_stack_set.member_eventbridge_rules.name
  region         = each.key

  deployment_targets {
    # Targeting the root deploys to ALL accounts in the organization.
    organizational_unit_ids = [var.org_root_id]
  }

  operation_preferences {
    failure_tolerance_percentage = 20
    max_concurrent_percentage    = 50
    region_concurrency_type      = "PARALLEL"
  }
}
