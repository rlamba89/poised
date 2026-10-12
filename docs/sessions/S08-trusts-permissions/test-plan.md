# S08 test plan: trusts, hospitals and permissions

For a tester who uses only the browser. You don't need to read any code.

**Changes for retest 3 (after report 3, 12 Oct):**
- **F-1 fixed at the root:** outputs saved without codes or a note category (like the "E2E …" questionnaires') now work everywhere: the Outputs panel, **Save answers**, **Run all**, the Publish check and the POA Summary. T-25 step 4 now uses an "E2E …" draft on purpose, and new case **T-28** covers the rest.
- **The editor's "Delete this?" dialog** keeps its text while it fades out (T-29).

**Changes for retest 2 (after report 2, 12 Oct):**
- **F-1 fixed:** for anyone who can't change the questionnaire (clinicians, Sue Super), the Preview's **Test cases** panel has no name box, **Save answers** or delete icons, and says "Read only: …". The header button reads **Open in editor** for them. New case **T-25**.
- **Narrow windows:** the staff header scrolls sideways inside itself instead of pushing Episodes and Sign out off screen. New case **T-26**.
- **Publish dialog:** cancelling a blocked Publish no longer flashes the "ready" text. **Preview:** an output without codes no longer crashes it (T-27).

**Changes for the retest (after report 1, 12 Oct):**
- **New rule (Rahul's answer):** only super clinicians and admins of the **whole trust** can change the trust's questionnaires (and its option lists). A super clinician or admin of one hospital only can see them but not change them. New user **Sue Super** (super clinician, Hospital A1 only) and new cases **T-20 to T-22**.
- **Switching hospital from the header now keeps you in the same section** (Episodes stays Episodes). T-03 step 5 and T-06 changed; new case T-23.
- Report 1's cosmetic items: a questionnaire's **View** menu item now has an eye icon (pencil only for **Edit**), and the delete dialog keeps the name while it fades out (T-24).

## What changed in S08

- Staff now belong to **trusts**. A trust has one or more **hospitals**. Someone can be a member of a **whole trust** (every hospital in it) or of **one hospital** only.
- There are three roles, one above the other: **Clinician < Super clinician < Admin**. Each can do everything the one before it can.
  - **Clinician:** patients and episodes, and validating HQs. Can *see* questionnaires, but not change them.
  - **Super clinician:** also creates, edits and publishes questionnaires.
  - **Admin:** also deletes other people's draft questionnaires. (Staff screens come in S09.)
- Everyone in a hospital sees all of that hospital's data. Roles only limit what they can *do*.
- **Changing the trust's questionnaires and option lists needs a whole-trust role:** a super clinician or admin of the whole trust. A super clinician of one hospital only can view them, not change them.
- Questionnaires belong to the trust, so every hospital of a trust sees the same questionnaires.
- Patient records belong to the trust, so a clinician at one hospital can pick a patient first seen at another hospital of the trust. **Episodes belong to one hospital**, and a clinician at Hospital A1 never sees Hospital A2's episodes.
- Every staff page's address now starts `/o/<trust id>/h/<hospital id>/…`. Old addresses (`/h/<id>/…`) are gone.

## Setup

**Rahul runs, in his own terminal** (an automatic run does this for you):

```sh
docker compose up -d db
make migrate seed
make dev
```

Leave `make dev` running. Open **http://localhost:3000/login**.

**Signing in:** the login page lists every seeded user; click a name to sign in. To switch user, click **Sign out** (top right), or open http://localhost:3000/login again.

**The seeded users:**

| User | Member of | Role |
| --- | --- | --- |
| Ada Admin | Trust A, whole trust (Hospitals A1 and A2) | Admin |
| Alex Author | Trust A, whole trust | Super clinician |
| Cara Clinician | Trust A, **Hospital A1 only** | Clinician |
| Sue Super | Trust A, **Hospital A1 only** | Super clinician (can't change the trust's questionnaires) |
| Val Clinician | Trust A, **Hospital A2 only** | Clinician |
| Bea Author | Trust B, whole trust (Hospital B1) | Super clinician |
| Max Multi | Trust A, **Hospital A2 only** (Clinician) **and** Trust B, whole trust (Admin) | differs per trust |
| Sam Superadmin | no trust (a platform admin) | — |

**Data you'll need:**
- **A published HQ in Trust A.** The questionnaire list usually has some (State: Published, e.g. "E2E …" ones from the automated tests). If there are none, make one in T-11 first.
- **A patient:** the seed adds three made-up Trust A patients (Jo Bloggs, Pat Example, Robin Sample). Create others with made-up names only.

**Reading ids from the address bar:** several cases ask you to copy the page's address. The part after `/o/` is the trust's id and the part after `/h/` is the hospital's.

## Test cases

### Signing in and the hospital picker

**T-01 · One hospital: straight in.**
1. Sign in as **Cara Clinician**.
2. **Expected:** no picker. You land on **Episodes** at `/o/…/h/…/episodes`. The header shows **Poised**, **Trust A**, a hospital box showing **Hospital A1**, the **Questionnaires** and **Episodes** buttons, "Cara Clinician" and a **Clinician** badge.

**T-02 · Several hospitals: the picker.**
1. Sign in as **Alex Author**.
2. **Expected:** a **Choose a hospital** page with one card, **Trust A**, marked "Super clinician, whole trust", listing **Hospital A1** and **Hospital A2**, each showing "Super clinician".
3. Click **Hospital A1**.
4. **Expected:** **Questionnaires** (super clinicians and admins start there), header **Trust A**, **Hospital A1**, badge **Super clinician**.

**T-03 · The two-trust user (Max Multi).**
1. Sign in as **Max Multi**.
2. **Expected:** the picker shows two cards:
   - **Trust A** with only **Hospital A2** ("Clinician"), and no "whole trust" badge;
   - **Trust B** marked "Admin, whole trust", with **Hospital B1** ("Admin").
   - Hospital A1 is **not** listed.
3. Click **Hospital B1**. **Expected:** Questionnaires; header **Trust B**, badge **Admin**.
4. In the header's hospital box, open the list. **Expected:** hospitals grouped under **Trust A** (Hospital A2) and **Trust B** (Hospital B1).
5. Click **Episodes** in the header, then choose **Hospital A2** in the hospital box. **Expected:** you land on Hospital A2's **Episodes** (switching keeps the section); header **Trust A**, badge **Clinician**; no "All hospitals" switch.

**T-04 · The platform admin.**
1. Sign in as **Sam Superadmin**.
2. **Expected:** the page says **You don't have access to any hospital yet** and "You're a platform admin…". No hospital pages. **Sign out** works.

**T-05 · Ada Admin's picker.**
1. Sign in as **Ada Admin**.
2. **Expected:** Trust A, "Admin, whole trust", with Hospital A1 and Hospital A2, each "Admin".

### Switching hospitals and trusts

**T-06 · Switch from the header, then Back.**
1. As **Alex Author**, open Hospital A1's **Questionnaires**.
2. In the header's hospital box, choose **Hospital A2**.
3. **Expected:** the address's `/h/…` part changes, you're still on **Questionnaires**, the header shows Hospital A2, and the same Trust A questionnaires are listed (they belong to the trust).
4. Press the browser's **Back** button. **Expected:** Hospital A1 again, the header showing Hospital A1.
5. Reload. **Expected:** the same page and hospital.

**T-07 · Back to the picker.**
1. Signed in as **Max Multi** on any hospital page, go to http://localhost:3000/.
2. **Expected:** the picker again, with both trusts.

### Episodes: one hospital, and "All hospitals"

**T-08 · Each hospital sees only its own episodes.**
1. Sign in as **Cara Clinician** (Hospital A1). Click **New episode**, pick an existing patient (e.g. Jo Bloggs) and a published HQ, and create it. Note the patient's name.
2. Sign out. Sign in as **Val Clinician** (Hospital A2). Create an episode for **Pat Example** with a published HQ.
3. **Expected:** Val's Episodes list shows Pat Example's new episode and **not** the one Cara made at A1.
4. Sign in as Cara again. **Expected:** her list shows her A1 episode and **not** Val's A2 episode.
5. **Expected for both:** there's **no "All hospitals"** switch (they're members of one hospital only).

**T-09 · "All hospitals" for trust-wide staff.**
1. Sign in as **Ada Admin**, choose **Hospital A1**, click **Episodes**.
2. **Expected:** a switch above the list: **Hospital A1 | All hospitals**, with Hospital A1 selected. Only A1's episodes are listed (Val's A2 episode from T-08 isn't).
3. Click **All hospitals**.
4. **Expected:** the address ends in `?hospitals=all`; a **Hospital** column appears; both Cara's (Hospital A1) and Val's (Hospital A2) episodes are listed.
5. Reload. **Expected:** still All hospitals.
6. Click Val's A2 episode. **Expected:** the episode opens, and the header's hospital box now shows **Hospital A2** (it opened at its own hospital).
7. Back. **Expected:** the All hospitals list again.
8. Use the **Status** filter with All hospitals on. **Expected:** it filters across both hospitals.

**T-10 · A patient record is the trust's.**
1. As **Cara Clinician** (A1), click **New episode**, keep **Existing patient**, and open the patient list.
2. **Expected:** it includes the patient Val used at A2 in T-08 (Pat Example), so Cara can open an episode for them at A1.
3. Create it. **Expected:** it appears in Cara's list. Val's A2 episode for the same patient still doesn't.

### The role ladder

**T-11 · A super clinician authors and publishes** (also makes the published HQ other cases need).
1. Sign in as **Alex Author**, choose Hospital A1, Questionnaires → **New Questionnaire**. Name it "S08 test <your initials> <time>", any description, **Save**.
2. In the editor: **Add Question Set**, **Add Page**, then add a **Yes/No** (or Text) question and give it text.
3. Click **Publish** in the header and confirm.
4. **Expected:** it publishes ("v1 · published", Read only).

**T-12 · A clinician can't author or publish.** (Also check the **…** menu: it says **View** with an eye icon.)
1. Sign in as **Cara Clinician**, click **Questionnaires**.
2. **Expected:** the list shows Trust A's questionnaires (everyone sees the hospital's data), but there's **no New Questionnaire** button.
3. Open a **draft** questionnaire (State: Draft).
4. **Expected:** the editor is read-only: no Publish button, nothing editable.

