#!/usr/bin/env bash
# QA's test recipients (S17): the only people QA really sends texts and emails to (everything is
# still captured too). Their mobiles and addresses go only into the SSM SecureString
# <prefix>/messaging-test-recipients and to AWS's verification calls: never into the repo, and
# this script never prints them. QA only; the entry must have qa_test_recipients=True.
#
#   test-recipients.sh <env> set          TEST_RECIPIENTS='<+44… mobile>,<email>' (comma-separated):
#                                         stores them, and starts the sandbox's verifications: SES
#                                         emails each address a link, and each mobile gets a code
#   test-recipients.sh <env> verify-sms   CODE=<the code the mobile got>
#   test-recipients.sh <env> status       how many are stored, and the verifications' state
#   test-recipients.sh <env> off          deletes the parameter: QA really sends to no one
#
# In SES's and End User Messaging SMS's sandbox only verified recipients can be sent to; with
# production access the verifications aren't needed (QA still sends only to these).
source "$(dirname "$0")/lib.sh"

[ "$(field stage)" = qa ] || { echo "Test recipients are for QA only." >&2; exit 1; }
[ "$(field qa_test_recipients)" = True ] || { echo "$ENV_NAME's entry hasn't qa_test_recipients=True (countries.py)." >&2; exit 1; }
PARAM="$PREFIX/messaging-test-recipients"
SENDER_ID="$(field sms_sender_id)"
cmd="${2:?usage: $0 <env> set | verify-sms | status | off}"

# recycle makes each function start afresh, so it reads the test recipients now rather than at
# its next cold start: a configuration change (here only the description, which CDK doesn't set)
# retires the warm instances.
recycle() {
  for fn in staff-api patient-api jobs; do
    aws lambda update-function-configuration --function-name "poised-$ENV_NAME-$fn" \
      --description "Test recipients last changed $(date -u +%Y-%m-%dT%H:%MZ) (make test-recipients)" >/dev/null
  done
  echo "The functions read the change from their next request."
}

# The pending SMS verifications (their IDs, never their numbers).
pending_sms() {
  aws pinpoint-sms-voice-v2 describe-verified-destination-numbers \
    --query "VerifiedDestinationNumbers[?Status=='PENDING'].VerifiedDestinationNumberId" --output text
}

case "$cmd" in
set)
  list="${TEST_RECIPIENTS:?TEST_RECIPIENTS='<+44… mobile>,<email>' is needed}"
  aws ssm put-parameter --name "$PARAM" --type SecureString --value "$list" --overwrite >/dev/null
  aws ssm add-tags-to-resource --resource-type Parameter --resource-id "$PARAM" --tags Key=project,Value=poised
  echo "Stored $(tr ',' '\n' <<<"$list" | grep -c .) test recipients in $PARAM."
  recycle
  failed=0
  IFS=',' read -ra items <<<"$list"
  for item in "${items[@]}"; do
    item="$(tr -d '[:space:]' <<<"$item")"
    if [[ "$item" == +* ]]; then
      known="$(aws pinpoint-sms-voice-v2 describe-verified-destination-numbers \
        --destination-phone-numbers "$item" --query 'VerifiedDestinationNumbers[0].[VerifiedDestinationNumberId,Status]' \
        --output text 2>/dev/null || echo None)" # not found is an error, whose message holds the number
      id="${known%%[[:space:]]*}"
      if [ "$id" = None ] || [ -z "$id" ]; then
        id="$(aws pinpoint-sms-voice-v2 create-verified-destination-number --destination-phone-number "$item" \
          --tags Key=project,Value=poised --query VerifiedDestinationNumberId --output text 2>/dev/null)" \
          || { echo "AWS refused to add a mobile for verification (its message isn't shown: it holds the number)." >&2; exit 1; }
      elif [[ "$known" == *VERIFIED ]] && [[ "$known" != *PENDING ]]; then
        echo "A mobile is already verified."
        continue
      fi
      origin=()
      [ "$SENDER_ID" = None ] || origin=(--origination-identity "$SENDER_ID") # the entry's sender ID, if any
      aws pinpoint-sms-voice-v2 send-destination-number-verification-code \
        --verified-destination-number-id "$id" --verification-channel TEXT "${origin[@]}" >/dev/null 2>&1 \
        || { echo "AWS couldn't text a mobile its verification code (NO_ORIGINATION_IDENTITIES_FOUND: the SMS sandbox needs a sender ID or number first; see README)." >&2; failed=1; continue; }
      echo "A mobile was sent a verification code: run make test-recipients-verify CODE=<the code>."
    else
      if aws sesv2 get-email-identity --email-identity "$item" >/dev/null 2>&1; then
        echo "An address is already an SES identity (the sender's, or verified before)."
      else
        aws sesv2 create-email-identity --email-identity "$item" --tags Key=project,Value=poised >/dev/null 2>&1 \
          || { echo "SES refused to add an address (its message isn't shown: it holds the address)." >&2; exit 1; }
        echo "SES emailed an address a link: click it to verify."
      fi
    fi
  done
  exit "$failed"
  ;;
verify-sms)
  code="${CODE:?CODE=<the code the mobile got> is needed}"
  ids="$(pending_sms)"
  [ -n "$ids" ] || { echo "No mobile is waiting for a code."; exit 0; }
  for id in $ids; do
    if aws pinpoint-sms-voice-v2 verify-destination-number --verified-destination-number-id "$id" \
      --verification-code "$code" >/dev/null 2>&1; then
      echo "A mobile is verified."
    else
      echo "That code didn't verify a pending mobile."
    fi
  done
  ;;
status)
  if has_param "$PARAM"; then
    n="$(aws ssm get-parameter --name "$PARAM" --with-decryption --query Parameter.Value --output text | tr ',' '\n' | grep -c .)"
    echo "$n test recipients are stored in $PARAM."
  else
    echo "No test recipients: QA really sends to no one."
  fi
  echo "SMS verifications: $(aws pinpoint-sms-voice-v2 describe-verified-destination-numbers --query 'VerifiedDestinationNumbers[].Status' --output text)"
  ;;
off)
  if aws ssm delete-parameter --name "$PARAM" 2>/dev/null; then
    echo "Deleted $PARAM: QA really sends to no one."
    recycle
  else
    echo "There was no $PARAM."
  fi
  ;;
*)
  echo "usage: $0 <env> set | verify-sms | status | off" >&2
  exit 2
  ;;
esac
