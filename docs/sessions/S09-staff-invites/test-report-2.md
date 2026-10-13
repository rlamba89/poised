# S09 test report 2 (retest)

**Summary: 24 passed, 0 failed, 0 blocked.** All 19 plan cases, including the changed T-02 to T-11 and the new T-18 and T-19, and all 5 regression cases pass as written. F-1 from report 1 is fixed: the Invite, Add access and Change dialogs now start fresh every time they open. Because the rework touched most of the Staff screen, I retested every case, not only the changed ones.

**Environment**
- Commit `80c01e8`. The working tree was not clean: it had the S09 build's uncommitted code plus the planning-doc changes.
- Tested on 13 Oct 2026, 01:18 to 01:39 BST, against the `make dev` the conductor was running (API :8080, web :3000).
- Browser: Claude in Chrome, in automatic mode. This is round 2, so the emails are `*.r2@test.example`.
- Ada was signed in at http://localhost:3000. The second person was signed in at http://tester.localhost:3000.
- The database still had round 1's people (`nia.r1`, `wendy.r1`, `late.r1`, `dbl.r1`), so it wasn't freshly seeded. This made no difference to the cases.
- On the development sign-in list I clicked a person's button with a one-line script (`button.click()`), not the mouse, to save time. Everything else went through the real UI, except where a case says to use the console or SQL, and the API checks under "Found while exploring".

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Only whole-trust admins see Staff | Pass | Ada at A1 sees Questionnaires · Episodes · Staff. Max sees Staff at B1. After Max switches to A2, there's no Staff link and he lands on A2 Episodes. |
| T-02 The staff list | Pass | The heading and "Everyone with access to Trust A." are right. One row per person, each access on its own line ("Hospital A1 · Clinician", "Whole trust · Admin", Max "Hospital A2 · Clinician") with Change and Remove. Every row has Add access and Remove from trust. All seeded people are Active, Ada is marked (you), and Bea isn't listed. |
| T-03 Invite a new clinician | Pass | The dialog's fields and defaults are right (Hospital A1, Clinician, "expires after 7 days"). The message is "Invitation sent to nia.r2@test.example. It expires on 20/10/2026." The row shows "Hospital A1 · Clinician", Invited · expires 20/10/2026, and Add access, Resend invite and Remove from trust. |
| T-04 Invitee signs in | Pass | Nia New 2 is on the sign-in list and goes straight to A1 Episodes. The header shows Trust A, Hospital A1 and Clinician, with no Staff link, and the box lists only A1. After Ada reloads, Nia is Active and Resend invite is gone. |
| T-05 Whole trust, no name | Pass | The row's name is the email, with "Whole trust · Super clinician" and Invited. The picker shows A1 and A2 under "Super clinician, whole trust". |
| T-06 Existing account (STF-02) | Pass | The message matches the plan word for word. Bea's row shows "Hospital A2 · Clinician" and Active. She appears once on the sign-in list. Her picker shows Trust A with A2 (Clinician) and Trust B with B1 ("Super clinician, whole trust"). Clean-up with Remove from trust gives "Bea Author no longer has access to Trust A." |
| T-07 Access already held | Pass | `  Cara.Clinician@Hospital-A.example ` at A1 shows "Cara Clinician already has that access. Change it in the list instead." The dialog stays open and Cara's row is unchanged. Enter submits the form. |
| T-08 Role change takes effect | Pass | The Change dialog is "Change Nia New 2's access at Hospital A1", filled with A1 and Clinician. The message is "Nia New 2 is now Super clinician, whole trust." After a reload: the Super clinician badge, New Questionnaire, and A2 in the box. Changing back reverses all three. |
| T-09 Hospital change moves them | Pass | Her next click shows "Hospital not found." A reload shows "You don't have access to this hospital", and Choose a hospital takes her to A2 Episodes. |
| T-10 Remove from trust (STF-04) | Pass | The dialog wording is right. The message is "Nia New 2 no longer has access to Trust A." and her row is gone. Her next click shows "Hospital not found." and a reload shows no access. She's still on the sign-in list and gets "You don't have access to any hospital yet". Cancel on wendy.r2 keeps her. |
| T-11 Last whole-trust admin | Pass | The demote to Clinician, the change to A1 + Admin, the per-access Remove and Remove from trust all show the exact red message in the dialog (PATCH 409, DELETE access 409, DELETE person 409). Ada is unchanged. |
| T-12 Clinicians and super clinicians can't see Staff | Pass | Cara, Alex and Sue get no Staff link. The `/staff` URL shows "Only the trust's admins can manage staff" with no list. The GET staff API returns 403. |
| T-13 Super clinician can't invite via the API | Pass | 403 `{"error":"Only the trust's admins can manage staff."}`. There's no sneaky.r2 in Staff, on the sign-in list or in `users` (checked in the database). |
| T-14 Another trust's staff | Pass | Max at B1 sees only Bea and Max, with "Everyone with access to Trust B." Ada at Trust B's staff URL gets "You don't have access to this hospital", and the console fetch returns 404. |
| T-15 Invalid email | Pass | Empty gives "Enter the person's email address.". The other four give "Enter a valid email address, like name@hospital.org." (POST 400 each time). Nothing was added to Staff or the sign-in list. |
| T-16 Expired invite | Pass | The SQL step gave `UPDATE 1`. Lee shows Invite expired with Resend invite. Lee's sign-in shows the red expired message and stays on the sign-in page. Resend shows "Invitation sent again to late.r2@test.example. It expires on 20/10/2026." and Invited. Lee then goes straight into A1 Episodes, and Ada sees Active after a reload. |
| T-17 Reload, deep link, back, switch, cancel | Pass | A reload and a deep link in a new tab open Staff directly. Episodes then Back returns to the Staff list. Switching to A2 keeps Staff, under A2's address, with the same list. Cancel adds nothing (cancel.r2 isn't in the database). |
| T-18 Several accesses in one trust | Pass | All 12 steps work as written. Add access offers Whole trust and A2, not A1. Mo's picker shows A1 (Clinician) and A2 (Super clinician). Re-inviting with Whole trust adds a third line, listed first. Change offers only A2 for that access. Each per-access Remove gives the right message, and Mo keeps A1 through the whole trust. Removing the whole-trust line gives "Hospital not found." at A1 while A2 still works. Removing the last line gives "…no longer has access to Trust A." and the row disappears. See note 3 below about step 11. |
| T-19 Dialogs start fresh (F-1) | Pass | Invite after Cancel (with Whole trust, Admin and an error showing) reopens empty with A1, Clinician and no error. Invite after a successful invite also reopens empty with A1 and Clinician. Add access on another row opens with that row's options and Clinician, not the cancelled Whole trust + Admin. Change on Cara after a cancelled change to Admin reopens with A1 and Clinician. Change on fresh.r2 shows Whole trust and Super clinician. |
| R-01 Max picker | Pass | Trust A (A2, Clinician) and Trust B ("Admin, whole trust"). B1 opens. |
| R-02 Cara, hospital-only access | Pass | She goes straight to A1 Episodes, the box lists only A1, and an episode opens. |
| R-03 Trust content | Pass | Alex sees New Questionnaire; Sue doesn't. |
| R-04 New episode | Pass | As Cara, created a Jo Bloggs episode ("MT S09 r2 hip") from a published HQ. It opens with a patient link. |
| R-05 Patient link at 390 wide | Pass | Resizing the window didn't take effect, so I loaded the link in a 390×700 iframe. The HQ fits without sideways scrolling. "Do you smoke?" was answered (No), and it shows "Saved". |

