# S09 test plan: staff invites, roles and removing access

For a tester who uses only the browser. You don't need to read any code. One case (T-16) needs one SQL command, and says so.

**Changes for retest 1 (after report 1, 13 Oct):**
- **F-1 fixed:** the Invite dialog starts afresh every time it opens (empty email and name, the first hospital, Clinician), after Cancel and after an invite. New case **T-19**.
- **New rule (Rahul, 13 Oct): a person can have several accesses in one trust,** e.g. Clinician at Hospital A1 *and* Super clinician at Hospital A2. Staff now shows **one row per person**, with each access on its own line ("Hospital A1 · Clinician") and its own **Change** and **Remove** buttons. Each row also has **Add access** and **Remove from trust** (plus **Resend invite** while invited). T-02, T-03, T-06–T-11 changed to match; new case **T-18**.
  - Inviting someone already in the trust **to a hospital they don't have yet now adds that access** (it used to be refused). Only inviting them to an access they already have is refused (T-07).
- **The second sign-in address is now http://tester.localhost:3000** (Chrome won't open `[::1]`).

## What changed in S09

- A trust's **whole-trust admins** have a new **Staff** screen (a **Staff** link in the header). Nobody else sees it: not clinicians, not super clinicians, not an admin of one hospital only.
- **Staff** lists everyone with access to the trust: name, email, **Access** (the whole trust, or one hospital), **Role**, and **Status**:
  - **Active**: they can sign in;
  - **Invited · expires <date>**: a new account that hasn't signed in yet;
  - **Invite expired**: a new account that didn't sign in within 7 days. It can't sign in until it's invited again (**Resend invite**).
- **Invite:** email, an optional name, the access (whole trust or one hospital) and the role. The person gets access at once.
  - **A new email** gets a new account. Locally no email is sent: the person simply appears in the development sign-in list, and must sign in for the first time within **7 days**.
  - **An email that already has an account** (for example someone from another trust) only gets access to this trust. No second account is made.
  - **Someone already in the trust** who's invited to a hospital they don't have gets that access added. Inviting them to an access they already have is refused: change it instead.
- **A person can have several accesses in one trust** (Rahul, 13 Oct): the whole trust and/or some of its hospitals, each with its own role. Staff shows one row per person, with one line per access.
- **Add access** gives someone on the staff another hospital (or the whole trust). Each access has its own **Change** (its hospital or role) and **Remove**. **Remove from trust** removes all of their accesses at once. Changes take effect on their **next click**. Their account and past actions are kept.
- A trust always keeps **at least one whole-trust admin**: the last one can't be removed or demoted.

## Setup

**Rahul runs, in his own terminal** (an automatic run does this for you):

```sh
docker compose up -d db
make migrate seed
make dev
```

Leave `make dev` running. Open **http://localhost:3000/login**.

**Signing in:** the login page lists every user; click a name to sign in. To switch user, click **Sign out** (top right), or open the login page again.

**Two people at once (needed in several cases):** a browser keeps one sign-in per web address. The same app also answers at **http://tester.localhost:3000**. So:
- keep **Ada Admin** signed in at **http://localhost:3000**;
- sign the **other person** in, in another tab, at **http://tester.localhost:3000/login**.

Each address keeps its own sign-in. Everything else is the same app and the same data.

**The seeded users:**

