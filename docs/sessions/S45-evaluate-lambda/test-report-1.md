# S45 test report 1

**12 passed, 0 failed, 0 blocked** (T-01 to T-04, R-01 to R-04; H-1 and H-2 are Rahul's, as the plan says).

- **Where:** QA, https://main.d33kh0gi8vk83v.amplifyapp.com, running S45's working tree as deployed by hand on 29 Oct. The local tree is at `655e3ce` with 34 uncommitted changes (S45's), and the evaluate function's health check says `clinical@655e3ce8a31c-dirty`.
- **When:** 29 Oct 2026, 12:56 to 13:25 GMT.
- **Browser:** Claude in Chrome. The window wouldn't go narrower than **503×701**, so the patient steps ran at 503 px wide, not 390×844. The layout was fine at that width.
- **Sign-in:** Rahul signed in each staff user by hand on Cognito's page (Cara, Sam, Cara, Alex). The tester never handled a password or code. Sam was only used to read `/qa/messages`.
- **HQ:** `E2E J11b 2910 122027 1866`, v1 at the start. T-03 published v2. It has three sets: About you, Your heart (patient) and Nurse checks (clinician).

## Test data (made up)

| Episode | Date of birth | Used for |
| --- | --- | --- |
| MT S45 1301 Low | 12 May 1980, male | T-01, R-01 |
| MT S45 1302 High | 20 Mar 1975, male | T-02, T-03 step 1, R-02, R-03 |
| MT S45 1303 Change | 2 Aug 1982, female | R-04 |
| MT S45 1304 Spare | 15 Nov 1970, male | exploring (left half-answered) |

Their contact details were `mt-s45-130N-…@example.com` and `07700 90030N`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Low risk auto-triaged | Pass | The patient saw only "Thank you, MT S45 1301". The episode was on Auto-triaged at Ready for admission. The card says "Auto-triaged by the rule “Low risk”", and the history says "Auto-triaged: Ready for review → Ready for admission, Automatic (Low risk)". There was no "automatic checks didn't run" note. **Checked** then showed "Checked by Cara Clinician on 29/10/2026, 13:17", and the episode left the list. |
| T-02 High risk: task and flag | Pass | On Anaesthetic review needed, at Ready for review, with the red ANAESTHETIC REVIEW badge on the row. The flag says "Added by the rule “High risk” on 29/10/2026, 13:11". The task **Book anaesthetic review** is for the ANAESTHETIST team ("Added by rule: High risk"). The page also says "Ready for admission needs the anaesthetist's ASA grade first (the rule “High risk”)". The General notes list the task, the flag and the requirement, and have no "checks didn't run" note. |
| T-03 step 1 Complete review | Pass | After all three sets were validated, Complete HQ review gave "HQ review completed by Cara Clinician on 29/10/2026, 13:19", with no error. |
| T-03 step 2 Publish | Pass | As Alex: **Create new version**, then I changed the wording "Have you been told you have heart failure?" to end with "(MT S45)" and clicked **Publish**. The window says "This version auto-triages: ASA I–II, no red flags → Ready for admission…". Publishing gave V2 · PUBLISHED, read only (`POST …/publish` 204 in 496 ms). |
| T-04 Logs hold no answers | Pass | The evaluate log group, read with `aws logs tail`, holds only INIT_START, START, END and REPORT lines and `evaluate: evaluate form <id>: 200 in N ms`, `shown form <id>: 200 in 0 ms`, `publish-problems: 200 in 2 ms` and `health`. Searching it for the test names, dates of birth, emails, mobiles and words like smoke, snore, heart, failure, yes and nurse found nothing. |
| R-01 Auto-triaged count | Pass | 0 before the sends, 2 after (Low and Change), and 1 after Checked. |
| R-02 Clear flag | Pass | An empty reason is refused with "Say why the flag is cleared." With a reason (submitted with Enter), the badge goes and "Cleared by Cara Clinician on 29/10/2026, 13:19: MT S45: anaesthetist review booked" stays. The POA Summary's notes have the "Flag cleared…" line. |
| R-03 ASA III then Ready for admission | Pass | Before the grade, the status menu shows "Ready for admission (add the anaesthetist's ASA grade first)" greyed out. On the POA Summary I added ASA III ("Counts", given by Cara Clinician). The change to **Ready for admission** then went through, and Undo was offered. |
| R-04 Smoke Yes then No | Pass | Changing smoke from Yes to No changed nothing else on the page. The send was auto-triaged by "Low risk" (no ASA grade, no flags). There was no leftover smoker ASA. |

The plan didn't ask for these, but I checked them while there:
- **Complete review refused before Nurse checks was validated:** I called `POST …/complete-review` from the page with the button still greyed out. It returned **409** "Validate every Question Set first: “Nurse checks” isn't validated.", so the Lambda's `shown` check works.
- **No EVALUATE_FAILED:** staff-api and patient-api logged none in the last hour.
- **Alarms:** `delivery-failures`, `evaluate-failures` and `jobs-failed` are all **OK**.

## Failures

None.

## Found while exploring

- **The evaluate Lambda's times on QA, from my sends** (REPORT lines, not timed by feel):

  | Call | Time |
  | --- | --- |
  | The first send, cold | 278 ms, plus 181 ms Init |
  | The other two sends | 136 ms, 122 ms |
  | `shown` | 3 ms |
  | `publish-problems` | 4 ms |

  Memory used stayed at 130–150 MB of 1024 MB. Patient-api's whole submit took 787 ms on the cold send. On the warm one, the browser timed the send at 297 ms end to end.
- **Double-clicking the confirm Send button** (T-02) sent once. There was one evaluate call, and the patient got "Thank you" with no error.
- **A wrong date of birth** gives "That date of birth doesn't match our records… You have 4 tries left." The right one then works.
- **Another patient's episode URL** (`/home/episodes/<Low's id>` while signed in as Spare) says "Episode not found."
- **Another hospital's URL** (Hospital A2, as Cara) says "You don't have access to this hospital."
- **Leaving mid-form and coming back** (Spare: smoke Yes, then I went away) shows About you as **In progress**, with the answer kept.
- **A nurse's note with leading spaces and an emoji** (`  MT S45 note 🙂 made-up test text`) saved and validated without trouble.
- **The tool, not the app:** Claude in Chrome's network log showed **503** for each `POST /api/p/episodes/…/submit`. The page's own `fetch` got **204** (297 ms) for the same request. API Gateway's `5xx` metric was 0 for those minutes, and no 503 reached the app. Not a finding.

## Console and network errors

- **Console:** no errors or warnings in the patient, clinician or author pages.
- **Network:** no unexpected 4xx or 5xx, apart from the tool's false 503s above. The only 409 was the deliberate Complete review call.

## GIFs

Saved in `~/Downloads`:
- `S45-T01-patient.gif` (the patient's low-risk send at 503 px)
- `S45-T01-clinician.gif` (Auto-triaged, open, Checked, the count goes down)
- `S45-T03-publish.gif` (New version, change the wording, the Publish window, published)

## Hand checks for Rahul

From the plan:
- **H-1:** a broken function, the note and the alarm email.
- **H-2:** a green pipeline after the push, and the health check naming the commit without `-dirty`.