## Failures

None.

## Found while exploring

None of these is a failure; the builder may want to look at 1 and 2.

1. **A change to your own access always sends you to the hospital picker, with no message.** As Ada, I used **Add access** on my own row (Hospital A1 · Admin). The access was added, but I landed on "Choose a hospital" and the green confirmation never appeared. The same happens when you remove one of your own extra accesses. This is deliberate: `StaffPage.tsx` sends you to `/` after any change to yourself, because a change could end your right to manage staff. But **adding** access can never reduce what you can do, and losing the confirmation is a little confusing. I'd rate it Cosmetic or Minor at most.
2. **Wording when someone already on the staff is given a new access through Invite (T-18 step 7).** The message is "Mo Multi 2 already had an account, so they were given access to the whole trust as Clinician. No new account was made." It's true, but it's the message for someone from another trust, even though Mo was invited by this admin minutes earlier. The plan doesn't specify this wording. Cosmetic.
3. **T-18 step 11:** Mo was already on A1 Episodes, so clicking **Episodes** again made no request and the list stayed on screen. The next click to another page (Questionnaires, then Episodes) showed "Hospital not found." So the server rule holds, and only the plan's wording is loose. After losing the whole trust, Mo's A1 Episodes page still showed the "Hospital A1 | All hospitals" switch until a reload, which the plan lists as out of scope.
4. **Dialog wording (Cosmetic):**
   - "Remove Mo Multi 2's access at the whole trust?" would read better as "…access to the whole trust?".
   - When the person keeps that hospital through their whole-trust access, "They lose access to Hospital A1 at once" isn't quite true, although the next sentence, "Their other access stays", softens it.
5. **API checks, all refused correctly:**
   - As Ada, with A1 · Admin added to her own whole-trust Admin:
     - DELETE her whole-trust access → 409, the last-admin message. A hospital admin doesn't count.
     - PATCH whole trust → A1 when she already has A1 → 409 "They already have that access. Change or remove that one instead."
     - Invite herself to A1 again → 409 "already has that access".
   - As Ada:
     - PATCH Cara with Trust B's hospital id → 400.
     - DELETE an access Cara doesn't have (A2, Trust B's B1, or `not-a-uuid`) → 404 "That person doesn't have that access."
     - Role `superadmin` → 400.
     - DELETE through Trust B's URL → 404 "Trust not found."
   - As Max, a Trust B admin, acting through Trust B's URL on Cara, a Trust A person:
     - PATCH → 404.
     - DELETE person → 404 "That person isn't on this trust's staff."
     - Invite with Trust A's hospital → 400.
     - GET Trust A's staff list → 403.
6. **Double-clicking Add access** sent exactly one POST.
7. **Tool artefacts, not app bugs:**
   - A Mantine modal opens or closes fully only once a screenshot is taken in this tool.
   - The first click after a dialog closes was sometimes swallowed.
   - `resize_window` didn't change the page width, so R-05 used an iframe.

## Console and network errors

There were no console errors or warnings on the Staff page, including with dialogs open. The only 4xx responses were the ones the cases expected (400 validation, 403, 404, 409). The patient questionnaire request was sent twice per page load, which is probably React's development-only double run (StrictMode) and harmless.

## GIFs

`~/Downloads/S09-r2-T03-T04-happy-path.gif`: invite Nia New 2, she signs in, and Ada sees her Active. There were no failures to record.

## Test data left behind (Trust A)

- `fresh.r2@test.example`: whole trust, Super clinician, Invited.
- `wendy.r2@test.example`: whole trust, Super clinician, Active.
- `late.r2@test.example` (Lee Late 2): A1, Clinician, Active.
- `nia.r2@test.example` and `multi.r2@test.example`: removed from Trust A, but their accounts remain.
- Episode "MT S09 r2 hip" for Jo Bloggs at A1, with "Do you smoke?" answered No.
- Bea Author is back to Trust B only. Ada is back to "Whole trust · Admin" only.