**T-13 · Deleting someone else's draft: super clinician no, admin yes.**
1. As **Alex Author**, create a draft questionnaire "S08 delete me <time>" and go back to the list.
2. Sign in as **Max Multi**, choose **Hospital B1** (Trust B, Admin). **Expected:** Alex's Trust A questionnaire is **not** in Trust B's list.
3. Sign in as **Ada Admin** (Trust A admin), Hospital A1, Questionnaires. Open the **…** menu on "S08 delete me". **Expected:** **Delete** is offered; deleting it removes it.
4. (Super clinician check) As **Alex Author**, his own drafts can be deleted; for a draft created by Ada, Alex's **…** menu has no Delete.

**T-14 · The same person, different roles in different trusts.**
1. Sign in as **Max Multi**, choose **Hospital B1** (Admin). **Expected:** **New Questionnaire** is there.
2. Switch to **Hospital A2** (Clinician), click Questionnaires. **Expected:** no New Questionnaire; drafts open read-only.

### Leaks: by URL

**T-15 · A hospital-scoped clinician opens another hospital's episode by its URL.**
1. Sign in as **Ada Admin**, open the **All hospitals** list (T-09), click an **Hospital A2** episode, and copy the address.
2. Sign out, sign in as **Cara Clinician** (A1 only), paste the address and go.
3. **Expected:** **You don't have access to this hospital**, with a **Choose a hospital** link. No patient name or episode detail appears.
4. Change only the `/h/<A2 id>` part of that address to Cara's own hospital id (copy it from her own Episodes address). **Expected:** "Episode not found." (A2's episode isn't at A1.)

