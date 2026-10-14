#!/usr/bin/env bash
# The QA test users (S12; apps/api/cmd/qa-users): the seed's staff get sign-ins in the staff user
# pool with known passwords and authenticator secrets, kept in the SSM SecureString
# <prefix>/test-users. QA only: it refuses any other stage.
#   qa-users.sh <env> setup             make qa-users (the pipeline runs it after the seed)
#   qa-users.sh <env> code <who>        make qa-code WHO='Cara Clinician': the current 6-digit code
#   qa-users.sh <env> password <who>    make qa-password WHO=…: copies the password to the clipboard
# The database URL is read from SSM into this process only; no password or secret is printed.
source "$(dirname "$0")/lib.sh"

stage="$(field stage)"
[ "$stage" = qa ] || { echo "Test users are for QA only; $ENV_NAME is $stage." >&2; exit 1; }
cmd="${2:?usage: $0 <env> setup | code <who> | password <who>}"

param() { aws ssm get-parameter --name "$PREFIX/$1" "${@:2}" --query Parameter.Value --output text; }
export STAGE="$stage" QA_USERS_PARAM="$PREFIX/test-users"
if [ "$cmd" = setup ]; then
  STAFF_POOL_ID="$(param cognito-user-pool-id)"
  STAFF_POOL_CLIENT_ID="$(param cognito-client-id)"
  DATABASE_URL="$(param database-url --with-decryption)"
  export STAFF_POOL_ID STAFF_POOL_CLIENT_ID DATABASE_URL
fi
# The Go AWS SDK can't always refresh an SSO sign-in that the CLI can, so a named profile's
# credentials are handed to it as the CLI has them (into this process only, never printed).
if [ -n "${AWS_PROFILE:-}" ]; then eval "$(aws configure export-credentials --format env)"; fi
cd "$REPO/apps/api"
if [ "$cmd" = password ]; then
  command -v pbcopy >/dev/null || { echo "make qa-password copies to the clipboard with pbcopy (macOS)." >&2; exit 1; }
  go run ./cmd/qa-users password "${3:?who: a name or an email}" | pbcopy
  echo "Copied ${3}'s password to the clipboard."
else
  go run ./cmd/qa-users "$cmd" "${@:3}"
fi
