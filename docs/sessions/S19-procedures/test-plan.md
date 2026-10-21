# S19 manual test plan: procedures and episode dates

**Changes after test report 1:** T-13 is now expected to refuse a partly typed date on both New episode and Edit procedure and dates (F1, fixed). T-02/T-04: pressing Enter in the procedure form also sends it, and Edit on a lower row scrolls the form into view.

**Changes after test report 2:** T-06 step 3, an empty name, shows the app's "Enter the procedure's name." (not the browser's bubble), replacing any earlier message. Retire / Bring back on a lower row scrolls up to its message. Enter in Edit procedure and dates saves it.

**Changes after test report 3:** after Retire or Bring back on a lower row, the page jumps (instantly) to the top, where the message is, and stays there once the list has reloaded.

For a tester who hasn't seen the code: use the browser only.

## Setup

1. In your own terminal, from the repo root:
   ```sh
   docker compose up -d db
   make migrate seed
   make dev
   ```
   Leave `make dev` running. (In an automatic run, the conductor does this.)
2. Open **http://localhost:3000**. It shows the stub sign-in page: click a person's name to sign in as them. To change person, use **Sign out** (top right) and pick another.
3. **Seeded people** (made up):

   | Name | Role | Where |
   | --- | --- | --- |
   | Ada Admin | admin | the whole of Trust A |
   | Alex Author | super clinician | the whole of Trust A |
   | Cara Clinician | clinician | Hospital A1 only |
   | Sue Super | super clinician | Hospital A1 only |
   | Val Clinician | clinician | Hospital A2 only |
   | Bea Author | super clinician | the whole of Trust B |

4. **Seeded data for this session:**
   - Each trust has a published HQ called **Demo pre-op HQ** (v1).
   - **Trust A's procedures:** Total hip replacement (HIP-01, Orthopaedics, default HQ: Demo pre-op HQ); Knee arthroscopy (KNEE-02, Orthopaedics, Demo pre-op HQ); Laparoscopic cholecystectomy (no code, General surgery, **no default HQ**); Cataract surgery (EYE-01, Ophthalmology, **Hospital A1 only**, Demo pre-op HQ).
   - **Trust B's procedures:** Total hip replacement, Inguinal hernia repair.
   - The local database may also hold procedures and HQs named "E2E …" from the automated journeys; ignore them. Some are retired.
5. **Captured messages:** texts and emails aren't sent. **http://localhost:3000/qa/messages** lists them, with the patient's invite link.
6. **Dates** show in the deployment's locale, en-GB locally: e.g. 3 November 2026 shows as **03/11/2026**. Date fields are the browser's date picker; you can type into them.

## Test cases

### Managing the procedure list

**T-01: A super clinician of the whole trust sees Procedures.**
1. Sign in as **Alex Author**, choose **Hospital A1** if asked.
2. In the header, click **Procedures**.
- **Expected:** the Procedures page lists Trust A's four seeded procedures (plus any "E2E …" ones), each with its code (or —), specialty, "Offered at" (**Every hospital**, or **Hospital A1** for Cataract surgery) and default HQ (**Demo pre-op HQ**, or **The hospital's default** for Laparoscopic cholecystectomy). Retired ones are last, marked **Retired**. No Trust B procedure (e.g. Inguinal hernia repair) appears.

**T-02: Add a procedure with a code and a default HQ.**
1. As Alex on Procedures, click **Add procedure**.
2. Name `Test hip revision`, Code `HIP-02`, Specialty `Orthopaedics`, Offered at **Every hospital of the trust**, Default HQ **Demo pre-op HQ**. Click **Add procedure**.
- **Expected:** "Test hip revision is added." The row shows HIP-02, Orthopaedics, Every hospital, Demo pre-op HQ.

**T-03: Add a procedure without a code, for Hospital A1 only.**
1. **Add procedure**: Name `Test grommets`, leave Code and Specialty empty, Offered at **Hospital A1 only**, Default HQ **None: use the hospital's default HQ**. Add it.
- **Expected:** the row shows — for code and specialty, Hospital A1, and "The hospital's default".

**T-04: Edit a procedure.**
1. Click **Edit** on Test hip revision. The form shows its values.
2. Change Specialty to `Trauma and orthopaedics` and the Code to `HIP-03`. Click **Save changes**.
3. Reload the page.
- **Expected:** "Test hip revision is saved."; after the reload the row still shows HIP-03 and Trauma and orthopaedics. **Cancel** on an edit changes nothing.

