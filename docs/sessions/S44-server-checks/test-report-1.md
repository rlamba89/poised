# S44 test report 1: the server's checks on send and on Complete review

**Summary: 15 passed, 0 failed, 0 blocked, 2 skipped (optional regressions R-01 and R-04), plus 1 hand check (phone size).**

- Commit `de4859a` with the S44 working tree uncommitted (the builder's changes); 27 Oct 2026, 06:09–06:30 GMT.
- App: http://localhost:3000 (web), API :8080, evaluate service on 127.0.0.1:8090, all from the conductor's `make dev`.
- Browser: Claude in Chrome ("rahul chrome"). Signed in as Cara Clinician, Alex Author, Val Clinician and Bea Author through the stub dev login. Patients signed in through their invite links (`/i/<token>`: Continue, then date of birth). The rate-limited sign-in link wasn't used.
- **Phone size couldn't be set.** `resize_window` reported success, but the window stayed 1400×717 (it looks maximised or full screen). The patient cases ran at desktop width. Phone layout isn't what this session changes, so it's listed as a hand check, not a failure.

## How the data was made

- **T-01 was done fully in the UI:** New episode form; the patient opened the invite, entered the date of birth, answered every question and pressed Send my answers. The GIF is of this case.
- **Later episodes were made with the same `POST …/episodes` request the New episode page sends,** from a signed-in staff tab, to save time. Invite tokens were read from `/api/qa/messages`.
- **Patients mostly signed in with the same two calls the invite page makes** (`/continue`, `/dob`).
- **Answers were given in the page** (radio clicks, typing, Finish this section, Send) in T-01 to T-09. T-11, T-12, R-03 and exploration X1 saved them with the patient's own `PUT …/answers` instead.
- **The S44 test HQ was made through the API, then checked and finished in the editor UI.** It has the three sets with their audiences and with conditions on the snore question. In the editor, Nurse checks showed Clinician and "Conditionally: Do you snore? is Yes". In Rules, the no-ASA-counts-as-ASA-I box was unticked, then the HQ was published with Publish. Its name is "S44 test 0625".
  - **The first publish was refused** by the S23 key lock: `q_snore` already exists in Trust A with other options. The keys were renamed to `q_s44snore0625`, `q_s44tired0625` and `q_s44bp0625`. That's expected behaviour, not a bug.
- **Database reads (psql)** were used only to confirm: no `form_outputs` row and `submitted_at` still empty after a refused send, and that an evaluator failure was stored.
- **The API's log:** `make dev`'s output file (`…/tasks/blwrocmkz.output`) could be read, so T-11's log bullet was checked rather than left as a hand check.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 normal send (phone) | Pass | The "Thank you, Mia" screen. Cara sees **Ready for review**, which the plan allows: Trust A's Demo pre-op HQ v5 has no rules in this database (see Notes), so nothing auto-triages. Phone size: hand check. |
| T-02 patient send, one set unfinished | Pass | `409 {"error":"Please finish “Your health” before sending your answers."}`. After a reload the page still shows the sections, About you Complete. Episode **HQ not complete**; no "HQ completed" note; the status history has only Created. DB: form `open`, `submitted_at` empty, 0 `form_outputs`. |
| T-03 two sets unfinished, then finishing | Pass | `409 … “About you” and “Your health” …`. After finishing every set in the page, Send worked and the episode is Ready for review. |
| T-04 staff Fill in send refused | Pass | `409 … “Your health” …`; still HQ not complete, and the only notes are "Episode created" and "Invite sent by email". Finishing Your health on the Fill in page, then Send the answers, worked ("Sent for Mia …"). |
| T-05 hidden set needn't be finished | Pass | With snore = No, only About you is listed; the send works and the episode is Ready for review. With snore = Yes and Sleep unanswered: `409 … “Sleep” …`, HQ not complete. |
| T-06 Complete review with sets not validated | Pass | About you validated: `409 … “Sleep” and “Nurse checks” aren't validated.`, still Ready for review with no review note. Sleep validated too: `409 … “Nurse checks” isn't validated.` After Nurse checks (120/80), Complete HQ review on the page: **Ready for POA**, "HQ review completed by Cara Clinician". |
| T-07 nothing validated | Pass | `409 … “About you”, “Sleep” and “Nurse checks” aren't validated.`; status unchanged. |
| T-08 set hidden by its condition | Pass | snore = No: the card lists only About you. Validated without changes, then Complete HQ review: **Ready for POA**. |
| T-09 clinician's corrected answers decide | Pass | Cara changed snore No → Yes and validated; the card then listed Sleep and Nurse checks. The request gave `409 … “Sleep” and “Nurse checks” aren't validated.` After validating both: completes, Ready for POA. |
| T-10 refused paths as before | Pass | 1: Val, A1 episode → `404 {"error":"Hospital not found."}`. That's 404 as expected; the wording comes from the hospital check, which runs before the episode lookup. 2: no cookies → `401`. 3: HQ not complete → `409 "The patient hasn't sent their answers yet."` 4: sending again → `409 "These answers have already been sent."` (on T-09's sent episode, whose patient session was the current one). |
| T-11 evaluate service stopped | Pass | `kill $(lsof -ti tcp:8090 -sTCP:LISTEN)`: health refused, web and API still up. A send with About you only gave `204`. Ready for review, with General notes "The automatic checks didn't run. Review this HQ by hand." (Automatic on 27/10/2026, 06:25). Complete review with nothing validated: `204`, Ready for POA, "HQ review completed by Cara Clinician". The log lines (below) hold no answers. |
| T-12 restart, checks back | Pass | `make evaluate` started in the background; health `{"ok":true,…}`. T-02 repeated on a new episode: `409 … “Your health” …`, HQ not complete. |
| R-01 (S34) video tick | Skipped | No HQ with a required video tick to hand (optional). |
| R-02 (S41) Suggested ASA line | Pass | T-01's episode shows "No suggested ASA: no answer gives an ASA grade", and nothing new. |
| R-03 (S42) auto-triage on a complete send | Pass | Run on **Trust B's** Demo pre-op HQ v1 as Bea Author, because Trust A's v5 has no rules here. The incomplete send was refused (`409 … “Your health” …`). The complete send: **Ready for admission**, "Auto-triaged by the rule “Low risk”: no ASA grade, no red or amber flags → Ready for admission". It's on Hospital B1's **Auto-triaged** worklist. |
| R-04 (S43) flags and rule task | Skipped | No high-risk HQ to hand (optional). |
| R-05 (S20) Re-open, then Complete again | Pass | Alex re-opened T-06's episode (reason "MT S44 R-05 recheck"). The sets stayed validated, and Complete HQ review worked again: Ready for POA, "HQ review completed by Alex Author". |

## Failures

None.

## Hand checks for Rahul

- **The patient's form at 390×844 (T-01, T-02, T-05):** the browser window couldn't be resized from the extension. S44 adds nothing visible, so this only rechecks the layout from earlier sessions.

## Found while exploring

1. **Clinician hides sets by correcting the answer:** this is the reverse of T-09, and it works.
   - On a snore = Yes send with Sleep answered and nothing validated, Cara changed About you to No and validated it. The card then listed only About you.
   - Sending `complete-review` straight to the API gave **204** and moved the episode to Ready for POA. Sleep and Nurse checks, now hidden, didn't need validating.
2. **Two sends at once:** two parallel patient `submit` requests on a complete form gave one `204` and one `409 "These answers have already been sent."`. There's exactly one "HQ completed by the patient" note and one HQ sent row in the status history.
3. **Two Complete reviews at once:**
   - With nothing validated, both requests gave the same 409 naming About you.
   - Once it was validated, one gave `204` and the other `409 "The HQ review is already complete, so the answers can't be changed."`. That wording is a little odd for a repeated Complete review. **It isn't new in S44:** the line is unchanged context in `validation.go`.
4. **Cosmetic, probably older:** after a send, `get_page_text` on the patient's page still finds the "Send your answers?" modal's text in the DOM below "Thank you, Mia". Screenshots show the modal closed, so it's most likely just closed and left in the page. It isn't visible.
5. **Validating a set the patient never saw** (T-09's Sleep) shows an amber "The patient didn't answer this Question Set." banner. That reads well.

## Notes on the data (not bugs)

- **Trust A's published "Demo pre-op HQ" is v5,** last changed on 23 Oct, and has no rules. Re-seeding doesn't overwrite it, so a Trust A Demo pre-op send goes to Ready for review, not auto-triage. T-01 allows either, and R-03 was checked on Trust B's v1.
- **Trust A's Demo pre-op v5 includes a clinician set, "Nurse checks",** so Complete review on it needs all three sets validated.

## API log lines (T-11), from `make dev`'s output

```
2026/10/27 06:25:25 EVALUATE_FAILED form 9d5a67e6-1bda-4270-96a7-61d3400533da: evaluate: /evaluate: Post "http://127.0.0.1:8090/evaluate": dial tcp 127.0.0.1:8090: connect: connection refused
2026/10/27 06:25:37 EVALUATE_FAILED complete review, form 9d5a67e6-1bda-4270-96a7-61d3400533da: evaluate: /shown: Post "http://127.0.0.1:8090/shown": dial tcp 127.0.0.1:8090: connect: connection refused
```

The stored `form_outputs.error` for that form holds the same `/evaluate` message. Stopping the service also made `make dev`'s output show `npm error … code 143` for the evaluate workspace, which is expected from the kill. The API and web carried on.

## Console and network errors

None seen in the console of the staff and patient tabs (tracking started partway through the run). The only 4xx responses were the ones the cases asked for.

## Evaluate service at the end of the round

Exactly one copy is listening on 127.0.0.1:8090. It's the copy started for T-12 (`make evaluate`, log `/tmp/mt-s44-evaluate.log`), not `make dev`'s own, which T-11 stopped. Its process chain:

- `make evaluate`: PID **68797** (parent 1)
- `npm run evaluate`: PID **68806**
- `npm run serve`: PID **68832**
- `sh -c node build.mjs && node dist/evaluate.cjs`: PID **68866**
- `node dist/evaluate.cjs`: PID **68872** (the listener)

No zsh wrapper is left: the background subshell exited.

## GIFs

- `/Users/rahul/Downloads/S44-T01.gif`: the T-01 happy path (the patient answers both sections and sends).

## Test data made

- Patients "Mia MT S44 …" at Hospital A1: 0610 Tone, 0615 Two, 0616 Three, 0618 Four, 0630 T5a/T5b/T6/T7/T9, 0640 X1, 0650 T11/T12.
- "Mia MT S44 0645 R03" at Hospital B1.
- The HQ "S44 test 0625" v1, published in Trust A.
