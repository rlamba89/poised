#!/usr/bin/env bash
# The pipeline's E2E step (S12): fetches QA's test users' sign-ins (cmd/qa-users) into
# e2e/.qa-users.json, which the journeys read to sign in through Cognito. Never printed; the file
# is gitignored. Locally: make e2e-users ENV=uk-qa PROFILE=…
source "$(dirname "$0")/lib.sh"

out="$REPO/e2e/.qa-users.json"
umask 077
aws ssm get-parameter --name "$PREFIX/test-users" --with-decryption --query Parameter.Value --output text >"$out"
echo "The test users' sign-ins are in e2e/.qa-users.json ($(python3 -c 'import json,sys; print(len(json.load(open(sys.argv[1]))))' "$out") users)."
