# Sourced by the other scripts: reads the environment's entry from countries.py, prints the AWS
# profile and account it will use, and stops unless the account is the entry's, so nothing is
# ever done in the wrong one. The Makefile's PROFILE=<name> sets AWS_PROFILE for it.
# Never prints a secret.
set -euo pipefail

INFRA="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO="$(cd "$INFRA/.." && pwd)"
# countries.py needs only Python's standard library, so the pipeline's steps use the image's own.
PY="$INFRA/.venv/bin/python"
[ -x "$PY" ] || PY=python3

ENV_NAME="${1:?usage: $0 <environment in infra/countries.py, e.g. uk-qa, or pipeline>}"
field() { (cd "$INFRA" && "$PY" countries.py "$ENV_NAME" "$1"); }
ACCOUNT="$(field account)"
REGION="$(field region)"
PREFIX="$([ "$ENV_NAME" = pipeline ] || field param_prefix)"
export AWS_REGION="$REGION" AWS_DEFAULT_REGION="$REGION"

profile="${AWS_PROFILE:-(none: the default credentials)}"
echo "AWS profile: $profile"
caller="$(aws sts get-caller-identity --query Account --output text)" || {
  echo "Can't use AWS profile $profile. Name the right one, e.g. make <target> PROFILE=<profile>" >&2
  echo "(and sign in first if needed: aws sso login --profile <profile>)." >&2
  exit 1
}
echo "AWS account: $caller ($ENV_NAME is in $ACCOUNT, region $REGION)"
if [ "$caller" != "$ACCOUNT" ]; then
  echo "Wrong account: stopping before doing anything. Name the right profile with PROFILE=<profile>." >&2
  exit 1
fi

# stack_output <stack> <output key>
stack_output() {
  aws cloudformation describe-stacks --stack-name "$1" \
    --query "Stacks[0].Outputs[?OutputKey=='$2'].OutputValue" --output text
}

# has_param <name>: whether the SSM parameter exists (its value isn't read).
has_param() {
  aws ssm describe-parameters --parameter-filters "Key=Name,Values=$1" --query 'length(Parameters)' --output text | grep -qx 1
}
