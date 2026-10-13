# S10 manual test plan: the superadmin console, trusts, hospitals and settings

For a tester who hasn't seen the code. Use only the browser.

**Changed after verification 1 (for the retest):** new cases **T-22** (a platform admin who is also staff of a suspended trust still signs in and can re-activate it) and **T-23** (a trust whose hospitals are all suspended is left out for its whole-trust staff). Only these two, and a quick T-12/T-13 regression, need running.

**Changed after test report 1:**
- `make seed` now also resets the demo trusts' and hospitals' **settings** (and time zones) to Poised's defaults, so each round starts clean.
- **T-05 step 7** adds the duplicate name to QA Trust Two, not Trust B (a second hospital in Trust B would change Bea's and Max's later cases).
- **T-08 step 2** uses Zoe Clinician One's email, not Cara's, so Cara stays a Hospital A1 clinician for T-13.
- **T-07** now expects the heading "Your trusts" (not "Choose a hospital") when the trust has no hospitals.
- **T-14, T-15**: the green message now appears beside what you changed (the Save button, or the opt-ins table). **T-15 step 3**: the trust-only settings are shown at a hospital.
- **T-17**: the questionnaire list can be searched. **T-18 step 4** is new (switching hospital from a refused Settings page).
- **New: T-20 and T-21**, editing a trust's and a hospital's details (Rahul, 13 Oct).

## Setup

**Rahul, in your own terminal** (the automatic run does this itself):

```sh
docker compose up -d db
make migrate seed
make dev        # leave it running
```

`make seed` puts the demo trusts and hospitals back to **active**, with their names, codes and Poised's default settings, each time it runs.

**Open:** http://localhost:3000. It goes to the development sign-in page: click a person to sign in as them. **Sign out** is in the page header.

**A second person at the same time:** open **http://tester.localhost:3000** in another tab. It's the same app but keeps its own sign-in, so you can be two people at once.

**Seeded people:**

| Person | What they are |
| --- | --- |
| **Sam Superadmin** | Poised's platform admin (superadmin). No trust, no hospital. |
| Ada Admin | Admin of the whole of Trust A (Hospitals A1 and A2) |
| Alex Author | Super clinician of the whole of Trust A |
| Sue Super | Super clinician of Hospital A1 only |
| Cara Clinician | Clinician of Hospital A1 only |
| Val Clinician | Clinician of Hospital A2 only |
| Bea Author | Super clinician of the whole of Trust B (Hospital B1) |
| Max Multi | Clinician at Hospital A2, and admin of the whole of Trust B |

**Good to know:**
- Emails are never sent (local phase). Someone who is invited appears in the sign-in list at once; sign in as them by clicking their name.
- People you invite need **unique emails**, e.g. `zoe.t06@test.example`. Give them names starting "Zoe" so they're easy to find at the bottom of the sign-in list.
- Trust **codes** must be unique too: use e.g. `QA1`, `QA2`… (2 to 12 letters, digits or hyphens).
- The Claude in Chrome network log shows a `204` response as `503`; ignore that.

## Test cases

### The console

**T-01: The platform admin starts in the console**
1. Sign in as **Sam Superadmin**.
2. Expected: you land on `/admin`, headed "Trusts", with a purple **Platform admin** badge in the header.
3. The table lists **Trust A** (code TA, 2 hospitals, Active) and **Trust B** (TB, 1 hospital, Active), plus any trusts earlier test runs made (named "E2E Trust …").
4. Go to http://localhost:3000/ (the start page). Expected: a card "You're one of Poised's platform admins." with **Open the platform admin console**, which goes back to `/admin`.

