#!/usr/bin/env bash
# make deploy-web ENV=<environment>: uploads apps/web/dist to the environment's Amplify app as
# a manual deployment (create-deployment → upload the zip → start-deployment) and waits for it.
source "$(dirname "$0")/lib.sh"

dist="$REPO/apps/web/dist"
[ -f "$dist/index.html" ] || { echo "$dist/index.html is missing: run npm run build -w apps/web" >&2; exit 1; }
app_id="$(stack_output "poised-$ENV_NAME-web" AppId)"
[ -n "$app_id" ] && [ "$app_id" != None ] || { echo "No poised-$ENV_NAME-web stack: run make deploy-stacks first" >&2; exit 1; }

zip="$(mktemp -d)/web.zip"
(cd "$dist" && zip -qr "$zip" .)
read -r job url < <(aws amplify create-deployment --app-id "$app_id" --branch-name main \
  --query '[jobId,zipUploadUrl]' --output text)
curl -sSf -X PUT -H 'Content-Type: application/zip' --upload-file "$zip" "$url" >/dev/null
aws amplify start-deployment --app-id "$app_id" --branch-name main --job-id "$job" >/dev/null
echo "Deploying job $job to $(stack_output "poised-$ENV_NAME-web" AppUrl) …"
for _ in $(seq 60); do
  status="$(aws amplify get-job --app-id "$app_id" --branch-name main --job-id "$job" --query job.summary.status --output text)"
  case "$status" in
    SUCCEED) echo "Deployed."; exit 0 ;;
    FAILED | CANCELLED) echo "Deployment $status" >&2; exit 1 ;;
  esac
  sleep 5
done
echo "Still $status after 5 minutes" >&2
exit 1
