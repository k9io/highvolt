#!/usr/bin/env bash
# Highvolt AWS S3 Lambda — org-wide deployment script.
#
# Builds the Lambda binary, then deploys the full org-wide PII scanning
# infrastructure via Terraform:
#   - Central EventBridge event bus (this account)
#   - Lambda function + IAM role
#   - Org-level CloudTrail with S3 data events (optional)
#   - CloudFormation StackSet deploying EventBridge forwarding rules to all
#     member accounts in every monitored region
#
# Must be run from the AWS management account (or a delegated CloudFormation
# StackSets administrator account) with sufficient IAM permissions.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=========================================================="
echo "  Highvolt AWS S3 Lambda — Org-Wide Deployment"
echo "=========================================================="
echo ""

# ── Prerequisites ──────────────────────────────────────────────────────────────
missing=()
for cmd in go zip terraform aws; do
    command -v "$cmd" &>/dev/null || missing+=("$cmd")
done
if [[ ${#missing[@]} -gt 0 ]]; then
    echo "ERROR: Required tools not installed: ${missing[*]}"
    exit 1
fi

# ── Helpers ────────────────────────────────────────────────────────────────────
prompt_required() {
    local label="$1" value=""
    while [[ -z "$value" ]]; do
        read -rp "${label}: " value
        [[ -z "$value" ]] && echo "  This value is required."
    done
    printf '%s' "$value"
}

prompt_default() {
    local label="$1" default="$2" value
    read -rp "${label} [${default}]: " value
    printf '%s' "${value:-$default}"
}

prompt_yesno() {
    local label="$1" default="$2" value
    read -rp "${label} [${default}]: " value
    value="${value:-$default}"
    [[ "${value,,}" == "y" || "${value,,}" == "yes" ]] && echo "true" || echo "false"
}

# ── Step 1: Organization settings ─────────────────────────────────────────────
echo "Step 1/4: AWS Organization"
echo "──────────────────────────"
echo ""
echo "Attempting to auto-detect org values from the AWS CLI..."

ORG_ID=""
ORG_ROOT_ID=""

if ORG_ID=$(aws organizations describe-organization --query 'Organization.Id' --output text 2>/dev/null); then
    echo "  Org ID:      $ORG_ID"
else
    ORG_ID=$(prompt_required "AWS Organization ID (e.g. o-ab12cd34ef)")
fi

if ORG_ROOT_ID=$(aws organizations list-roots --query 'Roots[0].Id' --output text 2>/dev/null); then
    echo "  Org root ID: $ORG_ROOT_ID"
else
    ORG_ROOT_ID=$(prompt_required "Org root ID (e.g. r-ab12)")
fi

echo ""
AWS_REGION=$(prompt_default "AWS region for Lambda + event bus" "us-east-1")

echo ""
echo "Which AWS regions have S3 buckets you want to monitor?"
echo "The StackSet deploys EventBridge forwarding rules to each region listed."
echo "Enter a comma-separated list (e.g. us-east-1,us-west-2,eu-west-1)."
REGIONS_INPUT=$(prompt_default "Monitored regions" "us-east-1")
# Convert comma-separated to Terraform list syntax: ["us-east-1","us-west-2"]
MONITORED_REGIONS=$(echo "$REGIONS_INPUT" | tr ',' '\n' | sed 's/^ *//;s/ *$//' | \
    awk 'BEGIN{printf "["} NR>1{printf ","} {printf "\"%s\"", $0} END{print "]"}')

CROSS_ACCOUNT_ROLE=$(prompt_default "Cross-account IAM role name in member accounts" "OrganizationAccountAccessRole")

echo ""

# ── Step 2: JSONAIR / Highvolt settings ───────────────────────────────────────
echo "Step 2/4: JSONAIR & Secrets Manager"
echo "────────────────────────────────────"
echo ""
JSONAIR_URL=$(prompt_required "JSONAIR service URL (e.g. https://jsonair.example.com)")
JSONAIR_NAME=$(prompt_required "JSONAIR config profile name")
JSONAIR_TYPE=$(prompt_required "JSONAIR config profile type")
JSONAIR_SECRET_NAME=$(prompt_default "Secrets Manager secret name for JSONAIR PAT" "highvolt/jsonair-pat")
echo ""

# ── Step 3: Lambda & CloudTrail settings ──────────────────────────────────────
echo "Step 3/4: Lambda & CloudTrail"
echo "──────────────────────────────"
echo ""
LAMBDA_NAME=$(prompt_default "Lambda function name" "highvolt-aws-s3-lambda")
LAMBDA_MEMORY=$(prompt_default "Lambda memory (MB)" "256")
LAMBDA_TIMEOUT=$(prompt_default "Lambda timeout (seconds, max 900)" "300")
LOG_RETENTION=$(prompt_default "CloudWatch log retention (days)" "30")

echo ""
echo "An org-level CloudTrail trail with S3 data events is required."
echo "WARNING: You can only have ONE org trail per organization."
CREATE_CT=$(prompt_yesno "Create a new org CloudTrail trail? (choose 'n' if one already exists)" "y")

CT_BUCKET_NAME=""
if [[ "$CREATE_CT" == "true" ]]; then
    ACCT_ID=$(aws sts get-caller-identity --query Account --output text 2>/dev/null || echo "unknown")
    CT_BUCKET_NAME=$(prompt_default "S3 bucket name for CloudTrail logs" "highvolt-cloudtrail-logs-${ACCT_ID}")
fi
echo ""

# ── Step 4: Enable CloudFormation trusted access ───────────────────────────────
echo "Step 4/4: CloudFormation StackSets pre-flight"
echo "──────────────────────────────────────────────"
echo ""
echo "Enabling CloudFormation trusted access with AWS Organizations"
echo "(required for SERVICE_MANAGED StackSets — safe to run multiple times)..."
aws organizations enable-aws-service-access \
    --service-principal stacksets.cloudformation.amazonaws.com 2>/dev/null || true
echo "Done."
echo ""

# ── Build ──────────────────────────────────────────────────────────────────────
echo "Building Lambda binary (linux/amd64, CGO disabled)..."
cd "$SCRIPT_DIR"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o bootstrap .
zip -j lambda.zip bootstrap
rm bootstrap
echo "Created: ${SCRIPT_DIR}/lambda.zip"
echo ""

# ── Write terraform.tfvars ─────────────────────────────────────────────────────
echo "Writing terraform/terraform.tfvars..."
cd "${SCRIPT_DIR}/terraform"

cat > terraform.tfvars <<EOF
# Auto-generated by deploy.sh — re-run deploy.sh to regenerate.

aws_region           = "${AWS_REGION}"
org_id               = "${ORG_ID}"
org_root_id          = "${ORG_ROOT_ID}"
monitored_regions    = ${MONITORED_REGIONS}
cross_account_role_name = "${CROSS_ACCOUNT_ROLE}"

jsonair_url          = "${JSONAIR_URL}"
jsonair_name         = "${JSONAIR_NAME}"
jsonair_type         = "${JSONAIR_TYPE}"
jsonair_secret_name  = "${JSONAIR_SECRET_NAME}"

lambda_function_name = "${LAMBDA_NAME}"
lambda_memory_size   = ${LAMBDA_MEMORY}
lambda_timeout       = ${LAMBDA_TIMEOUT}
log_retention_days   = ${LOG_RETENTION}
lambda_zip_path      = "../lambda.zip"

create_cloudtrail    = ${CREATE_CT}
cloudtrail_bucket_name = "${CT_BUCKET_NAME}"
EOF

echo ""
echo "Initializing Terraform..."
terraform init -upgrade

echo ""
terraform plan

echo ""
read -rp "Apply this plan? [y/N]: " CONFIRM
if [[ "${CONFIRM,,}" != "y" ]]; then
    echo "Deployment cancelled. The plan was not applied."
    exit 0
fi

terraform apply -auto-approve

echo ""
echo "=========================================================="
echo "  Deployment complete!"
echo "=========================================================="
echo ""
echo "POST-DEPLOYMENT CHECKLIST"
echo ""
echo "1. Create (or verify) the Secrets Manager secret:"
echo ""
echo "   aws secretsmanager create-secret \\"
echo "     --name '${JSONAIR_SECRET_NAME}' \\"
echo "     --region '${AWS_REGION}' \\"
echo "     --secret-string '{\"JSONAIR_PAT\":\"<your-pat-here>\"}'"
echo ""
echo "   To update an existing secret:"
echo "   aws secretsmanager put-secret-value \\"
echo "     --secret-id '${JSONAIR_SECRET_NAME}' \\"
echo "     --region '${AWS_REGION}' \\"
echo "     --secret-string '{\"JSONAIR_PAT\":\"<your-pat-here>\"}'"
echo ""
echo "2. Verify the StackSet deployed successfully to all accounts:"
echo ""
echo "   aws cloudformation list-stack-set-operations \\"
echo "     --stack-set-name highvolt-s3-event-forwarding \\"
echo "     --region '${AWS_REGION}'"
echo ""
echo "3. Confirm the '${CROSS_ACCOUNT_ROLE}' role exists in each"
echo "   member account (AWS Organizations creates it automatically"
echo "   for accounts created or invited via the console/API)."
echo ""
echo "4. Monitor Lambda logs:"
echo "   aws logs tail /aws/lambda/${LAMBDA_NAME} --follow --region ${AWS_REGION}"