**T-02: Create a trust**
1. As Sam, click **New trust**.
2. Fill in: Trust name `QA Trust One`, Code `qa1` (lower case), Logo web address `https://www.example.org/logo.png`, Contact name `Pat Contact`, Contact email `pat@qa.example`, Contact phone `0100 000 0000`, Admin's email a fresh one (e.g. `zoe.admin1@test.example`), Admin's name `Zoe Admin One`.
3. Under **Features and opt-ins**, note Assessments, Files and Observations start ticked and the others not. Tick **PROMs**.
4. Click **Create trust**.
5. Expected: the trust's page opens with a green message "QA Trust One was created. An invitation was sent to zoe.admin1@test.example. Next, add its hospitals." The heading shows the name and **Active**; under it "Code **QA1**" (capitals).
6. "Contact at the trust" shows Pat's name, email and phone, and the logo address (the image itself may not load; that's fine).
7. "Admins of the whole trust" lists Zoe Admin One with **Invited · expires** (a date 7 days from now).
8. "Hospitals" says "No hospitals yet…".
9. "Features and opt-ins": the **Trust default** switch for PROMs is On; Assessments, Files, Observations On; the rest Off.
10. Back to **Trusts** (header): QA Trust One is listed with 0 hospitals.

**T-03: New trust: invalid input is refused**
On **New trust**, try each of these in turn (fill everything else correctly), clicking **Create trust** each time. Expected: a red message under the form saying what's wrong, and you stay on the form:
1. Trust name left empty → "Enter the trust's name."
2. Code `A` → "Enter a short code of 2 to 12 letters…". Also try `A B` and `ABCDEFGHIJKLM`.
3. Contact name empty → asks for it.
4. Contact email `nope` → "Contact: Enter a valid email address…".
5. Contact phone `call me` → "Contact: enter a phone number…".
6. Logo web address `javascript:alert(1)` → "Logo web address: enter a web address starting with https://…".
7. Admin's email empty, then `ann@` → "First admin: …".
8. Expected afterwards: none of these trusts appears in the Trusts list.

