# S20 manual test plan: every status, cancel and re-open

**Changes after test report 1:** every reason box (status, Undo, Re-open, Cancel's details) is now one line, so **Enter sends the form** (T-03 step 4 as written), and takes at most 1,000 characters. In a stale tab, if after **Reload** the episode no longer offers the move you'd started, the form closes and says "The episode changed, so that change is no longer offered." (T-10). An unknown status sent to the API is 400. The patient's cancelled page has **Sign out** (T-09 step 8).

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

   The **role ladder** is clinician < super clinician < admin.
4. **Seeded data:** each trust has a published HQ, **Demo pre-op HQ** v1, with three Question Sets: *About you* and *Your health* (the patient's) and *Nurse checks* (the clinician's). The database may also hold "E2E …" HQs, patients and episodes from the automated journeys; ignore them.
5. **Captured messages:** texts and emails aren't sent. **http://localhost:3000/qa/messages** lists them, with the patient's invite link and sign-in links.
6. **Dates** show in the deployment's locale, en-GB locally (e.g. 10/01/2026). Date fields are the browser's date picker; you can type into them.

### How to make an episode reach "Ready for POA" (used by many cases)

1. As **Cara Clinician** (Hospital A1): **Episodes** → **New episode** → enter a new patient (first name `Sam`, a surname of your own such as `S20 Walk`, date of birth `1980-05-17`, sex Female, email e.g. `s20.walk@example.test`), Health questionnaire **Demo pre-op HQ v1**, **Create episode**.
2. On the episode page, **Fill in for the patient**: answer every question in About you and Your health, and send it. Back on the episode the status is **Ready for review**.
3. **Validate** each of the three Question Sets (answer Nurse checks' blood pressure, e.g. `120/80`), then **Complete HQ review**. The status is now **Ready for POA**.

## What's new, in short

- The **Episode status** dropdown lists only the moves *you* may make from here. Picking one opens a short form under it (no pop-up) with **Change status** and **Back**; Admitted and Discharged ask for a date.
- Buttons beside it, only when allowed: **Re-open HQ review** (super clinicians and admins), **Undo: back to …** (any clinician; steps back the latest move made by hand) and **Cancel episode** (final).
- A **Status history** card on the episode page: when, the change, by whom, and the reason.
- A move or a save made in a tab that's out of date is refused with "This episode changed since you opened it. Reload to see the latest; your change was not saved.", keeping what you typed, with a **Reload** button.

## Test cases

### The status control and history

**T-01: A new episode offers only Cancel.**
1. As Cara, create an episode (step 1 of the recipe above) and stay on its page.
- **Expected:** the Episode status dropdown shows **HQ not complete** and is disabled, with "The status can be changed by hand once the HQ review is complete."; the only button beside it is **Cancel episode**. The **Status history** card has one row: "Created: HQ not complete", by Cara Clinician, reason —.

**T-02: The patient's submit and the review are in the history.**
1. Do steps 2–3 of the recipe on that episode.
- **Expected:** the status is **Ready for POA**. The history now reads, in order: Created: HQ not complete; **HQ sent: HQ not complete → Ready for review** (by Cara Clinician, since she sent it for the patient; if the patient sends it from their own link, by **Patient**); **HQ review completed: Ready for review → Ready for POA** (Cara). The dropdown now offers Ready for POA's moves: POA complete, On hold, Not ready for admission, Ready for admission (and not Admitted, Discharged, HQ not complete or Ready for review). Cara sees **Cancel episode** but no **Re-open HQ review** and no **Undo**.

**T-03: A pick, with and without a reason.**
1. Pick **On hold** in the dropdown. A form "Change the status to On hold" opens, with **Reason (optional)**.
2. Click **Back**. **Expected:** the form closes; the status is still Ready for POA.
3. Pick **On hold** again, leave the reason empty, **Change status**. **Expected:** status **On hold**; history row "Ready for POA → On hold", Cara, reason —; General notes "Status changed to On hold".
4. Pick **Not ready for admission**, reason `Bloods abnormal`, press **Enter** in the reason box or click **Change status**. **Expected:** status Not ready for admission; the history row has the reason; General notes "Status changed to Not ready for admission: Bloods abnormal".
5. Check the dropdown from Not ready: it offers Ready for POA, POA complete, On hold and Ready for admission.

**T-04: Walk an episode to Discharged, with the dates.**
1. Pick **Ready for admission**, **Change status**. **Expected:** the dropdown now also offers **Admitted**.
2. Pick **Admitted**. The form shows **Admission date** filled with today. Clear it and **Change status**. **Expected:** "Enter the admission date as a date."; nothing changes.
3. Type only part of a date (e.g. day and month, leaving the year) and **Change status**. **Expected:** the same message, nothing sent.
4. Enter a date in the future (e.g. next year). **Expected:** "The admission date can't be in the future."
5. Enter `10/01/2026` (10 January 2026) and **Change status**. **Expected:** status **Admitted**; the details card shows **Admission date 10/01/2026**; the history row "Ready for admission → Admitted"; General notes "Status changed to Admitted (admission date 2026-01-10)". The dropdown now offers only **Discharged**.
6. Pick **Discharged** with `09/01/2026`. **Expected:** "The discharge date can't be before the admission date (2026-01-10)."
7. Discharged with `12/01/2026`. **Expected:** status **Discharged**; Discharge date 12/01/2026 in the details; the dropdown is disabled (nothing offered); no Cancel episode button (a discharged episode can't be cancelled).
8. Reload. **Expected:** the same status, dates and history.

**T-05: Undo, one step at a time (any clinician).**
1. On the discharged episode from T-04, as Cara, click **Undo: back to Admitted**. A form "Undo the last status change: back to Admitted" asks for a **Reason**.
2. Click **Undo** with the reason empty. **Expected:** "Say why the status change is being undone."; nothing changes.
3. Reason `Pressed by mistake`, **Undo**. **Expected:** status **Admitted**; the Discharge date is gone from the details; the history's "Admitted → Discharged" row says "(undone)", and a new row "Undone: Discharged → Admitted", Cara, Pressed by mistake. The button now reads **Undo: back to Ready for admission**.
4. Keep undoing, each time with a reason. **Expected:** Ready for admission (the Admission date is gone), then Not ready for admission, then On hold, then Ready for POA: each a new "Undone" row, and the button names the next step back each time.
5. At Ready for POA, **Expected:** no Undo button (completing the review isn't undone; Re-open is for that).
6. Any clinician may undo anyone's move: as **Sue Super** pick On hold, then sign in as **Cara** and undo it. **Expected:** allowed; back to Ready for POA.

**T-06: Re-open as a super clinician; not as a clinician.**
1. Take an episode to **On hold** (recipe, then pick On hold).
2. As **Cara**: **Expected:** no **Re-open HQ review** button.
3. Sign in as **Sue Super** (Hospital A1), open the episode. Click **Re-open HQ review**. The form explains that the clinician's answers can be changed again and the patient isn't asked again, and asks for a **Reason**.
4. Click **Re-open review** with the reason empty. **Expected:** "Say why the HQ review is being re-opened."
5. Reason `Missed the cardiac history`, **Re-open review**. **Expected:** status **Ready for review**; the "HQ review completed by …" line is gone; **Complete HQ review** is back; each Question Set shows **Review** (still validated, by Cara) and opens with its answers editable. History row "HQ review re-opened: On hold → Ready for review", Sue Super, the reason; General notes "HQ review re-opened: Missed the cardiac history". **Edit procedure and dates** is available again (the review isn't complete).
6. Change an answer in a set, validate it, **Complete HQ review** again. **Expected:** Ready for POA again.
7. The patient's side: open the patient's home or HQ (if you sent it from their own link) — it still shows **Sent**: re-opening doesn't ask the patient again.
8. **Ada Admin** can re-open too (she's above super clinician).

### Cancel

**T-07: Cancel needs a reason; Other needs a sentence.**
1. As Cara, make a new episode (recipe step 1) and click **Cancel episode**. The form "Cancel this episode" says cancelling is final and lists four reasons: Procedure cancelled, Patient withdrew, Moved to another hospital, Other.
2. Click the red **Cancel episode** in the form with nothing chosen. **Expected:** "Choose why the episode is cancelled."
3. Choose **Other**, leave **Say why** empty, submit. **Expected:** "Say why the episode is cancelled."
4. **Back** closes the form; nothing changed.

**T-08: Cancel with each reason.**
1. Make four episodes (any status before Discharged: e.g. one new, one Ready for review, one Ready for POA, one Admitted) and cancel each with a different reason; for **Moved to another hospital** add details `Leeds General`; for **Other** `Moved abroad`.
- **Expected for each:** a grey **This episode is cancelled** box with "Reason: …" (e.g. "Reason: Moved to another hospital: Leeds General", "Reason: Other: Moved abroad", "Reason: Patient withdrew"); the dropdown shows **Cancelled**, disabled; no Cancel, Re-open or Undo buttons; no **Edit procedure and dates**; no patient's invite card; the history's last row "Cancelled: … → Cancelled" with the reason; General notes "Episode cancelled: …".
2. On the cancelled episode, try the POA Summary's ASA grades **Save**. **Expected:** "This episode is cancelled, so it can't be changed."
3. **Add comment** in General notes still works.
4. **Episodes** list: the episodes show **Cancelled**; the Status filter offers Admitted, Discharged and Cancelled too.

**T-09: The patient's side of a cancel (phone size 390×844).**
1. Set the window (or device emulation) to **390×844**.
2. As Cara, create an episode for a new patient with an email of your own (e.g. `s20.cancel@example.test`, date of birth `1980-05-17`).
3. In **/qa/messages**, open that patient's invite link; **Continue**; date of birth 17 / 5 / 1980; **Continue**. Their questionnaire opens. Keep this tab.
4. In another window (desktop size), as Cara, cancel the episode (Patient withdrew).
5. Back in the patient's tab, reload. **Expected:** the patient is signed out ("For your security, you've been signed out…").
6. Open the same invite link again, **Continue**. **Expected:** "This link doesn't work".
7. Go to **/signin**, enter the email, **Send me a link**; open the sign-in link from /qa/messages; Continue; date of birth.
- **Expected:** the patient's home lists the questionnaire with a **Cancelled by the hospital** badge, a sentence that there's nothing to fill in, and **no Start/Continue/View button**. Nothing scrolls sideways.
8. Type `/home/episodes/<the episode's id>` (from the staff page's address). **Expected:** "This questionnaire was cancelled by the hospital…", with **Your questionnaires** and **Sign out** (which signs them out).
9. As Cara on the cancelled episode, open `…/episodes/<id>/patient-form` (the on-behalf form's address). **Expected:** the same cancelled message; nothing to fill in.
10. In **/qa/jobs**, run the jobs as of a week later. **Expected:** no reminder for this patient appears in /qa/messages, and the episode doesn't get "Not responding".

### Stale tabs

**T-10: A move made in a stale tab.**
1. Open a Ready for POA episode in **two tabs** as Cara.
2. In tab 1, pick **On hold** and confirm.
3. In tab 2 (not reloaded), pick **POA complete** and **Change status**.
- **Expected:** "This episode changed since you opened it. Reload to see the latest; your change was not saved.", with **Reload**; the form keeps your choice and reason. **Reload** shows On hold, with the history row from tab 1. Nothing from tab 2 was saved.
4. The same with **Cancel episode** in a stale tab: refused the same way; the episode isn't cancelled.
5. A move that's gone: in tab 1 take the episode to Ready for admission and pick **Admitted** (leave the form open). In tab 2 (reloaded), **Undo** back to Ready for POA. In tab 1 click **Change status**: the stale message; click **Reload**. **Expected:** the form closes, "The episode changed, so that change is no longer offered.", and the status shows Ready for POA.

**T-11: Two tabs: Nurse ASA in one, Anaesthetist ASA in the other.**
1. Open the **POA Summary** of a reviewed episode in two tabs.
2. Tab 1: Nurse ASA grade **ASA II**, **Save**. **Expected:** saved ("Nurse ASA grade set to 2" in General notes).
3. Tab 2 (not reloaded): Anaesthetist ASA grade **ASA III**, **Save**.
- **Expected:** the stale message with **Reload**; ASA III stays in the field. Click **Reload**: Nurse shows **ASA II** (tab 1's), Anaesthetist still **ASA III** (unsaved). **Save** it: both grades are kept (check on a fresh reload of either tab).

### Refused paths

**T-12: Moves the table doesn't allow aren't offered.**
- At **HQ not complete** and **Ready for review**: only Cancel. At the five reviewed statuses: the other four of them (plus Admitted from Ready for admission). At **Admitted**: Discharged and Cancel. At **Discharged** and **Cancelled**: nothing.

**T-13: Another hospital's or trust's staff can't change the status.**
1. As **Val Clinician** (Hospital A2), paste the address of a Hospital A1 episode. **Expected:** not found; no status control.
2. As **Bea Author** (Trust B), the same address. **Expected:** not found.

**T-14: Undo by a clinician, re-open only by a super clinician.** (Covered in T-05 and T-06; re-check that Cara, a clinician, sees **Undo** after her own pick, and that **Sue Super** sees Re-open but on an Admitted episode does **not**: re-open is only from the five reviewed statuses.)

### Reload, deep links, back

**T-15:** On an episode with a few moves, reload: status, history and dates stay. Go to the POA Summary: the same status control works there (pick, undo, cancel), and the episode page's history shows what you did. Browser **Back** from the POA Summary to the episode page shows the current status.

**T-16:** Audit: as **Ada**, **Audit**: the moves appear as "Changed an episode's status", "Cancelled an episode", "Re-opened the HQ review", "Undid a status change", by the person who made them.

## Regression (earlier sessions)

- **R-1 (S14, S18):** the patient fills in and sends the HQ from their own link; the episode becomes **Ready for review**, and Cara can validate each set and complete the review; the POA Summary shows both tabs.
- **R-2 (S16):** a new episode's invite and the Resend button still work (before cancelling); **/qa/jobs** still sends a reminder for an episode that isn't cancelled.
- **R-3 (S19):** **Edit procedure and dates** before the review saves, with its General notes lines.
- **R-4 (S15):** a returning patient's home lists every trust's questionnaires with To do / In progress / Sent.
- **R-5 (S11):** the episode actions (open, change, note) still appear in Audit.

## Out of scope (don't report)

- Archive and un-archive (S22); tasks, worklists and status messages to the patient (S21); auto-triage (C6); Follow-up (PROMs) / recovery (C9).
- Episodes made before this session have no status history (only moves made from now on are recorded).
- The cancel link message to the patient is the general "This link doesn't work" (no cancel-specific wording on the link page).
- "E2E …" HQs, patients and episodes left by the automated journeys.
