# S15 test report 1

**Summary: 23 passed, 0 failed, 1 blocked** (18 plan cases + 6 regression cases).

- Environment: HEAD `08a3be0`, working tree not clean (S15's uncommitted build plus the planning-doc changes), 13 Oct 2026, 15:26–16:45 BST. `make dev` was started by the conductor (API :8080, web :3000).
- Browser: Claude in Chrome. Staff pages ran at desktop size on `localhost:3000`. Patient pages ran in a 390×844 iframe ("phone") on `tester.localhost:3000` and on `phone2.localhost:3000`, so two patients could be signed in at once.
- Test data (all made up, initials `QZ`):
  - Sam HomeQZ (17/05/1980, 07700 900501) at A1 and B1;
  - Sam ReturnQZ (03/04/1975, return.qz@ → new.qz@example.org; mobile later 07700 900777 → 900778) at A1 and B1;
  - Kim QZ and Lee QZ (both 04/05/2016, 07700 900501) at A1;
  - Fay FreshQZ (09/09/1990, fresh.qz@) at A1.
- Order note: T-18 was reached naturally partway through. Ten messages from this machine had used up the hourly network limit by 15:49.
  - Re-running `make seed` to reset the counts was refused by the permission system.
  - So T-18 was tested at 15:50, and T-15 and T-16 (which send messages) waited until the hour had passed (16:41).
  - T-06 used a link opened 15 min 9 s after it was asked for (just after its 15-minute expiry), not 16 minutes.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 One home lists both trusts | Pass | `/home`: "Hello Sam", B1 (Hip replacement) above A1 (Knee replacement), both To do / Start. Contacts show `•••501` and Email "Not given". The `/api/p/home` JSON has no DOB, surname, NHS number or record ids. No sideways scroll (scrollWidth 388 = clientWidth). |
| T-02 Open from home in either trust | Pass | B1 bar shows "Hospital B1". After an answer: In progress / Continue, and the answer is kept on reopen. A1 opens unanswered (B's answer doesn't leak), then the same behaviour. |
| T-03 Returning patient by email | Pass | `  RETURN.QZ@EXAMPLE.ORG  ` → "Check your email" (if-used wording, once, 15 minutes, Send another link). Email "Your Poised sign-in link" has no name. `/s/` page shows "Sign in to Poised" + Continue only, then "Poised needs your date of birth" → `/home`. |
| T-04 Returning patient by text | Pass | "Check your text messages". SMS text as specified → Continue → 17 5 1980 → both trusts. |
| T-05 Sign-in link works once | Pass | Second use → "This link has expired …" with Get a new sign-in link (→ `/signin`). The API's `/dob` on the used token also gives 404 `expired`. The first session still works. |
| T-06 Expired link refused | Pass | Link from 15:34:31 opened at 15:49:40 → "This link has expired". DB confirmed it was unused and not revoked, so expiry alone refused it. |
| T-07 Wrong DOBs on sign-in link | Pass | 4, 3, 2, 1 tries left, then "This link is locked … ask for a new sign-in link". The right date is then 403 `locked` (API). A new link (T-14's) worked. |
| T-08 Rate limit per contact | Pass | 4th request → exact red message, and no 4th email. A different contact (07700 900501) still worked. An unknown address gets the same 429 on its 4th request (checked through the API), so the limit doesn't reveal accounts. |
| T-09 Mobile outside the country | Pass | +1 202 555 0143 → the "in this country" message, nothing captured. `hello` / `123` → "Enter a real mobile number or email address." `a@b` → "Enter a real email address…". Empty → "Enter your mobile number or email address." |
| T-10 Unknown contact reveals nothing | Pass | nobody.qz@ → same "Check your email", nothing sent. fresh.qz@ (invite never opened) → same screen with the grey invite hint, nothing sent. |
| T-11 Sign out | Pass | "You've been signed out" + Get a sign-in link. Reload `/home` and `/home/episodes/<own id>` stay signed out. |
| T-12 Another patient's episode address | Pass | ReturnQZ opening Sam HomeQZ's A1 episode → "Episode not found." inside the bar with ← Your questionnaires. GET, PUT answers and POST submit on that id all 404. `/home/episodes/abc` gives the same. |
| T-13 Shared mobile stays apart | Pass | Kim (4 5 2016) sees only Kim's questionnaire. Sign-in by 07700 900501 with 17 5 1980 → Sam's home (both trusts). A fresh link with 4 5 2016 → Kim's home. Kim's session gets 404 on Sam's episodes. |
| T-14 Record joins at sign-in | Pass | Bea's later B1 episode for ReturnQZ was not on the home until sign-in by email. After sign-in, B1 (Cataract) is above A1. Audit `account.join` row in Trust B. |
| T-15 Change email (PAT-10) | Pass | Green note; reload still `r•••@example.org`. Email "Confirm your new email address for Poised" (24 h). Confirm in another window → "Your email address has been updated" → home `n•••@example.org`. Old address told. A1 and B1 records both have the new email (seen in Bea's B1 episode). General notes unchanged. One `contact.update` audit row per trust. Second Confirm → "This link has expired". `/signin` with the new email sends; with the old one, nothing. GIF recorded. |
| T-16 Change/add mobile; refusals | Pass | 07700 900778 → text with `/v/` → Confirm → "Your mobile number has been updated", home `•••778`, and the old mobile (777) told. Refusals: current email "That's already your email address.", current mobile "That's already your mobile number.", `hello` "Enter a real…", +1 202… "in this country". Cancel closes the box. |
| T-17 Reload, deep links, Back | Pass | Reload `/home` keeps the session. `/home` in a new tab of the same profile works. Back from a questionnaire → home. A window whose session had ended → signed-out card. After the `/s/` DOB, reload stays on `/home` and Back skips the `/s/` form. |
| T-18 Rate limit per network | Pass | 11th request (ip1.qz@) → "There have been too many requests from this network. Please try again in an hour.", nothing captured. A contact change from the home got the same message. |
| R-1 (S14) New invite → send | Pass | Continue shows only "Hospital B1". A wrong date → 4 tries left; the right one opens. Both sections done → in-app "Send your answers?" modal → "Thank you, Sam". Home: B1 **Sent** / **View**. |
| R-2 (S14) Locked invite + Resend | Pass | Five wrong dates → locked. Cara sees "The link is locked" + Resend invite. The old link → "This link doesn't work"; the new one opens. |
| R-3 (S14) Fill in for the patient | Pass | "Patient's answers entered by Cara Clinician" on About you. |
| R-4 (S13) Search scoped to trust | Pass | Cara at A1 searching ReturnQZ finds one record (Trust A's), not Trust B's. |
| R-5 (S11) Audit "Patient opened their home" | Pass | Rows present in Trust A's log (Ada Admin). Trust B's join and contact rows sit under Trust B in `audit_log`. |
| R-6 (S10) Suspend Trust B | **Blocked** | The permission system refused confirming "Suspend trust" (shared environment). Dialog cancelled; Trust B left Active. **Hand check for Rahul.** |

## Failures

None in the planned cases.

## Found while exploring

1. **Major (privacy; needs a product decision): people sharing a contact *and* a date of birth (e.g. twins) are merged into one account.**
   - Steps: Cara creates Lee QZ, 04/05/2016, mobile 07700 900501 (Kim QZ's twin: same mobile, same DOB). Lee's invite is opened with 4 5 2016.
   - Result: the home says **"Hello Lee"** and lists both Lee's Grommets and **Kim's Tonsillectomy** questionnaires. The cards don't say whose each one is, and either twin can open, fill in and send the other's.
   - This follows from the builder's matching rule (same DOB + same contact). The plan only covers different DOBs.
   - Options: also match on first name, or show the patient's first name on each card until PAT-08 (carers).
2. **Minor (security hardening): adding a contact tells nobody.**
   - Adding a mobile to an account that had only an email sent no notice to the email.
   - The build notes' reason for telling the old contact ("a session left open on a shared phone could move the account … unnoticed") applies just as much to an *add*. Someone with a briefly open session could add their own mobile and keep signing in, and the owner would never know.
   - Suggest telling the existing contact(s) on an add too.
3. **Minor: each contact box accepts the other kind of contact.**
   - In **New email address**, typing `07700900777` was accepted. A text went to that mobile, and the green note said "Until then, we'll keep using your current **email address**". Confirming then changed the **mobile**.
   - The mobile box likewise accepts an email.
   - `hello` in the email box says "Enter a real mobile number or email address."
   - Suggest checking the input against the box it was typed in.
4. **Minor: a record that joins later doesn't get the account's checked mobile.**
   - The Trust B ReturnQZ record (email only) joined after the patient had confirmed mobile 777. It kept no mobile until the next change, which then went to every joined record.
   - So B1 can't text the patient even though the account has a confirmed mobile. This may be intended ("never replaces"), but a missing value isn't a replacement.
5. **Minor (audit):** when Lee's invite DOB joined Lee's record to Kim's existing account, only a `link.dob` row was written, with no `account.join` row. The `account.join` rows seen were for records joined through sign-in links, or other trusts' records joined at an invite.
6. **Cosmetic:**
   - A verification link replaced by a newer request shows "This link doesn't work. Please check it, or contact the hospital." "This link has expired …" would be clearer.
   - The "doesn't work" and expired verification cards have no button onward (the sign-in cards do).
7. **Cosmetic/product:** **View** on a Sent questionnaire shows only "Thank you, Sam", not the answers.
8. **Minor:** `😀😀@example.org` and an email with a 300-character local part are accepted as valid emails (they passed validation and reached the rate limit).
9. **Info:**
   - The patient cookie is `poised_patient; Path=/api/p; Max-Age=1800; HttpOnly; SameSite=Strict`, with no `Secure` (expected on local http; S06 should add it).
   - Sessions do end after ~30 min idle: ReturnQZ's home showed the signed-out card at 16:40.
10. **Info (test plan): one tester following the plan uses up the 10-per-hour network limit by about T-14.**
    - Ten messages had been asked for by 15:49, and the plan has about 13 message requests before T-18.
    - Running `make seed` between T-14 and T-15 would need to be stated in the plan for an automatic tester, or the per-network limit made configurable for local dev.

## Console and network errors

- No console errors were seen on the staff tab.
- Iframe consoles weren't captured by the extension.
- The only 4xx responses were the expected ones (404 expired/unknown, 403 locked, 401 wrong DOB, 400 invalid, 429 limits).
- The staff audit log shows every read twice (e.g. "Opened an episode" ×2). This looks like React StrictMode double fetching in dev and predates S15.

## GIFs

Downloaded by Chrome to `~/Downloads/`:
- `S15-T01-T02-home.gif`: invite → home with both trusts → B1 answered → In progress.
- `S15-T15-change-email.gif`: change email → green note → confirm elsewhere → home shows the new email.

## Hand checks for Rahul

- **R-6:**
  1. As Sam Superadmin, `/admin/trusts/<Trust B>` → **Suspend trust** → Suspend.
  2. Sam HomeQZ's home (07700 900501, 17 5 1980) should list only Hospital A1, and the B1 episode address should refuse.
  3. Re-activate Trust B afterwards.
