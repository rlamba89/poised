# S15 test report 2 (retest)

**Summary: 27 passed, 0 failed, 1 blocked.** That is 22 plan cases (T-01 to T-22) and 6 regression cases. R-6 was skipped as a hand check for Rahul.

- **Environment:**
  - HEAD `08a3be0`. The working tree is not clean: S15's uncommitted build plus the planning-doc changes.
  - 13 Oct 2026, 17:41–18:10 BST.
  - `make dev` was started by the conductor (API :8080, web :3000). `/api/health` was ok.
- **Browser:** Claude in Chrome.
  - Staff pages ran at desktop size on `localhost:3000` (Cara, then Ada) and `bea.localhost:3000` (Bea).
  - Patient pages ran in a 390×844 iframe ("phone") on `tester.localhost:3000` and `phone2.localhost:3000`.
- **Test data:** all made up, initials `RT`.
  - Sam HomeRT: 17/05/1980, 07700 900521, at A1 and B1.
  - Sam ReturnRT: 03/04/1975.
    - Email: return.rt@ → new.rt@example.org.
    - Mobile: added 07700 900881, then changed to 900882.
    - Records at A1 and B1.
  - Twins Kim TwinRT and Lee TwinRT: 04/05/2016, 07700 900621, at A1 and B1. Later they also shared parent.rt@example.org.
  - Twins Ann TwinRT2 and Bob TwinRT2: 07/06/2015, 07700 900631, at A1 and B1.
  - Kim OtherRT: 04/05/2016, 07700 900621. Same first name as Kim TwinRT, different surname.
  - On 07700 900521: Sammy HomeRT (17/05/1980), " sam " CaseRT (17/05/1980) and Kim ShareRT (04/05/2016).
  - Pat FillRT: 02/02/1985, fill.rt@ plus 07700 900883, at A1 and B1.
  - Fay FreshRT: fresh.rt@, invite never opened.
- **How the cases were run:**
  - The main flows were done in the UI at phone size: the invite, the home, the twins' first-name step, sign-in by email and by text, the contact boxes, confirming a contact, sending answers, and Resend invite.
  - To save time, these went through the same `/api/...` calls the pages make, from the browser, and are marked "(API)" below:
    - creating most extra episodes;
    - repeated sign-ins for the probes;
    - the batch of invalid inputs for T-09/T-10;
    - the ten T-18 requests.
  - Captured messages were read through `GET /api/qa/messages`, the data behind `/qa/messages`.
  - Database checks were read-only selects.
- **Reset sign-in limits:** pressed on `/qa/messages` at the start, before T-18 and after T-18.
  - The probes used extra requests, so it was also pressed twice mid-run (about 17:56 and 17:58), plus one `DELETE /api/qa/rate-limits` (about 17:52), the same call the button makes.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 One home lists both trusts | Pass | UI. "Hello Sam". B1 (Procedure: Hip replacement) is above A1 (Procedure: Knee replacement). Both cards say **For Sam HomeRT**, To do and Start. Mobile `•••521`, Email "Not given" with **Add**. No date of birth or NHS number. scrollWidth 390 = clientWidth. Two invites captured, one from A1 and one from B1. |
