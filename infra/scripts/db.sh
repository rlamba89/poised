#!/usr/bin/env bash
# make aws-db ENV=<environment>: runs the migrations (goose up) and the seed (idempotent) against
# the environment's database, whose URL is read from SSM into this process only, never printed.
source "$(dirname "$0")/lib.sh"

DATABASE_URL="$(aws ssm get-parameter --name "$PREFIX/database-url" --with-decryption \
  --query Parameter.Value --output text)"
export DATABASE_URL
cd "$REPO/apps/api"
go tool goose -dir db/migrations postgres "$DATABASE_URL" up
go run ./cmd/seed
