# S09 test report 1

**Summary: 22 passed, 1 failed (Minor), 0 blocked.** The 17 plan cases and 5 regression cases all pass as written. The one failure is a Minor bug in the Invite dialog, found while testing T-03/T-06/T-17: it keeps what you entered last time instead of going back to its defaults.

**Environment**
- Commit `80c01e8`. The working tree was not clean: it had the S09 build's uncommitted code plus the planning-doc changes.
- Tested on 13 Oct 2026, 00:40–01:05 BST, against `make dev` run by the conductor (API :8080, web :3000), freshly migrated and seeded.
- Browser: Claude in Chrome, in automatic mode (round `<n>` = 1, so emails are `*.r1@test.example`).
- **Second sign-in address:** I used **http://tester.localhost:3000** instead of `http://[::1]:3000`. The Chrome extension refuses to navigate to `[::1]`, and `127.0.0.1:3000` is refused because Vite listens only on `[::1]:3000`. `tester.localhost` reaches the same dev server and keeps its own cookie, so it works the same way for the two-people cases.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Only whole-trust admins see Staff | Pass | Ada at A1: Questionnaires · Episodes · Staff. Max at B1: Staff. Max switched to A2: no Staff, and the switch from the Staff page went to A2's Episodes. |
| T-02 The staff list | Pass | Heading and "Everyone with access to Trust A." are right. All 6 seeded people are there with the right access, role and Active status. Ada is marked (you). No Bea. |
| T-03 Invite a new clinician | Pass | The dialog's fields, defaults and expiry line are as planned. Green message "…expires on 20/10/2026." Row shows Invited · expires 20/10/2026, with Resend/Change/Remove. (See F-1 for what the dialog does when opened again.) |
| T-04 Invitee signs in | Pass | Nia New 1 is in the list and goes straight to A1 Episodes. Header shows Trust A, Hospital A1, Clinician, no Staff. The box lists only A1. Ada's reload shows Active and no Resend. |
| T-05 Whole trust, no name | Pass | The row's name is the email, Whole trust, Super clinician, Invited. The picker shows A1 and A2, marked "Super clinician, whole trust". |
| T-06 Existing account (STF-02) | Pass | The message matches the plan word for word. Bea is Active in Trust A, appears once on the login page, and her picker shows Trust A → A2 (Clinician) plus Trust B. After clean-up she's removed from A and keeps Trust B (goes straight into B1). |
| T-07 Already on staff | Pass | Spaces and capitals are handled. The red message is exact, and the dialog stays open. Enter submits the form. |
| T-08 Role change takes effect | Pass | The Change dialog is pre-filled. Message "Nia New 1 is now Super clinician, whole trust." After a reload: Super clinician badge, New Questionnaire shown, A2 in the box. Changing back reverses all three. |
| T-09 Hospital change moves them | Pass | Next click: "Hospital not found." Reload: "You don't have access to this hospital" → Choose a hospital → A2 Episodes. |
| T-10 Remove (STF-04) | Pass | Dialog wording is right. Message "Nia New 1 no longer has access to Trust A." Next click: "Hospital not found."; reload: no access. She's still on the login page, and signing in gives "You don't have access to any hospital yet". Cancel on Wendy keeps her. |
| T-11 Last whole-trust admin | Pass | The demote to Clinician, the change to A1 + Admin and the Remove are all refused with the exact message (PATCH 409 ×2). Ada is unchanged. |
| T-12 Clinicians / super clinicians can't see Staff | Pass | Cara, Alex and Sue get no Staff link; the `/staff` URL shows "Only the trust's admins can manage staff" with no list or Invite. GET staff API returns 403. |
| T-13 Super clinician can't invite via API | Pass | 403 `{"error":"Only the trust's admins can manage staff."}`. No sneaky.r1 in Staff, on the login page or in the `users` table (checked in the database). |
| T-14 Another trust's staff | Pass | Max at B1 sees only Bea and Max, with "Everyone with access to Trust B." Ada at Trust B's staff URL gets no access; the console fetch returns 404. |
| T-15 Invalid email | Pass | Empty gives "Enter the person's email address."; the other four give the valid-email message (server 400, checked in the network log). Nothing was added to Staff, the login page or the database. |
| T-16 Expired invite | Pass | The SQL step gave `UPDATE 1`. Status shows Invite expired + Resend. Lee's sign-in shows the red message and stays on the login page. Resend shows "Invitation sent again… 20/10/2026." and Invited status. Lee then signs straight into A1 Episodes, and Ada sees Active. |
| T-17 Reload, deep link, back, switch, cancel | Pass | Reload, a deep link in a new tab, and Episodes → Back all return to the Staff list. Switching to A2 keeps Staff under A2's URL with the same list. Cancel adds nothing. |
| F-1 Invite dialog keeps old values (exploration of T-03/T-17) | **Fail (Minor)** | See below. |
| R-01 Max picker | Pass | Trust A (A2, Clinician) and Trust B ("Admin, whole trust"); B1 opens. |
| R-02 Cara hospital-only | Pass | Goes straight to A1 Episodes, the box lists only A1, and an episode opens. |
| R-03 Trust content | Pass | Alex sees New Questionnaire; Sue doesn't. |
| R-04 New episode | Pass | Created a Jo Bloggs episode (procedure "MT S09 knee") from a published HQ; it opens. |
| R-05 Patient link at 390 wide | Pass | Shown in a 390×700 iframe: the HQ shows and "Do you smoke?" can be answered ("Saved"). |