| T-02 Open from the home in either trust | Pass | The B1 bar shows "Hospital B1". After one answer the card shows In progress / Continue. |
| T-03 Returning patient by email | Pass | `  RETURN.RT@EXAMPLE.ORG  ` gives "Check your email" with the if-used wording, once, 15 minutes and Send another link. `/s/` shows "Sign in to Poised" and Continue only, then the date of birth, then `/home`. |
| T-04 Returning patient by text | Pass | UI. "Check your text messages". The SMS text matches the plan. The home shows both trusts (plus the probe record, see Found while exploring 1). |
| T-05 Sign-in link works once | Pass | Opening it again in another window gives "This link has expired", the exact text, and Get a new sign-in link. |
| T-06 Expired sign-in link | Pass | The link was asked for at 17:51:21 and opened at 18:07:35: "This link has expired". The database confirms it was unused and not revoked. |
| T-07 Wrong dates on a sign-in link | Pass | UI. 4…1 tries left, then "This link is locked …" with Get a new sign-in link. The right date is then refused (API 403). A new link worked (T-14). |
| T-08 Rate limit per contact | Pass | UI. The 4th request shows the exact red message and sends nothing. Another contact (07700 900521) still gets 202. |
| T-09 Mobile outside the country, invalid input | Pass | (API, same messages as the page.) `+1 202 555 0143` gets the "in this country" message. `hello`/`123`: "Enter a real mobile number or email address." `a@b`: "Enter a real email address…". Empty: "Enter your mobile number or email address." Nothing captured. |
| T-10 Unknown contact reveals nothing | Pass | `nobody.rt@` and `fresh.rt@` (never opened) both get 202 and nothing is sent. `😀😀@example.org` and a 70-letter local part are now refused ("Enter a real email address…"), and so are `ü` addresses. Staff typing `😀😀@example.org` on New episode gets "Enter a valid email address, or leave it empty." |
| T-11 Sign out | Pass | UI. "You've been signed out" + Get a sign-in link. Reloading `/home/episodes/<own id>` stays signed out. |
| T-12 Another patient's episode address | Pass | Kim's session on Sam HomeRT's A1 episode shows "Episode not found." inside the bar with ← Your questionnaires. GET, PUT answers and POST submit all 404. `/home/episodes/abc` shows the same. |
| T-13 Shared mobile, different dates | Pass | Kim ShareRT (2016) on Sam's mobile sees only her own card. Sign-in with 17 5 1980 gives Sam's home; 4 5 2016 gives Kim's home. |
| T-14 Record joins at sign-in | Pass | UI. Bea's later B1 record for ReturnRT appears after sign-in by email (B1 above A1). An `account.join` row is written in Trust B. |
| T-15 Change email (PAT-10) | Pass | UI. Green note. After a reload the home still shows `r•••`. "Confirm your new email address for Poised", then Confirm in another window: "Your email address has been updated", and the home shows `n•••@example.org`. The old address got "…were changed", and the mobile got an SMS too. The A1 and B1 records both have new.rt@ (Bea's page). General notes are unchanged. Ada's Audit shows "Patient changed their contact details". A second Confirm gives "This link has expired". Sign-in: the new address sends; the old address sends nothing. |
| T-16 Change/add mobile; refusals | Pass | UI. 881 → 882: Confirm, `•••882`, and the old mobile and the email are told. Refusals: "That's already your mobile number." / "That's already your email address." (typed `RETURN.RT@example.org`), `hello`, and `+1 202…` "in this country". **Cancel** closes the box. |
| T-17 Reload, deep links, Back | Pass | Reload keeps the session. `/home` in a new tab works. Back from a questionnaire returns to `/home`. A browser with no patient session shows the signed-out card. After the `/s/` date of birth, reload stays on `/home` and Back skips the `/s/` form. |
| T-18 Rate limit per network | Pass | 10 addresses (API), then the 11th in the UI: "There have been too many requests from this network. Please try again in an hour.", and nothing captured. A contact change from a home got the same 429. After **Reset sign-in limits**: "Check your email". |
| **T-19 Twins on one phone** | Pass | UI. Kim's invite gives "Hello Kim" with only her card ("For Kim TwinRT"); Lee's gives only Lee's. On a shared-mobile sign-in, the date of birth gives the red "More than one person… Please also enter your first name." and a **First name** box. `pat` gives "…don't match… 4 tries left". ` LEE ` gives Lee's home only. Sammy HomeRT (same date and mobile as Sam) stays out of Sam's home. GIF recorded. |
| **T-20 Adding a contact tells the existing one** | Pass | UI. Add mobile 07700 900881, Confirm: "Your mobile number has been updated". The email got "Your Poised contact details were added" ("The mobile number you use for Poised was added…"). |
| **T-21 Each box takes only its own kind** | Pass | UI. `07700 900555` in the email box: "Enter an email address, for example name@example.com." `someone@example.org` in the mobile box: "Enter a mobile number." Nothing captured for either. |
| **T-22 A later-joined record gets the account's contact** | Pass | Run with a fresh patient (see Found while exploring 8). Pat FillRT (A1, email only) adds mobile 883. Bea then creates a B1 episode in the UI with the email only. After sign-in by email, the B1 card is listed and Bea's page shows Mobile `+447700900883`, email unchanged. Trust B `account.join` row (database). |
| R-1 (S14) New invite → send | Pass | Wrong date: "4 tries left" (Sammy's invite, API). Pat's A1 questionnaire filled in at phone size, in-app "Send your answers?", "Thank you, Pat". The home shows **Sent** / **View**. |
| R-2 (S14) Locked invite + Resend | Pass | Five wrong dates lock Bob's invite. Cara sees "The link is locked … 5 times" and Resend invite (no browser dialog). The old link: "This link doesn't work". The new one opens Bob's account (A1 + B1 Bob, not Ann's). |
| R-3 (S14) Fill in for the patient | Pass | "Patient's answers entered by Cara Clinician" on About you. |
| R-4 (S13) Search scoped to trust | Pass | Cara at A1 searching "FillRT" finds one record, not Trust B's. |
| R-5 (S11) Audit | Pass | Ada's Trust A log has "Patient opened their home", "Patient's record joined their sign-in account" (now for the invited record too) and "Patient changed their contact details". Trust B rows are only in Trust B (database). |
| R-6 (S10) Suspend Trust B | **Blocked** | Hand check for Rahul (as the plan says). |

## Failures

None.

## Found while exploring

1. **Major privacy risk, needs a product decision: two people with the same first name, date of birth and contact but different surnames become one account.** This is what the agreed rule does (DOB + contact + first name), but it's how one person can see another's questionnaire.
   - **Steps:**
     1. Cara creates **Kim OtherRT** (04/05/2016, 07700 900621), alongside the existing Kim TwinRT account.
     2. Kim TwinRT signs in.
     3. Her home lists "For Kim OtherRT / A1 Kim Other op" next to her own two, and she can open it (200).
     4. The other way round: **Kim OtherRT's own invite** (her A1 link, her date of birth) opens Kim TwinRT's home, including Kim TwinRT's A1 and B1 questionnaires.
   - **The same on 07700 900521:** " sam " CaseRT joined Sam HomeRT's account. The home then greets "Hello **sam**" (see item 2).
   - **Why it matters:** in the data these are two different patients, so a shared family phone where cousins share a birthday and a first name is the risk. Each card says whose it is, which helps.
   - **Options:**
     - also require the surname to match (a married-name change would then stay unjoined, which fails safe, as answer 6 chose for spellings);
     - or keep the rule as agreed.
   - Not graded as a failure, because the plan and answer 6 define the match as first name only.
2. **Cosmetic:** the home's greeting takes its casing from the newest record. After " sam " CaseRT joined, `GET /api/p/home` says `firstName: "sam"`, and the home reads "Hello sam" for Sam HomeRT.
3. **Cosmetic:** adding a mobile where there was none shows a green note saying "Until then, we'll keep using your current mobile number", but there is no current mobile.
4. **Cosmetic: the first-name step's wording.**
   - The wrong-try message says "That date of birth and first name don't match… Please check **it**" (two fields).
   - After 5 wrong first names, the lock says "because the **date of birth** was entered wrongly".
   - Focus stays in the Year box when the First name box appears.
5. **Cosmetic:** after **Sign out** from the home, the card reads "You've signed out." then "You can also sign in…". The "To come back, open the link…" sentence that the questionnaire's sign-out shows is missing, so "also" reads oddly.
6. **Cosmetic:** a sign-in link sent to an email that the account changed away from afterwards is correctly refused. But it says "That date of birth doesn't match our records… 4 tries left" to someone who typed the right date. "This link has expired" would be clearer.
7. **Minor:** a 300-character first name (API) gets a generic 400 "The request could not be read." The UI box has no length limit, so a very long paste would show that message.
8. **Info (test plan):** T-22 as written passes trivially if T-14 was done first.
   - Bea's "new patient" with the same details is attached to Trust B's existing, already-joined record, which already has the mobile.
   - The fill was proven with a fresh patient (Pat FillRT) instead. The plan could use a patient with no Trust B record.
9. **Info, twins and shared contacts work as intended:**
   - Both twins can confirm the same email, parent.rt@ (no 409). A sign-in by that email then asks for the first name too.
   - `k i m` matches Kim.
   - A link reaching only one account doesn't ask for a first name. Bob's unopened records are never swept into Ann's account, at the invite or at sign-in.
   - The 409 "first name" answer doesn't use up a try; a wrong first name does; after 5 wrong tries the link locks and even the right name is refused.
10. **Info:** invite SMSs don't name the patient, so on a shared phone the twins' parent can't tell which invite is whose until after the date of birth. The home then says whose it is. This is a privacy trade-off, noted only.
11. **Info:** emails with non-ASCII characters (`ok.ünï@…`, `x@exämple.org`) are refused. That matches the "plain ASCII only" decision, but may matter for other countries later.
12. **Info:** double-clicking **Send me a link** sends one request.

## Console and network errors

- No console errors or warnings on any tab.
- The only 4xx responses were the expected ones: 400, 401, 403, 404, 409 and 429.
- The extension's network log showed **503** for `POST /api/dev/login` and `DELETE /api/qa/rate-limits`, even though both worked. Curl got 204 from the reset, both on :8080 and through :3000. This looks like a logging artefact of the extension, not the app.

## GIFs

- `~/Downloads/S15-T19-twins-signin.gif`: the twins' shared-mobile sign-in, from the date of birth to the First name box, a wrong name (`pat`), then ` LEE `, ending on Lee's home only.

## Hand checks for Rahul

- **R-6:**
  1. As Sam Superadmin, `/admin/trusts/<Trust B>` → **Suspend trust**.
  2. Sam HomeRT's home (07700 900521, 17 5 1980) should list only the Hospital A1 cards, and B1 episode addresses should be refused.
  3. Re-activate Trust B afterwards.