**T-16 · Trust A's admin opens a Trust B URL.**
1. Sign in as **Bea Author**, open Questionnaires, and copy the address (Trust B, Hospital B1).
2. Sign out, sign in as **Ada Admin**, paste it.
3. **Expected:** **You don't have access to this hospital**; nothing of Trust B shows. The header's hospital box is empty ("Choose a hospital").
4. Change the address so the `/o/…` part is **Trust A's** id (from Ada's own address) but keep **Hospital B1's** `/h/…` id. **Expected:** the same refusal.

**T-17 · The API refuses too.** Signed in as **Ada Admin**, type these into the address bar (use the ids copied in T-16):
1. `http://localhost:3000/api/o/<Trust B id>/h/<Hospital B1 id>/questionnaires` → **Expected:** `{"error":"Hospital not found."}` (a 404).
2. `http://localhost:3000/api/o/<Trust B id>/episodes` → **Expected:** `{"error":"Trust not found."}`.
3. `http://localhost:3000/api/o/<Trust A id>/episodes` → **Expected:** a list of every Trust A episode, each with a `hospitalName`.
4. `http://localhost:3000/api/me` → **Expected:** Ada's `trusts`: one trust (Trust A, `"role":"admin"`) with two hospitals, each with `"role":"admin"` and a `timeZone`.

