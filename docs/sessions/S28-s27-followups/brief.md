# S28: Follow-ups after S27

**Plan:** none of its own. These are Rahul's answers to S27's open items ([decisions.md](../../decisions.md) §2, 24 Oct, "S27's open items answered"), built before plan C5.
**Requirements:** PRE-01–PRE-04 and PAT-09 as changed on 24 Oct; the S27 items settled in [pending.md](../../pending.md) ("From S27's test report 1" onwards).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S27 verified.

**Ask Rahul first:**
1. **When the staff-side join happens.** **Recommended:** when staff open **Fill in for the patient** (`behalf.view`), before the form is pre-filled, so it happens only when staff are actually filling in.
   - *Or:* when staff create the episode, so every new record that matches an account joins it, whether or not staff fill in.
2. **What the staff-side join may do.** **Recommended:** it joins **only this record**, and **only to an account that already exists** (one the patient made with their own proof). It never creates an account, never sweeps the person's other unjoined records (`joinMatching`), and **copies no contacts** either way, so Trust A staff never see a mobile or email that only Trust B holds. One `account.join` row in Trust A's log names the staff member.
   - *Or:* the same as the patient's own join (S15: it creates an account if none matches, joins other matching records, and fills in missing contacts).
3. **What "first open only" covers.** **Recommended:** the whole pre-fill runs **once per form**, on its first open (this trust's answers and other trusts'). A form records that it was pre-filled, and later opens never read earlier HQs again.
   - *Or:* only other trusts are limited to the first open; this trust's own HQs are still read on every open while a set has no answers (as S25 does today).

## Build

- **The four-field join on the patient's behalf** (`httpapi/accounts.go`, `httpapi/patient.go`, `httpapi/prefill.go`), as answered:
  - a record with no account joins the account whose **date of birth, mobile or email, first name and surname** all match (`FindAccountForRecord`, S15's rule, each ignoring case and spaces); no match means no join and only this trust's answers, as today;
  - after the join, other trusts' answers fill in as they do for the patient (S27: the card names the hospital, with Start fresh);
  - the join is audited under the record's trust (`account.join`, the staff member as actor);
  - answers 1–2 as agreed.
- **Another trust's audit log names only the trust.** A `prefill.read` row that Trust A staff cause shows in Trust B's log, its CSV and its "Who" filter as staff at Trust A, with no name or email. Trust A's own rows still name the person. The patient's own reads are unchanged.
- **Pre-fill on the first open only** (answer 3): later opens don't read other trusts again, so there's one `prefill.read` row per form per trust.
- **The POA Summary says when a Question Set was started fresh**, with Validate's wording, on both its tabs.
- **Small wording and badge changes:**
  - an archived episode's read-only View drops "A member of the team will review them." (`PatientHQ.tsx`);
  - a cancelled questionnaire the patient had already sent shows on their home as "The hospital has cancelled this. Please contact them if you have questions." (`PatientHome.tsx`); an unsent one keeps today's wording;
  - NOT RESPONDING is hidden once an episode is cancelled, as well as archived (`notResponding`, `status.ts`).
- **The POA Summary's "Patient reported BMI"** uses the patient's own height and weight until About you is validated, then the validated ones (`reportedBmi`, `packages/clinical/src/poa.ts`).
- **S27's small fixes** ([pending.md](../../pending.md)):
  - a `scroll-margin-top` the height of the staff top bar on SurveyJS question roots, so a jumped-to question's title isn't hidden;
  - `choose` (`PatientHQ.tsx`) reports a failed reload after Start fresh, instead of leaving the card up;
  - `restoreSpot` (`jump.ts`) stops polling when the POA Summary is left.
- **Integration tests:**
  - on behalf, a record matching an account on all four fields joins it and gets Trust B's answers; one field different gets no join and nothing from Trust B;
  - the staff join does only what answer 2 allows (no new account, no other records joined, no contacts copied);
  - Trust B's `prefill.read` row from a staff read names no Trust A user; Trust A's row does;
  - a second open writes no new `prefill.read` row;
  - the five cases for any new or changed route.
- **End-to-end:** J16b adds the on-behalf case: a patient with a Trust B history who never opens Trust A's link; Cara fills in for them and sees the card naming Hospital B1. J5 checks the Start fresh line on the POA Summary.

## Not in this session

- An account merge for one person's two accounts (later; C10).
- IG/DPO sign-off on copying across trusts and on the staff-side join (before go-live, C10).
- The print-designed POA Summary PDF (plan C9).

## Manual test focus

1. A patient with a Trust B history who has **never opened** a Trust A link: Cara creates the episode with matching details and fills in for them. The card names Hospital B1, and Start fresh works.
2. The same with one detail different (another surname, another mobile): nothing comes from Trust B, and no join appears in either log.
3. After a staff join, Trust A's patient page shows no mobile or email that only Trust B holds (if answer 2 is as recommended).
4. Trust B's audit log, its CSV and its "Who" filter: the staff read shows as staff at Trust A, with no name or email. Trust A's log names Cara.
5. Open the same form twice: Trust B's log has one `prefill.read` row for it.
6. Start fresh on one set, then open the POA Summary: the set says the patient started fresh.
7. Archive a sent episode: the View has no "the team will review them". Cancel a sent one: the patient's home card reads the new wording, and the NOT RESPONDING badge is gone.
8. Before validating About you, the POA Summary's Patient reported BMI shows a number from the patient's answers.
9. On a desktop staff page, "Everything is still correct" brings the question into view with its title showing.