**T-04: A duplicate code is refused**
1. New trust with code `TA` (Trust A's), everything else valid. Expected: "Another trust already has the code TA. Choose another."
2. Try `ta` and `qa1` (from T-02). Expected: refused the same way (case doesn't matter).

**T-05: Add hospitals**
1. On QA Trust One's page, click **Add hospital**. Fill in Hospital name `QA General`, Address `1 Test Road` (new line) `Testville`, Phone for patients `0100 111 1111`, leave email empty, leave **Time zone** at "Default (…)". Click **Add hospital**.
2. Expected: "QA General was added to QA Trust One." The Hospitals table shows it with the address on two lines, the phone, "Trust's email", the deployment's time zone with "(default)", and **Active**.
3. Add another: `QA North`, any address, and pick a different **Time zone** by typing in the box (e.g. `America/Toronto`). Expected: its row shows that zone without "(default)".
4. Add a hospital called `qa general` (same name, other case). Expected: "This trust already has a hospital called qa general." in the dialog, which stays open.
5. In the dialog, try Hospital name empty, Address empty, Phone `ring ring`, Email `poa@`, Logo `ftp://x.example/a.png`. Expected: a red message each time, nothing added.
6. Expected: the opt-ins table now has a column for each hospital, each "Follow trust (…)".
7. Another trust may also have a hospital called QA General: open **QA Trust Two** (create it first if you're doing T-07 later: any trust except Trust A and Trust B), add `QA General`. Expected: accepted (names are unique per trust only).

**T-06: Onboard end to end (manual test focus 1)**
1. In the second tab (tester.localhost), open the sign-in page and click **Zoe Admin One** (from T-02).
2. Expected: since she has two hospitals, the hospital picker shows QA Trust One with QA General and QA North and her role Admin, whole trust. Click **QA General**.
3. Expected: the header shows "QA Trust One", hospital QA General, role **Admin**, and links Questionnaires, Episodes, **Staff**, **Settings**. No "Platform admin" link.
4. Click **Staff** → **Invite**. Email a fresh one (e.g. `zoe.clin1@test.example`), Name `Zoe Clinician One`, Access **QA General**, Role **Clinician** → **Send invite**. Expected: "Invitation sent to …".
5. Sign out (second tab) and sign in as **Zoe Clinician One**. Expected: straight into QA General's **Episodes**, role Clinician; no Staff or Settings links; the hospital box offers only QA General.
6. In Sam's tab, reload QA Trust One's page. Expected: Zoe Admin One is now **Active**.

**T-07: A new trust's admin before it has any hospital**
1. As Sam, create `QA Trust Two` (code `QA2`) with a fresh admin email, name `Zoe Admin Two`. Don't add a hospital.
2. In the second tab, sign in as **Zoe Admin Two**. Expected: the start page is headed **Your trusts** (there's no hospital to choose yet) and lists "QA Trust Two" (Admin, whole trust) with the note "QA Trust Two has no hospitals in Poised yet. Poised's platform admin adds them…". Not "You don't have access to any hospital yet".
3. As Sam, add a hospital `QA Two General` to QA Trust Two.
4. Second tab: reload. Expected: she goes straight into QA Two General.

**T-08: Invite another admin from the console**
1. On QA Trust Two's page, **Invite an admin** → a fresh email, name `Zoe Admin Three` → **Send invite**. Expected: "An invitation was sent to …", and she's listed as Invited.
2. **Invite an admin** again with **Zoe Clinician One**'s email (from T-06). Expected: "… already had an account, so they were made an admin of QA Trust Two." She's listed Active. (Sign in as her: the start page now has QA Trust One's QA General and QA Trust Two.) Don't use a seeded person's email: later cases rely on their access.
3. **Invite an admin** with Zoe Admin Two's email. Expected: a red message in the dialog: "Zoe Admin Two already has access to the whole trust. The trust's admins can change their role on the Staff screen."; the dialog stays open.
4. Email `nope` → "Enter a valid email address…".

### Who can reach it (manual test focus 2 and 6)

**T-09: Nobody else reaches the admin screens**
1. Sign in as **Ada Admin**. Expected: no "Platform admin" link in the header.
2. Type http://localhost:3000/admin in the address bar. Expected: "Only Poised's platform admins can open this page", with a link back to your hospitals; no trust list.
3. Also try `/admin/trusts/new` and `/admin/trusts/<QA Trust One's id from Sam's address bar>`. Expected: the same refusal, no trust details.
4. Repeat 2 as **Cara Clinician**. Same refusal.

**T-10: The admin API refuses them too**
1. Signed in as Ada, open http://localhost:3000/api/admin/orgs in a tab. Expected: `{"error":"Only Poised's platform admins can do this."}` (403).
2. Sign out, then open the same address. Expected: `{"error":"Please sign in."}` (401).

**T-11: The platform admin sees no patient data (OPS-06)**
1. As Sam, look through Trust A's page (it has patients and episodes from the demo data). Expected: only trust, contact, hospital, admin and opt-in details; no patient names, dates of birth or episodes anywhere.
2. Find Trust A's id and Hospital A1's id: sign in as Ada in the other tab and copy the address of Episodes, e.g. `/o/<trust>/h/<hospital>/episodes`.
3. As Sam, open that address. Expected: "You don't have access to this hospital". Also open `/api/o/<trust>/episodes` and `/api/o/<trust>/h/<hospital>/patients`. Expected: `{"error":"Trust not found."}` / `{"error":"Hospital not found."}`.

### Suspend and re-activate (manual test focus 3)

**T-12: Suspend a trust**
1. Second tab: sign in as **Bea Author** (Trust B) and open Episodes. Leave it open.
2. As Sam, open **Trust B** → **Suspend trust**. The dialog explains staff can't sign in and patients can't open links, and nothing is deleted. Confirm **Suspend trust**.
3. Expected: "Trust B is suspended…", a **Suspended** badge, a red "This trust is suspended" box, and the button is now **Re-activate trust**. The Trusts list shows Trust B **Suspended**.
4. Second tab (Bea): click Episodes or Questionnaires. Expected: a "not found" / "You don't have access to this hospital" message: her session no longer reaches it.
5. Sign Bea out and try to sign in as **Bea Author** again. Expected: a red message "Your trust or hospital is suspended in Poised at the moment, so you can't sign in. Please contact your trust." You stay on the sign-in page.
6. Sign in as **Max Multi** (clinician at Hospital A2 and Trust B's admin). Expected: sign-in works; he goes straight into Hospital A2 (Trust A); the hospital box has no Trust B.
7. As Sam, click **Re-activate trust**. Expected: "Trust B is active again…".
8. Sign in as Bea. Expected: works, Trust B's hospital as before. Max now sees both trusts again in the hospital box.

**T-13: Suspend one hospital**
1. As Sam, on **Trust A**, click **Suspend** on Hospital A1's row and confirm.
2. Expected: Hospital A1 shows **Suspended** and a **Re-activate** button; Trust A stays Active.
3. Try to sign in as **Cara Clinician** (A1 only; if an earlier case gave her other access, use **Sue Super**, also A1 only). Expected: the "suspended" refusal.
4. Sign in as **Ada Admin**. Expected: she signs in and goes straight into **Hospital A2** (the only one she can reach now); the hospital box has no A1. Episodes → **All hospitals**: no Hospital A1 episodes are listed.
5. Open Hospital A1's old address (`/o/<trust>/h/<A1>/episodes`). Expected: "You don't have access to this hospital".
6. As Sam, **Re-activate** Hospital A1. Cara signs in again; Ada sees both hospitals.

### Opt-ins (manual test focus 4)

**T-14: Trust off with a hospital on, and the reverse**
1. As Sam on **Trust A**: make sure **PROMs** "Trust default" is **Off**. Set **Hospital A1**'s PROMs to **On**. Expected: a green message "PROMs at Hospital A1 is on." just above the opt-ins table (visible without scrolling), and A2 still says "Follow trust (Off)".
2. Turn **EQ-5D-5L** Trust default **On**. Expected: both hospitals' selects say "Follow trust (On)". Set **Hospital A2**'s EQ-5D-5L to **Off**.
3. Reload. Expected: everything as you left it.
4. Check what each hospital ends up with: sign in as **Ada** → **Settings** → "Settings for" **Hospital A1**. Expected: PROMs **On**, EQ-5D-5L **On**. Switch to **Hospital A2**: PROMs **Off**, EQ-5D-5L **Off**. Each says "Set by Poised's platform admin" (Ada can't change them).
5. As Sam, set A1's PROMs back to **Follow trust (Off)** and turn EQ-5D-5L's trust default Off and A2 back to Follow. Expected: all follow the trust again.

### Trust settings

**T-15: The trust's settings and one hospital's own**
1. Sign in as **Ada** → header **Settings**. Expected: "Settings for" = "The whole trust…", with groups Whole trust, Features and opt-ins, What patients see, Reminders and archiving, Patient messages. Every field shows Poised's default (retention 8, reminders "2, 5", not responding 7, 30 days, 6 months, the default texts), greyed out with **Use Poised's default** ticked; opt-ins are badges.
2. Untick **Use Poised's default** under "Phone number for patients", type `0100 222 3333`. Untick it under "Keep episode data for (years)" and set 10. Click **Save the trust's settings**. Expected: "Trust A's settings were saved." appears beside the Save button, where you are (no scrolling up).
3. "Settings for" → **Hospital A2**. Expected: a blue note that A2 follows the trust unless you untick; the phone shows `0100 222 3333`, greyed, with **Use the trust's setting** ticked; the **Whole trust** group is there, with "Keep episode data" (10) and "Offer answers…" (Off) shown as badges "Set for the whole trust" (not editable here).
4. Untick **Use the trust's setting** for the phone, type `0100 444 5555`, and change the "What happens next" text. **Save Hospital A2's settings**. Expected: saved message.
5. Reload; check A2 keeps its own phone and text; **Hospital A1** shows the trust's `0100 222 3333`, greyed.
6. On A2, tick **Use the trust's setting** for the phone again and save. Expected: A2's phone is back to the trust's.
7. Switching the header's hospital box while on Settings keeps you on Settings.
8. Click Save with nothing changed. Expected: "Nothing has changed." beside the button, replacing any earlier "saved" message (not as well as it).

**T-16: Settings: invalid values are refused, nothing saved**
As Ada on the whole trust's settings, try each (one at a time), then Save. Expected: a red message naming the setting, and after a reload the old value is still there:
1. "Keep episode data for (years)": 0, then 101.
2. "Send reminders on these days…": `0, 99`, then `2, 2`, then `1, 2, 3, 4, 5, 6`, then `soon`.
3. "Invitation text message": `Hello {{firstName}}` (no link) → must include {{link}}; `Hi {{name}} {{link}}` → {{name}} isn't a placeholder.
4. "Email address for patients": `nope`. "Logo web address": `logo.png`. "Phone number for patients": `call us`.

**T-17: The default questionnaire**
1. As Ada on the whole trust's settings, untick the default for "Default health questionnaire" and open the list. Expected: "None" plus only Trust A's **published** questionnaires (drafts aren't offered). Typing part of a name narrows the list.
2. Pick one, save, reload. Expected: still chosen.

**T-18: Who can open Settings**
1. Sign in as **Alex Author** (whole-trust super clinician) and **Sue Super** (A1 super clinician). Expected: no Settings link in the header; typing `…/settings` after their hospital's address shows "Only the trust's admins can change its settings".
2. **Max Multi**: at Hospital B1 (Trust B, where he's admin) he has Settings and it shows Trust B's hospitals only. At Hospital A2 (Trust A, clinician) he has no Settings link, and `…/h/<A2>/settings` refuses him.
3. Signed in as Ada, open http://localhost:3000/api/o/<Trust B's id>/settings (Trust B's id from Max's address bar). Expected: `{"error":"Trust not found."}`.
4. Max on `…/h/<A2>/settings` (refused), switching the header's hospital box to **Hospital B1**: expected, he lands on B1's **Settings** (he manages Trust B), not Questionnaires.

### Reload, deep links and back

**T-19: Admin pages survive reload, deep links and Back**
1. As Sam, open a trust's page, reload: same page. Copy its address into a new tab: it opens there.
2. From Trusts → a trust → New trust, press the browser's **Back** twice: the trust page, then Trusts.
3. Signed out, open a trust's admin address directly. Expected: the sign-in page; after signing in as Sam, use the address again and it opens.
4. `/admin/nope` shows the not-found page.

### Editing details (Rahul, 13 Oct)

**T-20: Edit a trust's details**
1. As Sam, open **QA Trust One** → **Edit details**. Expected: a dialog filled with its name, code, logo address and contact.
2. Change the name to `QA Trust One Renamed`, the code to `qa1r`, the contact phone to `0100 999 9999`, clear the logo address. **Save**.
3. Expected: "QA Trust One Renamed's details were saved."; the heading, "Code QA1R", the phone and no logo; the Trusts list shows the new name and code. Its staff (Zoe Admin One) see the new trust name in their header after a reload.
4. **Edit details** again: code `TA` → "Another trust already has the code TA. Choose another." in the dialog, which stays open. Name empty → "Enter the trust's name.". Contact email `nope` → refused. Cancel: nothing changed.
5. As Ada, open http://localhost:3000/api/admin/orgs/<QA Trust One's id> → 403 (only the platform admin edits trusts).

**T-21: Edit a hospital's details**
1. On QA Trust One, click **Edit** on QA General's row. Expected: the dialog is filled with its name, address, phone, email, time zone ("Default (…)") and logo.
2. Change the name to `QA Central`, the address, clear the phone, set the email `central@qa.example`, and pick time zone `America/Toronto`. **Save**.
3. Expected: "QA Central's details were saved." just above the Hospitals table; the row shows the new name and address, "Trust's phone", the email and America/Toronto without "(default)". Zoe Clinician One sees "QA Central" in her header after a reload.
4. **Edit** QA Central again: name `QA North` (the other hospital) → "This trust already has a hospital called QA North."; address empty → refused. Set the time zone back to "Default (…)" and save: "(default)" is shown again.

### Fixes from verification 1

**T-22: A platform admin who is also a trust's staff can still sign in while it's suspended**
1. As Sam, create a trust `QA Trust Seven` (code `QA7`) with a fresh admin email, and add a hospital `QA Seven General`.
2. On QA Trust Seven's page, **Invite an admin** with Sam's own email, `sam.superadmin@poised.example`. Expected: "… already had an account, so they were made an admin of QA Trust Seven." Sam is listed as Active.
3. Sign out and back in as Sam. Expected: he goes straight into QA Seven General (now his only hospital), and the header has a **Platform admin** link back to the console.
4. Open QA Trust Seven in the console → **Suspend trust** → confirm.
5. Sign out, then sign in as **Sam Superadmin** again. Expected: **sign-in works** (no "suspended" message) and he lands on `/admin`. The start page (`/`) shows only the console card: QA Trust Seven isn't offered while it's suspended.
6. Open QA Trust Seven → **Re-activate trust**. Expected: it's active; the start page offers QA Seven General again.
7. In the second tab, the trust's first admin (from step 1) is refused while it's suspended and signs in once it's re-activated (as in T-12).

**T-23: A trust whose hospitals are all suspended**
1. As Sam, create `QA Trust Eight` (code `QA8`) with a fresh admin, `Zoe Admin Eight`, and add one hospital `QA Eight General`.
2. Second tab: sign in as Zoe Admin Eight; she goes straight into QA Eight General. Leave her there.
3. As Sam, **Suspend** QA Eight General (the hospital, not the trust).
4. Second tab: click Episodes. Expected: "Hospital not found." / "You don't have access to this hospital". Open `/` (the start page). Expected: **not** "QA Trust Eight has no hospitals in Poised yet"; the trust isn't listed ("You don't have access to any hospital yet").
5. Sign her out and sign in again as Zoe Admin Eight. Expected: the "Your trust or hospital is suspended…" refusal.
6. As Sam, **Re-activate** QA Eight General. Zoe Admin Eight signs in and goes straight into it.
7. Compare with a **new** trust (T-07): one that has no hospitals yet still shows the "no hospitals in Poised yet" note.

**Phone size:** nothing in this session is for patients, so no phone-size checks. (The admin pages scroll sideways in narrow windows like the other staff pages.)

## Regression (earlier sessions)

**R-1 (S08): hospital picker and switching.** Sign in as Max Multi: the picker lists Trust A (Hospital A2) and Trust B (Hospital B1); switching from the header changes trust and role.
**R-2 (S09): staff invites.** As Ada → Staff → invite a fresh email to Hospital A1 as Clinician; they appear Invited, can sign in, and become Active.
**R-3 (S08): All hospitals.** As Ada → Episodes → "All hospitals" lists A1 and A2 episodes (with A1 active).
**R-4 (S03/J3): the patient's link.** As Cara, create an episode on a published questionnaire, copy the patient link, open it in another tab: the questionnaire opens and saves an answer.
**R-5 (S01): authoring.** As Alex, open a questionnaire in the editor and change a question's title; reload; it's kept.

## Out of scope (don't report)

- Logo **uploads** (C8): only a web address now; the logo isn't shown to patients yet.
- Real Cognito sign-in and emails (S12, S17): invitations are never sent.
- The audit log (S11).
- Settings that nothing uses yet: reminders, auto-archive, retention, pre-fill, the "what happens next" text, message templates and the default HQ are stored and shown, but no screen uses them until later sessions (S13–S16, C3, C4).
- Deleting trusts or hospitals; a licence-holder note per instrument.
- The questionnaire editor saving a title only when the field loses focus (older than S10; known, in pending.md).
- `make platform-admin` (a terminal command): Rahul's hand check, not a browser case.
- Staff pages not laid out for phones (known since S08).