| User | Member of | Role |
| --- | --- | --- |
| Ada Admin | Trust A, whole trust (Hospitals A1 and A2) | **Admin** (manages Trust A's staff) |
| Alex Author | Trust A, whole trust | Super clinician |
| Cara Clinician | Trust A, Hospital A1 only | Clinician |
| Sue Super | Trust A, Hospital A1 only | Super clinician |
| Val Clinician | Trust A, Hospital A2 only | Clinician |
| Bea Author (`bea.author@hospital-b.example`) | Trust B, whole trust (Hospital B1) | Super clinician |
| Max Multi | Trust A, Hospital A2 only (Clinician) **and** Trust B, whole trust (**Admin**) | differs per trust |
| Sam Superadmin | no trust (a platform admin) | none |

**Names starting "Zoe E2E …"** in the sign-in list were made by the automated tests. Ignore them.

**Emails to invite:** use made-up addresses ending in `.example`. **Add your round number** to each one (e.g. `nia.r1@test.example`, then `nia.r2@test.example` in a retest), because an email already invited in an earlier round behaves differently. The cases below write `<n>` for that number.

**Dates:** "expires" dates are 7 days from today, in the app's date format (e.g. 20/10/2026).

## Test cases

### T-01: Only whole-trust admins see Staff
1. Sign in as **Ada Admin**, choose **Hospital A1**.
2. **Expected:** the header shows **Questionnaires · Episodes · Staff**.
3. Sign in as **Max Multi**. On the picker choose **Hospital B1** (Trust B, where he's the whole-trust admin).
4. **Expected:** the header shows **Staff**.
5. Switch the header's hospital box to **Hospital A2** (Trust A, where he's a clinician).
6. **Expected:** no **Staff** link.

### T-02: The staff list
1. As **Ada Admin** at Hospital A1, click **Staff**.
2. **Expected:**
   - the heading **Staff** and "Everyone with access to Trust A.";
   - rows for Ada Admin (marked **(you)**), Alex Author, Cara Clinician, Max Multi, Sue Super and Val Clinician (plus anyone invited earlier);
   - one row per person; the **Access and role** column has one line per access, matching the table in Setup (e.g. Cara: "Hospital A1 · Clinician"; Ada: "Whole trust · Admin"; Max: "Hospital A2 · Clinician"), each line with **Change** and **Remove**;
   - each row has **Add access** and **Remove from trust**;
   - **Status** is **Active** for all the seeded users;
   - no Trust B-only person (Bea Author) is listed.

### T-03: Invite a new clinician to one hospital (brief focus 1)
1. On Staff, click **Invite**.
2. **Expected:** a dialog "Invite someone to Trust A" with Email, Name (optional), Access (defaults to the first hospital), Role (defaults to Clinician), and "The invitation expires after 7 days."
3. Email `nia.r<n>@test.example`, Name `Nia New <n>`, Access **Hospital A1**, Role **Clinician**. Click **Send invite**.
4. **Expected:**
   - the dialog closes and a green message says "Invitation sent to nia.r<n>@test.example. It expires on <7 days from today>.";
   - a row for **Nia New <n>**: "Hospital A1 · Clinician" (with Change and Remove), **Invited · expires <that date>**, and **Add access**, **Resend invite**, **Remove from trust**.

### T-04: The invitee signs in and sees only their hospital (brief focus 1)
1. In a second tab, open **http://tester.localhost:3000/login**.
2. **Expected:** **Nia New <n>** is in the list.
3. Click it.
4. **Expected:**
   - you go straight to **Hospital A1's Episodes** (she has one hospital, so no picker);
   - the header shows **Trust A**, the hospital box **Hospital A1**, the role badge **Clinician**, and **no Staff** link;
   - opening the hospital box lists **only Hospital A1**.
5. Back in Ada's tab (localhost), reload Staff.
6. **Expected:** Nia's status is now **Active**, and **Resend invite** is gone.

### T-05: Invite someone to the whole trust, without a name
1. As Ada on Staff, **Invite**: Email `wendy.r<n>@test.example`, Name empty, Access **Whole trust**, Role **Super clinician**. **Send invite**.
2. **Expected:** a row whose name is the email itself, "Whole trust · Super clinician", Invited.
3. Sign this person in at http://tester.localhost:3000/login (sign Nia out first, or just pick them from the login page).
4. **Expected:** the hospital picker shows Trust A with **both** Hospital A1 and Hospital A2, marked "Super clinician, whole trust".

### T-06: An existing account only gets access, never a second account (STF-02, brief focus 2)
1. As Ada on Staff, **Invite**: Email `BEA.AUTHOR@hospital-b.example` (capitals on purpose), Access **Hospital A2**, Role **Clinician**. **Send invite**.
2. **Expected:**
   - a green message "Bea Author already had an account, so they were given access to Hospital A2 as Clinician. No new account was made.";
   - a row **Bea Author**, "Hospital A2 · Clinician", **Active** (not Invited: her account already works).
3. Open the login page (either address).
4. **Expected:** **Bea Author** appears **once**; there's no second Bea or a row with her email as the name.
5. Sign in as **Bea Author** at http://tester.localhost:3000.
6. **Expected:** the picker shows **Trust A → Hospital A2 (Clinician)** and **Trust B → Hospital B1**, "Super clinician, whole trust".
7. **Clean up:** as Ada, **Remove from trust** on Bea Author's row (confirm). She keeps Trust B.

### T-07: An access someone already has can't be invited again
1. As Ada, **Invite**: Email `  Cara.Clinician@Hospital-A.example ` (capitals and spaces on purpose), Access **Hospital A1**, any role. **Send invite**.
2. **Expected:** the dialog stays open with a red message "Cara Clinician already has that access. Change it in the list instead." Cara's row is unchanged.

### T-08: Changing someone's role changes what they can do (brief focus 3)
1. At http://tester.localhost:3000 sign in as **Nia New <n>** (from T-03). Open **Questionnaires**.
2. **Expected:** no **New Questionnaire** button (she's a clinician).
3. In Ada's tab, Staff → **Change** next to Nia's "Hospital A1 · Clinician".
4. **Expected:** a dialog "Change Nia New <n>'s access at Hospital A1" with Access **Hospital A1** and Role **Clinician** filled in.
5. Choose Access **Whole trust**, Role **Super clinician**. **Save**.
6. **Expected:** a green message "Nia New <n> is now Super clinician, whole trust."; her row shows "Whole trust · Super clinician".
7. In Nia's tab, reload the page.
8. **Expected:** the role badge reads **Super clinician**, and **New Questionnaire** is there. She can now also switch to **Hospital A2** in the hospital box.
9. As Ada, **Change** that access back to **Hospital A1**, **Clinician**. In Nia's tab, reload.
10. **Expected:** **New Questionnaire** is gone and the badge reads **Clinician**.

### T-09: Changing someone's hospital moves them
1. As Ada, **Change** Nia's "Hospital A1 · Clinician" to Access **Hospital A2**, Role **Clinician**. **Save**.
2. In Nia's tab (still on a Hospital A1 page), click **Episodes** in the header.
3. **Expected:** a red "Hospital not found." (her access to A1 ended on that click).
4. Reload.
5. **Expected:** "You don't have access to this hospital" with a link **Choose a hospital**. Following it takes her into **Hospital A2**.

### T-10: Removing someone ends their access on their next click (STF-04, brief focus 4)
1. In Nia's tab, open Hospital A2's **Episodes** (she's there after T-09).
2. As Ada, Staff → **Remove from trust** on Nia's row.
3. **Expected:** a dialog "Remove Nia New <n> from Trust A?" saying they lose access to Trust A at once, at every hospital, and their account and actions are kept. Click **Remove**.
4. **Expected:** a green message "Nia New <n> no longer has access to Trust A."; her row is gone.
5. In Nia's tab, click **Questionnaires** in the header.
6. **Expected:** a red "Hospital not found.". Reload: "You don't have access to this hospital".
7. Open the login page at http://tester.localhost:3000 and sign in as Nia again.
8. **Expected:** she's still in the list (the account is kept), and after signing in sees "You don't have access to any hospital yet. Ask an admin of your trust to invite you."
9. **Cancel works:** as Ada, click **Remove from trust** on Wendy's row (from T-05), then **Cancel**. **Expected:** Wendy is still listed.

### T-11: The last whole-trust admin can't be removed or demoted
1. As Ada on Staff, **Change** next to **Ada Admin (you)**'s "Whole trust · Admin": choose Role **Clinician** (Access Whole trust). **Save**.
2. **Expected:** the dialog shows a red "The trust needs at least one admin for the whole trust. Make someone else an admin first." Nothing changes.
3. Try Access **Hospital A1**, Role **Admin**. **Expected:** the same refusal (an admin of one hospital isn't a whole-trust admin).
4. **Remove** next to her "Whole trust · Admin", then **Remove** in the dialog. **Expected:** the same red message in the dialog. Then **Remove from trust** on her row, **Remove**: the same message. Ada is still listed.

### T-12: Clinicians and super clinicians can't see or use Staff (brief focus 5)
1. Sign in as **Cara Clinician** (localhost). **Expected:** no **Staff** link.
2. Change the address to `…/o/<trust id>/h/<hospital id>/staff` (replace the last part of the address, e.g. `/episodes`, with `/staff`).
3. **Expected:** a red "Only the trust's admins can manage staff"; no list, no Invite button.
4. Repeat steps 1–3 as **Alex Author** (a whole-trust super clinician) and as **Sue Super**. **Expected:** the same.

### T-13: A super clinician can't invite, even by calling the API (brief focus 5)
This case uses the browser's developer console, as the test plan allows.
1. Signed in as **Alex Author**, on any Trust A page, note the trust id (the part after `/o/` in the address).
2. In the console run:
   ```js
   await fetch("/api/o/<trust id>/staff/invites", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({email: "sneaky.r<n>@test.example", hospitalId: null, role: "admin"})}).then(async r => [r.status, await r.text()])
   ```
3. **Expected:** `403` and `{"error":"Only the trust's admins can manage staff."}`.
4. As Ada, reload Staff. **Expected:** no `sneaky.r<n>@test.example`. On the login page there's no such person either.

### T-14: Another trust's staff can't be seen
1. As **Max Multi** at **Hospital B1**, click **Staff**.
2. **Expected:** only Trust B people (Bea Author, Max Multi, plus any you invited there); "Everyone with access to Trust B."; no Trust A people such as Cara.
3. As **Ada Admin**, change the address to a Trust B hospital's staff page. (Get the Trust B ids from Max's address bar in step 1: `/o/<trust B id>/h/<hospital B1 id>/staff`.)
4. **Expected:** "You don't have access to this hospital".
5. As Ada, in the console: `await fetch("/api/o/<trust B id>/staff").then(r => r.status)`. **Expected:** `404`.

### T-15: An invalid email is refused (brief focus 6)
1. As Ada, **Invite**. Try each email in turn, clicking **Send invite** each time:
   - (empty)
   - `not-an-email`
   - `ann@localhost`
   - `Ann <ann.r<n>@test.example>`
   - `two@at@test.example`
2. **Expected:** the dialog stays open each time with a red message: "Enter the person's email address." for the empty one, and "Enter a valid email address, like name@hospital.org." for the others.
3. Click **Cancel**, reload Staff. **Expected:** none of them was added, and none is on the login page.

### T-16: An expired invite (brief focus 7; **uses one SQL step**)
Invites last 7 days. To avoid waiting, one SQL command ages the invite.
1. As Ada, **Invite** `late.r<n>@test.example`, Name `Lee Late <n>`, Hospital A1, Clinician.
2. In a terminal, in the repo folder, run (the `PATH=…` part is needed on Rahul's Mac; replace `<n>`):
   ```sh
   PATH="/Applications/Docker.app/Contents/Resources/bin:$PATH" docker compose exec -T db psql -U sj -d sj -c "UPDATE staff_invites SET expires_at = now() - interval '1 minute' WHERE accepted_at IS NULL AND user_id = (SELECT id FROM users WHERE email = 'late.r<n>@test.example');"
   ```
   **Expected:** `UPDATE 1`.
3. As Ada, reload Staff. **Expected:** Lee's status is **Invite expired**, with **Resend invite**.
4. At http://tester.localhost:3000/login, click **Lee Late <n>**.
5. **Expected:** a red message on the login page "Your invitation has expired. Ask your trust's admin to invite you again."; you stay on the login page.
6. As Ada, click **Resend invite** on Lee's row.
7. **Expected:** a green "Invitation sent again to late.r<n>@test.example. It expires on <7 days from today>."; the status is **Invited · expires …**.
8. Sign in as Lee at http://tester.localhost:3000. **Expected:** it works, straight into Hospital A1's Episodes. Ada's Staff then shows Lee **Active** after a reload.

### T-17: Reload, deep links, back button and switching hospital
1. As Ada on Staff, reload. **Expected:** the Staff page again, with the list.
2. Copy the Staff address, open a new tab (same address family, localhost), paste it. **Expected:** the Staff page opens directly.
3. From Staff click **Episodes**, then the browser's **Back** button. **Expected:** back on Staff, with the list.
4. On Staff, switch the header's hospital box to **Hospital A2**. **Expected:** you stay on **Staff** (now under Hospital A2's address), with the same Trust A list.
5. **Cancel** an invite: click **Invite**, type an email, click **Cancel**. **Expected:** nothing is added.

### T-18: Several accesses in one trust (Rahul, 13 Oct)
1. As Ada, **Invite** `multi.r<n>@test.example`, Name `Mo Multi <n>`, **Hospital A1**, **Clinician**.
2. On Mo's row click **Add access**.
3. **Expected:** a dialog "Give Mo Multi <n> more access" showing Mo's email (not editable), Access (offering Whole trust and Hospital A2, **not** Hospital A1, which Mo already has) and Role.
4. Choose **Hospital A2**, **Super clinician**, click **Add access**.
5. **Expected:** a green "Mo Multi <n> now also has access to Hospital A2 as Super clinician."; Mo's single row shows two lines, "Hospital A1 · Clinician" and "Hospital A2 · Super clinician".
6. Sign Mo in at http://tester.localhost:3000. **Expected:** the hospital picker shows Trust A with Hospital A1 (Clinician) and Hospital A2 (Super clinician).
7. As Ada, **Invite** `multi.r<n>@test.example` again with **Whole trust**, **Clinician**. **Expected:** it's added as a third line, "Whole trust · Clinician", listed first (no "already has" refusal, because Mo didn't have the whole trust).
8. **Change** Mo's "Hospital A2 · Super clinician": **Expected:** the Access list offers Hospital A2 (its own) but not Hospital A1 or Whole trust (Mo has those). Choose Role **Clinician**, **Save**. **Expected:** that line reads "Hospital A2 · Clinician"; the others are unchanged.
9. **Remove** next to "Hospital A1 · Clinician". **Expected:** a dialog "Remove Mo Multi <n>'s access at Hospital A1?" saying their other access stays. **Remove**. **Expected:** "Mo Multi <n> no longer has access to Hospital A1."; the row keeps the whole-trust and A2 lines.
10. In Mo's tab, open Hospital A1's Episodes. **Expected:** it still works, because Mo has the whole trust.
11. As Ada, **Remove** next to "Whole trust · Clinician", **Remove**. In Mo's tab, click **Episodes** at Hospital A1. **Expected:** "Hospital not found."; Hospital A2 still works after a reload.
12. **Remove** next to Mo's last line ("Hospital A2 · Clinician"). **Expected:** the dialog says they lose access to Trust A at once; after **Remove**, "Mo Multi <n> no longer has access to Trust A." and Mo's row is gone.

### T-19: The Invite dialog starts afresh every time (report 1, F-1)
1. As Ada on Staff, click **Invite**. **Expected:** Email and Name empty, Access **Hospital A1**, Role **Clinician**.
2. Type `cancel.r<n>@test.example`, choose **Whole trust** and **Admin**, then **Cancel**.
3. Click **Invite** again. **Expected:** Email empty, **Hospital A1**, **Clinician**.
4. Invite `fresh.r<n>@test.example` with **Whole trust**, **Super clinician**; **Send invite**.
5. Click **Invite** again. **Expected:** Email and Name empty, **Hospital A1**, **Clinician**.
6. **Add access** and **Change** dialogs also open with the row's own values each time, not the last dialog's.

## Regression (earlier sessions)

- **R-01 (S08, picker):** sign in as **Max Multi**. **Expected:** the picker shows Trust A (Hospital A2, Clinician) and Trust B ("Admin, whole trust"); choosing Hospital B1 opens it.
- **R-02 (S08, hospital-only access):** sign in as **Cara Clinician**. **Expected:** straight into Hospital A1's Episodes; the hospital box lists only Hospital A1; an episode opens.
- **R-03 (S08, trust content):** as **Alex Author**, Questionnaires shows **New Questionnaire**; as **Sue Super** it doesn't.
- **R-04 (episodes):** as Cara, **New episode** lists the patients (Jo Bloggs, Pat Example, Robin Sample) and the published HQs; creating one works.
- **R-05 (patient link, phone size 390×844):** open an episode's patient link in a 390×844 window. **Expected:** the HQ shows and can be answered, as before.

## Out of scope (don't report)

- Real emails and Cognito sign-in (S12): locally nobody gets an email; invited people appear in the sign-in list at once.
- The superadmin's screens and the trust's first-admin invite (S10).
- The audit log (S11).
- The Staff table on a phone-sized window: it scrolls sideways inside itself (staff use desktops).
- "Zoe E2E …" people in the sign-in list (made by the automated tests).
- That someone with an expired invite still has a row in Staff (they keep their access until removed; they just can't sign in until invited again).
- A user's buttons (e.g. **New Questionnaire**) not updating until they reload after an admin changes their role. What the server allows changes on their next click; the screen catches up on reload.
