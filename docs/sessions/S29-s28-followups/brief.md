# S29: Follow-ups after S28

**Plan:** none of its own. These are Rahul's answers to S28's open items ([decisions.md](../../decisions.md) §2, 25 Oct, "the open items after S28 answered"), built before plan C5.
**Requirements:** PRE-01–PRE-04 as changed on 24–25 Oct; the items settled in [pending.md](../../pending.md) on 25 Oct ("From S28's test report 1", "From S28's verification 1", "From S22's test report 1", "From S24's test report 2").
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S28 verified.

**Asked:** one question (which sets a read after a later join may fill). Rahul's answer, 25 Oct: an edge case, don't build the read after a later join at all ([answers.md](answers.md)). Decision 1 is reversed; S28's "once per form" stands.

## Build

- **Patient reported BMI uses only shown answers** (decision 2; `reportedBmi`, `packages/clinical/src/poa.ts`): a BMI, height or weight whose question or set is hidden by a condition, for the answers in use, is skipped. This holds both before validation (the patient's answers) and after it (the validated ones).
- **An audit action for each opening-card button** (decision 3; `httpapi/audit.go`, the `behalf.prefill` and `link.prefill` routes): the button pressed (`prefill_choice`) picks the action. Builder's default labels:
  - staff: "Checked each pre-filled answer with the patient", "Confirmed pre-filled answers with the patient", "Started a Question Set fresh for the patient";
  - patient: "Patient chose to check each pre-filled answer", "Patient confirmed pre-filled answers", "Patient started a Question Set fresh".

  Rows written before S29 keep their old action and label (the log is append-only), and the audit page's What filter offers the new actions.
- **Plain wording for network errors** (decision 4; `apps/web/src/lib/api.ts`): when `fetch` itself fails, or a response has no JSON body and a 5xx status (a gateway error), `api` throws an `ApiError` with "We couldn't reach the server. Check your connection and try again." Every page that shows `ApiError`'s message gets this for free. In `PatientHQ.tsx`, the red banner is hidden while the "The answers in this section were cleared" card shows, since the card already explains what happened.
- **No open-task count on archived episodes** (decision 5): the episode list's row (`EpisodeList.tsx`, with Show archived) and the episode's Tasks title (`EpisodePage.tsx`) show no count once the episode is archived. The tasks themselves stay listed, as a record.
- **The cancelled message on an archived-from-Cancelled link** (decision 6; `httpapi/patient.go`): an old link's form for an episode archived from Cancelled answers with `formCancelled` (and its code), not `formArchived`. An episode archived from any other status keeps `formArchived`. *(S27 gave a sent questionnaire a read-only View once archived; that stays.)*
- **A warning in "Same as…"** (decision 8; `apps/web/src/features/editor/library.tsx`, `SameAsWindow`): when the question already has a key, from the library or a published HQ, the window says "This question is already linked to '<that question's text>'. Using this replaces that link." before Use this. A question with no key yet shows no warning.
- **Integration tests:**
  - each opening-card button writes its own audit action, for staff and for the patient;
  - an archived-from-Cancelled link answers with the cancelled message; an archived-from-Validated one keeps the closed message;
  - the five cases for any new or changed route.
- **Unit tests:** `reportedBmi` skips hidden answers; `api` turns a failed `fetch` and a non-JSON 502 into the plain message, and keeps the API's own message otherwise.
- **End-to-end:** J5 checks the plain network-error message on one staff page; the audit journey checks one opening-card button's own label.

## Not in this session

- A second pre-fill after a later join (decision 1, reversed 25 Oct: an edge case). S28's "once per form" stands.
- The catch-up reminder alongside Not responding: kept as designed (decision 7).
- An account merge for one person's two accounts (later; C10).
- IG/DPO sign-off on copying across trusts and on the staff-side join (before go-live, C10).

## Manual test focus

1. Each opening-card button, pressed by staff and by the patient, shows its own label in Trust A's audit log and in the What filter.
2. With the API stopped (or offline), a staff page and a patient page show "We couldn't reach the server…", not "Failed to fetch". After Start fresh with a failed reload, only the cleared card shows.
3. Show archived: an archived episode with an open task shows no count in the list or on its Tasks title.
4. Cancel an episode, archive it, then open the patient's old link: the cancelled message. Archive a validated one: the closed message.
5. In the editor, "Same as…" on a question that came from the library warns before Use this; on a new question it doesn't.
6. The POA Summary's Patient reported BMI ignores a height hidden by a condition (a test HQ with a condition on its height question).