**T-05: Retire and bring back.**
1. Click **Retire** on Test hip revision.
- **Expected:** "Test hip revision is retired: it isn't offered for new episodes." The row moves to the bottom, marked **Retired**, with **Bring back**.
2. Click **Bring back**. **Expected:** "…is offered again."; it's no longer marked Retired. Retire it again for T-12.

**T-06: Invalid input gives a clear message.**
1. **Add procedure** with Name `total HIP replacement` (an existing name in another case). **Expected:** a red message: "The trust already has a procedure with this name (retired ones included)." Nothing is added.
2. Try the name of the retired Test hip revision. **Expected:** the same message.
3. Leave the Name empty and add. **Expected:** "Enter the procedure's name."
4. Edit a procedure and rename it to another existing procedure's name. **Expected:** the same duplicate message; the procedure keeps its name.
5. Open the **Default HQ** dropdown. **Expected:** only Trust A's **published** questionnaires (Demo pre-op HQ and any published E2E ones); no draft questionnaire, and nothing of Trust B's. (A draft or another trust's HQ sent to the API directly is refused with "Default HQ: choose one of the trust's published questionnaires." That's covered by the automated tests.)

**T-07: An admin manages it too; others don't.**
1. Sign in as **Ada Admin**: **Procedures** is in the header and she can add or edit (add `Test tonsillectomy`).
2. Sign in as **Cara Clinician** and as **Sue Super** (Hospital A1 only): **no Procedures** link in the header.
3. As Sue, type the address `…/procedures` (her hospital's URL with `/procedures` at the end) directly.
- **Expected:** Ada can; Cara and Sue have no link, and Sue's deep link shows "Only super clinicians and admins of the whole trust can manage its procedures", with no list.

**T-08: Another trust's list isn't visible.**
1. Sign in as **Bea Author** (Trust B) and open **Procedures**.
- **Expected:** only Trust B's procedures (Total hip replacement, Inguinal hernia repair); none of Trust A's (no Cataract surgery, no Test … ones).
2. As Bea, paste Alex's Procedures address (Trust A's, from T-01) into the address bar.
- **Expected:** "You don't have access to this hospital" (or similar); nothing of Trust A's.

### New episodes

**T-09: Picking a procedure chooses its default HQ; the HQ can still be changed.**
1. Sign in as **Cara Clinician** (Hospital A1). **Episodes → New episode**.
2. Type a new made-up patient: First name `Pat`, Surname `Procedure<anything unique>`, Date of birth `1975-04-02`, Sex Female, Mobile `07700 900555`, Email `pat.s19@example.test`.
3. Open **Procedure**. **Expected:** Trust A's procedures offered at A1, by name with the code in brackets, e.g. "Total hip replacement (HIP-01)", "Cataract surgery (EYE-01)", "Test grommets"; **no** retired ones (no Test hip revision).
4. Pick **Total hip replacement (HIP-01)**. **Expected:** **Health questionnaire** becomes **Demo pre-op HQ v1**.
5. Change **Health questionnaire** by hand to another published HQ (e.g. any "E2E …" one), if there is one. **Expected:** it stays as you chose; the procedure stays picked.
6. Set **Planned procedure date** `2026-11-03` and **POA appointment date** `2026-10-28`. Click **Create episode**.
- **Expected:** the episode page shows Procedure **Total hip replacement**, Planned procedure date **03/11/2026**, POA appointment date **28/10/2026**, and the HQ you chose last.

**T-10: A procedure with no default HQ falls back to the hospital's default HQ.**
1. Sign in as **Ada Admin**, open **Settings**, choose **Hospital A1**, set **Default health questionnaire** to a published HQ that isn't Demo pre-op HQ if there is one (else Demo pre-op HQ), and save.
2. Sign in as **Cara**, **New episode**.
- **Expected:** the page opens with that HQ already chosen (the hospital's default).
3. Pick **Total hip replacement**. **Expected:** the HQ becomes Demo pre-op HQ v1.
4. Pick **Laparoscopic cholecystectomy** (no default HQ). **Expected:** the HQ becomes the hospital's default from step 1.
5. Clear the procedure (the × in the field). **Expected:** the HQ stays as it is.
6. (Clean up: as Ada, set Hospital A1's default HQ back to empty.)

**T-11: A hospital-only procedure isn't offered at the trust's other hospitals.**
1. Sign in as **Val Clinician** (Hospital A2). **New episode**, open **Procedure** and type `Cataract`, then `grommets`.
- **Expected:** "No procedure of that name" for both. Total hip replacement, Knee arthroscopy and Laparoscopic cholecystectomy are offered.

**T-12: A retired procedure isn't offered, but an episode that has it still shows it.**
1. As **Cara**, create an episode (any patient) with procedure **Test grommets**.
2. As **Alex**, retire **Test grommets** on Procedures.
3. As **Cara**: New episode → **Procedure**: type `grommets`. **Expected:** not offered.
4. Open the episode from step 1. **Expected:** Procedure still shows **Test grommets**; the Episodes list shows it too. Click **Edit procedure and dates**: the Procedure dropdown shows "Test grommets (no longer offered)". Change only the POA date and Save: **Expected:** saved, the procedure is kept.

**T-13: Invalid dates.**
1. On New episode, type only part of a date (e.g. day and month, leaving `yyyy`) in **POA appointment date**, then **Create episode**. **Expected:** "Enter the POA appointment date as a date."; no episode is created, and the page stays.
2. The same with **Planned procedure date** ("Enter the planned procedure date as a date.").
3. On an episode, **Edit procedure and dates**, type a partial planned date and **Save**. **Expected:** the same message; the form stays open and nothing changes.
4. A missing date is fine: both are optional.

### The episode page, list and POA Summary

**T-14: Edit the procedure and dates until the review is complete.**
1. As **Cara**, open the episode from T-09. Click **Edit procedure and dates**.
2. Change the procedure to **Knee arthroscopy**, the Planned date to `2026-11-10`, clear the POA date. **Save**.
- **Expected:** the details show Knee arthroscopy, 10/11/2026 and — for the POA date. **General notes** gain "Procedure set to Knee arthroscopy", "Planned procedure date set to 2026-11-10", "POA appointment date cleared", each by Cara Clinician. **Cancel** on the form changes nothing.
3. Reload. **Expected:** the same.

**T-15: The list and the POA Summary show the procedure and dates.**
1. **Episodes**. **Expected:** columns Procedure, Planned date and POA date, filled for the episodes that have them, — otherwise; dates as dd/mm/yyyy.
2. Open an episode with both dates → **POA Summary**. **Expected:** Procedure, Planned procedure date and POA appointment date in the header details, in the same format.

**T-16: After the HQ review is complete, the procedure and dates can't change.**
1. Take an episode through to a complete review: as the patient, fill in and send the HQ (T-17 shows how), then as Cara validate every Question Set and click **Complete HQ review**.
- **Expected:** the **Edit procedure and dates** button is gone; the procedure and dates still show. The status can still be changed.

### The patient

**T-17: The patient's home shows the procedure's name (phone size 390×844).**
1. Set the browser window (or device emulation) to **390×844**.
2. Open **/qa/messages**, find the newest message to the patient from T-09 (pat.s19@example.test or 07700 900555), and open its invite link.
3. **Continue**, then enter the date of birth (2 / 4 / 1975) and **Continue**. Their questionnaire opens.
4. Go to **Your questionnaires** (the patient's home).
- **Expected:** the questionnaire's card shows "Procedure: Knee arthroscopy" (or whatever the episode has now), and nothing scrolls sideways at 390 px. No dates are shown to the patient.

### Reload, deep links, back

**T-18:** On Procedures, reload: still there. Use the browser's Back from the Procedures page to Episodes and Forward again: each page shows correctly. Switch hospital in the header (Alex, A1 → A2) while on Procedures: you stay on Procedures, with the same trust list.

## Regression (earlier sessions)

- **R-1 (S13):** New episode for an existing patient: search by surname, **Choose**, create with no procedure and no dates. **Expected:** created; the details show — for the procedure and dates.
- **R-2 (S14, S18):** the patient fills in and sends the HQ from their link (as in T-17); the episode becomes **Ready for review**, and Cara can validate each set and complete the review; the POA Summary shows both tabs.
- **R-3 (S10):** as Ada, **Settings** still saves a hospital value (e.g. the reminder days) and shows it.
- **R-4 (S11):** as Ada, **Audit**: the procedure changes from T-02–T-05 appear as "Added a procedure" / "Changed a procedure" by Alex, and episode actions as before.
- **R-5 (S08/S09):** Staff and Settings are still only in the header for Ada; Questionnaires and Episodes for everyone.

## Out of scope (don't report)

- Information content and the PROMs plan: they aren't shown or editable yet (C5, C9).
- Worklists by planned date (S21); admitted and discharged dates and the other statuses (S20).
- Changing the HQ after the patient has started (not possible, as before).
- The audit row for a procedure change not naming the procedure (the audit log records patients and episodes only).
- "E2E …" procedures, HQs and patients left by the automated journeys.
