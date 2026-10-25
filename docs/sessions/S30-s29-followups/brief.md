# S30: Follow-ups after S29

**Plan:** none of its own. Three of S29's verification findings, which Rahul asked to fix on 25 Oct ([pending.md](../../pending.md), "From S29's verification 1"), built before plan C5.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S29 verified.

**Asked:** nothing. Rahul, 25 Oct: fix J8 and both follow-ups. Everything else stays as decided (decisions.md §2; nothing here reopens a decision).

## Build

- **The plain network message for any 5xx without an API error** (`apps/web/src/lib/api.ts`): on AWS, API Gateway answers a 5xx with JSON (`{"message": …}`), so today it shows "Something went wrong". Give "We couldn't reach the server. Check your connection and try again." for a failed `fetch` and for any 5xx whose body has no API `error`. The API's own error message still wins.
- **The "answers were cleared" card hides only network errors** (`apps/web/src/features/patient/PatientHQ.tsx`): today it hides every error banner, so a Reload refused for another reason (for example, the episode was cancelled meanwhile) shows nothing. Hide only `NETWORK_ERROR` there; any other error shows as usual.
- **J8 stops tripping on old test data** (`e2e/journeys/test_j8_reminders.py`): it picks a mobile from only 1,000 (`+447700900NNN`, the fictional range), so on a long-lived database `captured(mobile)` can find another run's texts. Look only at this episode's texts (or this run's), rather than widening the number range, which has only 1,000 fictional numbers.
- **Unit tests:** `api` turns a JSON 502 or 503 without `error` into the plain message, and keeps an API's own `error`; the cleared card still shows a non-network error.
- **End-to-end:** J8 passes repeatedly on a database that already holds earlier J8 runs.

## Not in this session

- An action ("What") filter on the Audit page: Rahul, 25 Oct, not a priority, revisit later.
- The other S29 findings in pending.md (old label on a refused press, `reportedBmi` tidy-ups, `EpisodeList` tidy-up).
- Rahul's hand checks: he'll do them in one go later.

## Manual test focus

1. Make every `/api` request answer a JSON 502 (`{"message":"Internal Server Error"}`), e.g. by overriding `window.fetch`: a staff page and a patient page show "We couldn't reach the server…".
2. An API error with its own message (e.g. a 409 or 422) still shows that message.
3. After Start fresh, a failed reload shows only the cleared card; a Reload refused because the episode was cancelled meanwhile shows that message.
