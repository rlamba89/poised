#!/usr/bin/env bash
# make aws-setup ENV=<environment>: the one-off set-up of an AWS account and region for one
# environment. Safe to run again: each step is skipped if it's already done. See infra/README.md.
#
#   NEON_DATABASE_URL  the database's direct connection string (needed the first time only)
#   ALERT_EMAIL        where the budget alarm emails go (needed the first time only)
source "$(dirname "$0")/lib.sh"
AWS_CDK_VERSION="${AWS_CDK_VERSION:?run this through make, which pins the CDK CLI version}"

# Ask for every missing value at once, before changing anything.
need=()
has_param "$PREFIX/database-url" || [ -n "${NEON_DATABASE_URL:-}" ] || need+=("NEON_DATABASE_URL='<Neon direct URL>'")
has_param /sj/alert-email || [ -n "${ALERT_EMAIL:-}" ] || need+=("ALERT_EMAIL='<your email address>'")
if [ ${#need[@]} -gt 0 ]; then
  echo "Nothing changed. The first run needs: ${need[*]}" >&2
  echo "Run: make aws-setup ENV=$ENV_NAME PROFILE=${AWS_PROFILE:-<profile>} ${need[*]}" >&2
  exit 1
fi

echo "== 1. CDK bootstrap (Poised's own: qualifier poised, stack poised-cdk-toolkit)"
(cd "$INFRA" && npx --yes "aws-cdk@$AWS_CDK_VERSION" bootstrap "aws://$ACCOUNT/$REGION" \
  --qualifier poised --toolkit-stack-name poised-cdk-toolkit \
  --cloudformation-execution-policies arn:aws:iam::aws:policy/AdministratorAccess \
  --tags project=poised)

echo "== 2. The database URL in SSM ($PREFIX/database-url, a SecureString)"
if has_param "$PREFIX/database-url"; then
  echo "   already set: skipped (to change it: aws ssm put-parameter --overwrite …, see README)"
elif [ -n "${NEON_DATABASE_URL:-}" ]; then
  aws ssm put-parameter --name "$PREFIX/database-url" --type SecureString \
    --value "$NEON_DATABASE_URL" --tags Key=project,Value=poised >/dev/null
  echo "   stored"
else
  echo "   missing: run again with NEON_DATABASE_URL='<Neon direct URL>'" >&2
  exit 1
fi

echo "== 3. The budget alarm's email address in SSM (/sj/alert-email)"
if has_param /sj/alert-email; then
  echo "   already set: skipped"
elif [ -n "${ALERT_EMAIL:-}" ]; then
  aws ssm put-parameter --name /sj/alert-email --type String --value "$ALERT_EMAIL" \
    --tags Key=project,Value=poised >/dev/null
  echo "   stored"
else
  echo "   missing: run again with ALERT_EMAIL=<your email address>" >&2
  exit 1
fi

echo "== 4. The poised-env cost tag, so the budget counts this environment's costs"
# Billing learns a tag only after a tagged resource has cost something (up to a day after the
# first deploy); until then this can't be done, and the next run does it.
if aws ce update-cost-allocation-tags-status \
  --cost-allocation-tags-status TagKey=poised-env,Status=Active >/dev/null 2>&1; then
  echo "   active"
else
  echo "   not known to Billing yet: run make aws-setup again a day after the first deploy"
fi

echo "== Done. Next: make deploy-qa ENV=$ENV_NAME"