## Failures

### F-1 (Minor): the Invite dialog doesn't reset when it's opened again

The plan (T-03) says Access defaults to the first hospital and Role to Clinician. That holds only the first time the dialog opens after a page load. After that:
- **After a successful invite,** the next Invite opens with the **previous Access and Role**. Email and name are cleared.
- **After Cancel,** the next Invite opens with **everything from last time**: email, name, Access and Role. Only the red error is cleared.

**Steps to reproduce** (Ada Admin, http://localhost:3000/o/6ef150fb-…/h/…000a/staff):
1. Reload Staff and click **Invite**. Access is Hospital A1 and Role is Clinician, as expected.
2. Invite `wendy.r1@test.example` with **Whole trust** and **Super clinician**, then Send invite.
3. Click **Invite** again. **Expected:** Hospital A1, Clinician. **Actual:** Whole trust, Super clinician.
4. Or: type `cancel.r1@test.example`, choose Role **Admin**, click **Cancel**, then click **Invite** again. **Actual:** the email is still filled in and the Role is still **Admin**.

**Why it matters:** an admin inviting several people in a row can easily give the next person the last person's wider access, such as Whole trust + Admin, without noticing. The invite takes effect at once.

**Likely cause** (for the builder; I didn't change it): `InviteDialog` in `apps/web/src/features/staff/StaffPage.tsx` stays mounted, and its `useState` values (email, name, access) are never reset on close or open.

No console errors or unexpected network errors.

## Found while exploring

- **Hospital-only admin:** I made wendy an Admin of Hospital A1 only through the API, as Ada. She gets no Staff link, the Staff URL shows the refusal, and GET staff returns 403. This is as the brief says. She also has no New Questionnaire button, which is S08's trust-content rule and not S09's concern. I then set her back to whole-trust Super clinician.
- **Cross-trust and invalid API calls, as Ada** (all refused correctly):
  - PATCH, DELETE or invite on Trust B → 404 "Trust not found."
  - PATCH or DELETE for Bea in Trust A after her removal → 404 "That person isn't on this trust's staff."
  - Invite or PATCH with Trust B's hospital id → 400 "Choose the whole trust or one of this trust's hospitals."
  - Role `superadmin` → 400.
  - As Alex (super clinician), PATCH or DELETE on Cara → 403.
- **Second admin present:** making wendy a whole-trust Admin and then demoting her both succeed, so the last-admin guard only blocks the last admin.
- **Double-clicking Send invite** made exactly one account and one invite (checked in the database).
- **Name with emoji and leading/trailing spaces** (`  Dóra 🙂 Double  `) is saved trimmed and shows correctly in the list.
- **Odd inputs:**
  - A 300-character local-part email → the valid-email message.
  - A 300- or 1,200-character name → 400 "The name must be 200 characters or fewer."
  - A 5,000-character name → 400 "The request could not be read." This only happens through the API (the body is over the 4 KB limit), so it's cosmetic.
- **Re-inviting a removed person** (Nia, through the API) gives 201 and puts her back as Active, because her account already worked.
- **Failed sign-in on the dev login page:** after Lee's expired-invite sign-in failed, that tab was still signed in as the previous user (Sue). This is expected for the dev stub and just noted.
- **Tool artifacts, not app bugs** (no action needed):
  - The extension's network log shows `503` for every 204 response (dev/login, DELETE staff). Direct `fetch` and curl return 204.
  - Mantine modal transitions stall in a tab that isn't in front, so the first click after switching tabs was sometimes swallowed.

## Console and network errors

No console errors or warnings on the Staff, Episodes, login or patient pages. The only 4xx responses were the ones the cases expected (400 validation, 403, 404, 409). The "503"s are the tool artifact described above.

## GIFs

Saved to `~/Downloads/`:
- `S09-T03-T04-happy-path.gif`: invite Nia, she signs in, and Ada sees her Active.
- `S09-T16-expired-invite.gif`: Invite expired → resend → Active.

I didn't record a separate GIF for F-1; the steps above reproduce it in seconds.

## Test data left behind

All in Trust A:
- `nia.r1@test.example` (Nia New 1, A1 Clinician, Active);
- `wendy.r1@test.example` (whole trust, Super clinician);
- `late.r1@test.example` (Lee Late 1);
- `dbl.r1@test.example` (Dóra 🙂 Double, Invited);
- the episode "MT S09 knee" for Jo Bloggs at A1.

Bea Author is back to Trust B only.