**T-18 · Old and broken addresses.**
1. Signed in as anyone, open `http://localhost:3000/h/00000000-0000-4000-8000-00000000000a/episodes` (the old style).
2. **Expected:** the app's **Not found** page.
3. Open `http://localhost:3000/o/nope/h/nope/episodes`. **Expected:** "You don't have access to this hospital".

### Signed out

**T-19 · Deep link while signed out.**
1. Copy any hospital page's address, Sign out, then open the address.
2. **Expected:** you're sent to the login page.

### Whole-trust rule for the trust's questionnaires (Rahul's answer, 12 Oct)

**T-20 · A super clinician of one hospital can view but not change the trust's questionnaires.**
1. As **Alex Author** (whole trust), create a draft questionnaire "S08 trust-wide <time>" with one Question Set.
2. Sign in as **Sue Super**. **Expected:** straight into Hospital A1 (her only hospital), on **Questionnaires**, badge **Super clinician**.
3. **Expected:** the list shows Trust A's questionnaires, including Alex's new draft, but there's **no New Questionnaire** button.
4. Open the **…** menu on Alex's draft. **Expected:** **View** (eye icon), and no **Delete**.
5. Open the draft. **Expected:** **Read only** in the header; no **Publish** button; nothing can be edited (no Add Question Set, the question cards don't open for editing).
6. Open a **published** questionnaire. **Expected:** no **Create new version** button.

**T-21 · The API refuses Sue's changes too.** Signed in as **Sue Super**, open the browser's developer console on any Poised page and run (replace the ids with Sue's own `/o/…/h/…` and a draft's id from its address):
```js
await fetch("/api/o/<trust>/h/<hospital>/questionnaires", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({name: "x", description: "x"})}).then(r => r.status + " " + r.statusText)
await fetch("/api/o/<trust>/h/<hospital>/questionnaires/<draft id>/publish", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({versionId: "<any>"})}).then(r => r.json())
```
**Expected:** `403 Forbidden`, and `{"error": "Only super clinicians and admins of the whole trust can change the trust's questionnaires."}`. Nothing is created (the list is unchanged after a reload).

**T-22 · Whole-trust staff still can.**
1. As **Alex Author** (whole trust, Hospital A1), open his draft from T-20: it's editable, and **Publish** works.
2. As **Ada Admin** (whole trust), **Delete** is offered on someone else's draft (T-13 step 3).

**T-23 · Switching hospital keeps the section.**
1. As **Ada Admin** at Hospital A1, open **Episodes**, then choose **Hospital A2** in the header. **Expected:** Hospital A2's Episodes.
2. Open **Questionnaires**, switch back to **Hospital A1**. **Expected:** Hospital A1's Questionnaires.
3. Open an episode's **POA Summary** and switch hospital. **Expected:** the other hospital's **Episodes** list.

**T-24 · The delete dialog keeps the name while closing.** As **Alex Author**, create a draft "S08 fade <time>", choose **Delete** from its **…** menu, then **Delete**. **Expected:** while the dialog fades out it still shows "S08 fade <time>" (never an empty `""`), and the questionnaire is gone.

**T-25 · The Preview is read-only for those who can't change the questionnaire (F-1).**
1. As **Sue Super**, open a Trust A **draft** questionnaire, a Question Set, then **Preview**.
2. **Expected:** the Test cases panel says "Read only: only super clinicians and admins of the whole trust can change test cases."; there's no name box, no **Save answers**, and no delete icon on existing test cases. **Load** and **Run all** still work. The header button reads **Open in editor**.
3. Repeat as **Cara Clinician**. **Expected:** the same.
4. As **Alex Author** on the same draft (an "E2E trust-wide …" one is fine): the name box and **Save answers** are there; choose **Yes** for "Do you smoke?", name it, and **Save answers**. **Expected:** it's listed, and **Run all** says "All 1 passed". On a **published** version, Alex sees "Read only: test cases can be changed only in a draft version."

**T-26 · The staff header in a narrow window.**
1. As **Cara Clinician**, make the window about 390–700px wide (or use a phone-sized frame) on Episodes.
2. **Expected:** the header doesn't widen the page; it scrolls sideways inside itself, so **Questionnaires**, **Episodes** and **Sign out** can all be reached and clicked. (The Episodes table itself may still be wider than a phone: out of scope.)

**T-27 · Small fixes from report 2.**
1. As **Alex Author**, open a draft with a logic problem (e.g. from R-02), click **Publish**, then **Cancel**. **Expected:** while the dialog fades out it keeps showing the problem; it never switches to "Patients can then be given this version…".
2. Open the Preview of an "E2E trust-wide …" or "E2E trusts …" questionnaire's first Question Set and choose **Yes** for "Do you smoke?". **Expected:** no "Something went wrong"; Clinical outputs shows the note "Smoker".

**T-28 · Outputs without codes or a category, everywhere (F-1 from report 3).** Use an "E2E trust-wide …" or "E2E test case …" **draft** (its disclosures are notes with no category and no codes).
1. As **Alex Author**, in its Preview, choose **Yes** for "Do you smoke?". **Expected:** Clinical outputs shows "Smoker"; no crash.
2. Save a test case and **Run all**. **Expected:** saved, "All 1 passed".
3. In the editor, open the **Disclosures** of the "Yes" option. **Expected:** the dialog lists "Smoker" with no error.
4. Click **Publish**. **Expected:** the dialog opens normally (no crash), listing any real problems.
5. As **Cara Clinician**, open an episode named "Sam J5 …" (made by the automated tests on an "E2E J5 …" HQ, where the patient smokes) and its **POA Summary**. **Expected:** "Smoker" appears, with no error.

**T-29 · The "Delete this?" dialog keeps its text.** As **Alex Author**, in a draft, select a question card, click its bin icon, then **Cancel**. **Expected:** while the dialog fades out it still shows its message (never just "Cancel Delete"). Repeat with **Delete**: the question is deleted.

## Regression (earlier sessions, at the new addresses)

**R-01 · J1 Author:** as **Alex Author** at Hospital A1, create a questionnaire, add a Question Set, page and question; reload: it's kept; **Preview** shows it.

**R-02 · J2 Publish:** publishing a draft with a broken condition is blocked with the problem listed; fixing it lets it publish; **Create new version** gives v2 draft.

**R-03 · J3 Patient, at phone size (390×844):** as **Cara Clinician** create an episode, **Copy** the patient link, open it in a phone-sized window (390×844). **Expected:** "Hello <first name>", only the patient's Question Sets, no sideways scrolling; fill them in and **Send my answers**; "Thank you". Back as Cara, reload: **Ready for review**.

**R-04 · J4 Validate:** as Cara, open that episode, **Validate** each Question Set (change one answer: "Patient answered: …" appears), then **Complete HQ review** → Ready for POA.

**R-05 · J5 POA Summary:** open the POA Summary from the episode: both tabs show; change the status; add a comment; it appears in General notes.

**R-06 · Dates:** dates still show in the deployment's format (e.g. 17 May 1980; 12/10/2026, 22:44).

## Out of scope (don't report)

- Invites, the Staff screen, changing roles or removing people (S09).
- Platform admin screens: creating trusts or hospitals, suspending (S10). Sam Superadmin has nothing to do yet.
- The audit log (S11). Real sign-in (Cognito, S12): the dev login stays.
- Patient search, NHS numbers and duplicate warnings (S13). The New episode drawer still lists all of the trust's patients, and the **hospital number** still sits on the patient record (S13 moves it to the episode).
- Times are shown in the browser's time zone, not the hospital's.
- Questionnaires are shared by every hospital of a trust; there's no way yet to make one belong to a single hospital.
- The questionnaire **editor** has its own header without the trust and hospital names (it was like that before).
- Staff pages other than the header at phone width (logged in pending.md).
- The automated tests' "E2E …" questionnaires and "Sam …" patients cluttering the lists.
