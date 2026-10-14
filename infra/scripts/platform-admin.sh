#!/usr/bin/env bash
# make aws-platform-admin ENV=<env> EMAIL=… [NAME=…]: adds a superadmin (A-21) in an AWS
# environment: the account in its database, the platform_admins row, and the user in its staff
# pool, which emails them Cognito's invitation (a temporary password; S12). The database URL is
# read from SSM into this process only, never printed.
source "$(dirname "$0")/lib.sh"

email="${2:?usage: $0 <env> <email> [name]}"
param() { aws ssm get-parameter --name "$PREFIX/$1" "${@:2}" --query Parameter.Value --output text; }
STAFF_POOL_ID="$(param cognito-user-pool-id)"
DATABASE_URL="$(param database-url --with-decryption)"
export STAFF_POOL_ID DATABASE_URL
# The Go AWS SDK can't always refresh an SSO sign-in that the CLI can, so a named profile's
# credentials are handed to it as the CLI has them (into this process only, never printed).
if [ -n "${AWS_PROFILE:-}" ]; then eval "$(aws configure export-credentials --format env)"; fi
cd "$REPO/apps/api"
go run ./cmd/platform-admin add "$email" "${3:-}"
