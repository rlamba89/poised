# S15 test report 3 (retest, round 3)

**Summary: 28 passed, 0 failed, 1 blocked.** That is 23 plan cases (T-01 to T-23) and 6 regression cases. R-6 was skipped as a hand check for Rahul.

- **Environment:**
  - HEAD `08a3be0`. The working tree is not clean: S15's uncommitted build plus the planning-doc changes.
  - 13 Oct 2026, 18:23–18:52 BST.
  - `make dev` was started by the conductor (API :8080, web :3000). `/api/health` was ok.
- **Browser:** Claude in Chrome.
  - Staff pages ran at desktop size: Cara (then Ada) on `localhost:3000`, Bea on `bea.localhost:3000`.
  - Patient pages ran in a 390×844 iframe ("phone") on `tester.localhost:3000`.
- **Test data:** all made up, initials `R3`. Fresh mobiles and emails were used, so earlier rounds' data didn't interfere.
  - Sam HomeR3: 17/05/1980, 07700 900541, at A1 and B1. Later moved his own mobile to 900542.
  - Sammy HomeR3: 17/05/1980, 900541, at A1.
  - Kim ShareR3: 04/05/2016, 900541, at A1.
  - Sam ReturnR3: 03/04/1975, at A1 and B1.
    - Email: return.r3@ → new.r3@example.org.
    - Mobile: added 900888, then changed to 900887.
  - Twins Kim TwinR3 and Lee TwinR3: 04/05/2016, 900641. Both later added parent.r3@example.org. Kim also has a B1 record (`kim` / `twin r3`).
  - Kim AlphaR3 and Kim BetaR3: 06/07/2015, 900651. Alpha also has a B1 record (` kim ` / `ALPHAR3`). Beta later moved her own mobile to 900652.
  - Pat FillR3: 01/02/1985, fill.r3@, mobile 900893 added. Records at A1 and B1.
  - Jo DupR3: 01/01/2000, A1 with mobile 900671 and B1 with dup.r3@. These are two accounts, used for the collision probe.
- **How the cases were run:**
  - The main flows were done in the UI at phone size:
    - invites;
    - the home;
    - twins and same-first-name sign-ins through the name step;
    - sign-in by email and by text;
    - wrong dates and the lock;
    - the per-contact and per-network limits;
    - the contact boxes and confirming a change;
    - sending answers.
  - Staff: T-01's A1 episode and T-22's B1 episode were created in the New episode UI. The other episodes were created with the same `POST …/episodes` call the page makes, from Cara's or Bea's tab, marked "(API)" below.
  - The probes and some repeat sign-ins went through the same `/api/p/...` calls the pages make, from the phone tab, also marked "(API)".
  - Captured messages were read with `GET /api/qa/messages`, the data behind `/qa/messages`.
  - Database checks were read-only selects.
- **Reset sign-in limits:**
  - Pressed on `/qa/messages` at the start, before T-17's sign-in (about 18:42), before T-18 and after T-18.
  - The probes needed extra requests, so `DELETE /api/qa/rate-limits` was also called three times between 18:35 and 18:40. That is the same call the button makes.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 One home lists both trusts | Pass | UI. A1 episode created in the New episode UI. "Hello Sam". Hospital B1 (Procedure: Hip replacement) is above Hospital A1 (Procedure: Knee replacement). Both cards say **For Sam HomeR3**, To do and Start. Mobile `•••541`, Email "Not given" with **Add**. No date of birth or NHS number. scrollWidth 390 = clientWidth. Two invites captured, one from A1 and one from B1. |
| T-02 Open from the home in either trust | Pass | UI. B1 opens with "Hospital B1" in the bar. After one answer ("Saved") the card shows **In progress** / **Continue**. The A1 side was covered by R-1 (Pat). |
| T-03 Returning patient by email | Pass | UI. `  RETURN.R3@EXAMPLE.ORG  ` gives "Check your email" with the "if you use Poised" wording, once / 15 minutes, the grey first-timer line and Send another link. The email is "Your Poised sign-in link". `/s/` shows "Sign in to Poised" and Continue only, then "Confirm it's you … Poised needs your date of birth", then `/home` with Hello Sam. |
| T-04 Returning patient by text | Pass | UI. "Check your text messages". The SMS reads "Poised: your sign-in link. It works once, within 15 minutes: …". The home shows both trusts and not Sammy's questionnaire. |
| T-05 Sign-in link works once | Pass | UI. Opening it again gives "This link has expired" with the exact text and Get a new sign-in link. The first session still works. |
| T-06 Expired sign-in link | Pass | UI. The link was asked for at 18:34:04 and opened at 18:50:30: Continue gives "This link has expired" with the exact text (API 404). The database shows it unused, unlocked and not revoked, so expiry is the reason. |
| T-07 Wrong dates on a sign-in link | Pass | UI. "That date of birth doesn't match… You have 4 tries left", down to 1. The 5th gives "This link is locked because the details were entered wrongly too many times. Please ask for a new sign-in link." with Get a new sign-in link. The right date is then refused (API 403). New links worked afterwards (T-14). |
| T-08 Rate limit per contact | Pass | UI. The 4th request gives the exact red message. Captured messages stay at 4 → 4. Another contact (07700 900641) still gets 202. |
| T-09 Mobile outside the country, invalid input | Pass | (API, the page's call.) `+1 202 555 0143` gets the "in this country" message. `hello`/`123`: "Enter a real mobile number or email address." `a@b`: "Enter a real email address…". Empty: "Enter your mobile number or email address." Nothing captured. |
| T-10 Unknown contact reveals nothing | Pass | (API.) `nobody.r3@` gives 202 and nothing is sent. `😀😀@example.org` and a 70-letter local part get "Enter a real email address…". `fill.r3@`, a record whose invite was never opened, gives 202 and nothing is sent. The grey line on the page says to use the hospital's link. |
| T-11 Sign out | Pass | UI. The home's Sign out shows "You've been signed out" / "You've signed out. To come back, open the link in your text message or email again." / "You can also sign in…" / Get a sign-in link. `GET /api/p/home` then returns 401. A fresh `/home` shows the signed-out card. |
| T-12 Another patient's episode address | Pass | UI. Kim ShareR3's session on Sam HomeR3's A1 episode shows "Episode not found." inside the bar with ← Your questionnaires. `/home/episodes/abc` shows the same. API: GET 404 for Sam's A1 and B1 episodes. Kim BetaR3's session gets GET/PUT/submit 404 on Kim AlphaR3's and Kim TwinR3's episodes. |
| T-13 Shared mobile, different dates | Pass | UI. Kim ShareR3 (2016, on Sam's 900541) sees only her own card ("Hello Kim"). Sign-in with 900541 + 17 5 1980 gives Sam's home (both trusts). |
| T-14 Record joins at sign-in | Pass | Bea's later B1 record for Sam ReturnR3 (API) appears after sign-in by email (API): B1 above A1. A Trust B `account.join` row was written (database). |
| T-15 Change email (PAT-10) | Pass | UI. The box shows; the green "We've sent a link to new.r3@example.org. Open it to finish the change. Until then, we'll keep using your current email address." The home still shows `r•••`. Then the "Confirm your new email address for Poised" email, **Confirm**, "Your email address has been updated", and the home shows `n•••@example.org`. The old address got "Your Poised contact details were changed". Cara (A1) and Bea (B1) both see new.r3@example.org, and General notes are unchanged. `contact.update` rows in Trust A and Trust B; Ada's Audit page shows "Patient changed their contact details". A second Confirm gives 404 "expired". Sign-in: the new address sends; the old one sends nothing. |
| T-16 Change/add mobile; refusals | Pass | UI. 888 → 887 (900887 used instead of 900777, which earlier rounds use): Confirm, `•••887`. The old mobile got "…was changed", and so did the email. `hello`: red message. `+1 202 555 0143`: the "in this country" message. Email = current (`NEW.R3@example.org`): "That's already your email address." Mobile = current: "That's already your mobile number." **Cancel** closes the box. |
| T-17 Reload, deep links, Back | Pass | Reloading `/home` keeps the session and the list. Back from a questionnaire returns to `/home`. No session: the signed-out card. After the name step on `/s/…`, Back stays on `/home`. A new tab in the same browser shares the same cookie (not opened separately). |
| T-18 Rate limit per network | Pass | 10 addresses (API), then the 11th in the UI: "There have been too many requests from this network. Please try again in an hour." Nothing captured. A contact change also gets the 429. **Reset sign-in limits** shows "Sign-in limits cleared: links can be asked for again.", and the next request gives "Check your email". |
| **T-19 Twins on one phone** | Pass | UI. Kim's invite gives "Hello Kim", one card "For Kim TwinR3". Lee's gives only Lee's. Shared-mobile sign-in shows the exact red message ("…Please also enter your first name and surname.") with **First name** (focused, checked via `activeElement`) and **Surname**. `Lee`/`Someone` gives "That date of birth and name don't match our records. Please check them and try again. You have 4 tries left." ` LEE `/`TWINR3` gives Lee's home only. Both boxes have maxlength 100; a typed 101-character name stopped at 100. Sammy HomeR3 never appears in Sam's home. GIF recorded. |
| **T-20 Adding a contact tells the existing one** | Pass | UI. Sam ReturnR3 adds 07700 900888: "We've sent a link to 07700 900888. Open it to finish adding your mobile number." Then Confirm and "Your mobile number has been updated". The email got "Your Poised contact details were added" ("The mobile number you use for Poised was added…"). |
| **T-21 Each box takes only its own kind** | Pass | UI and API. `someone@example.org` in the mobile box: "Enter a mobile number." `07700 900555` in the email box: "Enter an email address, for example name@example.com." Nothing captured. |
| **T-22 A later-joined record gets the account's contact** | Pass | Pat FillR3 (A1, email only) opens the invite and adds and confirms 900893 (API). Bea then creates the B1 episode **in the UI** with only the email. After sign-in by email (API) the home lists B1 and A1. Bea's episode page shows Mobile `+447700900893`, email unchanged. A Trust B `account.join` row exists (database). |
| **T-23 Same first name, birthday and phone, different surnames** | Pass | Steps 2–4 and 6 in the UI, the step 1 and 5 episodes via API. Kim AlphaR3's invite shows only "For Kim AlphaR3"; Kim BetaR3's only hers. Sign-in on 900651 + 6 7 2015 gives the name step. `Kim`/`betar3` gives Kim Beta's home only. Bea's ` kim `/`ALPHAR3` record joins Alpha at her named sign-in: the home lists B1 ("For kim ALPHAR3") and A1, and greets **Hello Kim**. Database: the B1 record is on Alpha's account, Beta's account is untouched. Step 6: a link for return.r3@ was asked for, then the email was changed to new.r3@ and confirmed. The first link, with the right date, then gives "This link has expired" (UI). |
| R-1 (S14) New invite → send | Pass | A wrong date gives "4 tries left" (Jo's invite, API). Pat's A1 questionnaire was filled at phone size: in-app "Send your answers?" → "Thank you, Pat", and the home shows **Sent** / **View**. |
| R-2 (S14) Locked invite + Resend | Pass | Five wrong dates lock Jo DupR3's A1 invite (API). Cara's page shows "The link is locked" and Resend invite, and there is no browser dialog. The old link gives "This link doesn't work…". The new one opens Jo's A1 episode. |
| R-3 (S14) Fill in for the patient | Pass | Cara answered one question; the episode shows "Patient's answers entered by Cara Clinician". |
| R-4 (S13) Search scoped to trust | Pass | (API, the page's call.) Cara at A1 searching "FillR3" gets 1 record (A1's), not Trust B's. |
| R-5 (S11) Audit | Pass | Ada's Trust A log has "Patient opened their home", "Patient's record joined their sign-in account" and "Patient changed their contact details". The Trust B record "kim ALPHAR3" and "Hospital B1" don't appear in it. The Trust B rows are in Trust B (database). |
| R-6 (S10) Suspend Trust B | **Blocked** | Hand check for Rahul, as the plan says. |

## Failures

None.

## Probes on the new matching rule (all behaved correctly)

- **Same first name, date of birth and phone, different surname** (Kim AlphaR3 / Kim BetaR3): two accounts, both from the invite and at sign-in. Neither ever saw the other's card. Bea's B1 record joined only Alpha.
- **Same full name, odd case and spacing joins:**
  - ` kim `/`ALPHAR3` joined Kim AlphaR3.
  - `kim`/`twin r3` (Trust B, parent.r3@ only) joined Kim TwinR3 at its invite and not Lee.
  - At sign-in, `KIM`/`Beta R3` and `sam`/` home r3 ` are accepted.
- **Twins** (Kim/Lee TwinR3) on one phone and then one shared email:
  - Both can confirm parent.r3@.
  - A sign-in by that email asks for the name ("…uses this email address…").
  - Lee's name gives Lee's home only.
- **A different first name with the same surname, date of birth and phone** (Sammy vs Sam HomeR3):
  - Before Sammy opened her invite, Sam's sign-in had no name step and Sammy's card never showed.
  - Afterwards, the name step appears. `Sam`/`Home` (wrong surname) is a try ("…don't match… 4 tries left"). `sam`/` home r3 ` gives Sam only.
- **The name step's rules (API):**
  - The 409 "name" answer can repeat without using a try.
  - A first name alone gives 400 "Enter your first name and your surname." (also shown in the UI).
  - A 101-character name gives 400 "That name is too long…", not a try.
  - The right name with a wrong date is a try with the name wording.
  - Five wrong names lock the link, and even the right name is then refused (403 locked).
- **Contact changes on shared contacts:**
  - Kim BetaR3 moved from the shared 900651 to 900652. Only Beta's record and account changed; Alpha's two records stayed on 900651. Sign-in on 900651 then goes straight to Alpha (no name step).
  - Sam HomeR3 moved from 900541 to 900542. Sammy and Kim ShareR3 stayed on 900541. A 900541 sign-in with 17 5 1980 then goes straight to Sammy's home, with no name step and no Sam card.
- **A contact change that would collide with the same person's other account** (Jo DupR3 has two accounts: A1 by mobile, B1 by email):
  - Adding `DUP.R3@example.org` from the A1 account is accepted at first (202) and sends a confirmation email.
  - **Confirm** is refused: 409, and the page says "We couldn't change your details — This email address is already used by another account with your details. Please contact your hospital."
  - Nothing changed (database). See Found while exploring 1.
- **No leaks between people:** in every run of this round, a home listed only records with the signed-in person's full name and date of birth (also checked in `patients.account_id`).

## Found while exploring

1. **Minor: a change that will be refused is only refused after the patient opens the confirmation link.**
   - **What happens:**
     - `POST /api/p/contact` answers 202, and the home shows the green "We've sent a link…" note.
     - A "Confirm your new email address" email is sent to the address.
     - Only **Confirm** then says it's "already used by another account with your details".
   - **Why it matters:** the patient takes an extra step and gets a message that's wasted.
   - **Possible fix:** refuse at the request with the same message. The collision is the same person (same full name and date of birth) holding two accounts, so this is rare.
   - **Product note:** such a person keeps two homes, and nothing on their side can join them. The message tells them to contact the hospital, but no staff tool joins accounts yet.
2. **Cosmetic: two refusals in the mobile box mention email.**
   - In the mobile box, `hello` gives "Enter a real mobile number or email address." The box only takes a mobile; `someone@example.org` correctly gives "Enter a mobile number."
   - A US number in the mobile box gives "…Please use your email address instead." That is reasonable, but odd when the patient is in the mobile box and already has an email.
3. **Cosmetic:** **Change** / **Add** opens the contact box without putting the cursor in it (`activeElement` stays on the body), so keyboard users have to Tab into it. The name step does focus First name.
4. **Info: the per-contact limit (3 in 15 minutes) counts verification messages too.**
   - Kim and Lee each confirmed parent.r3@, which sent two confirmation emails, then one sign-in link went to that address.
   - The next sign-in request was refused: "You've asked for 3 links in the last 15 minutes".
   - Arguably fine, since it is a limit on messages to that inbox, but the wording says "asked for 3 links".
5. **Info:** on a shared phone, the "contact details were changed/added" notices go to the shared number. For example, when Kim TwinR3 added an email, 900641 got "the email address you use for Poised was added". So the other person on that phone sees the notice. This is expected given the design, noted only.
6. **Info (test tooling, not the app):** on the dev login page and the `/qa/messages` button, the first click right after a page load sometimes only focused the item, and a second click acted. This looks like the browser extension's window focus.

## Console and network errors

- No console errors or warnings on the three tabs (tracked from about 18:43).
- The only 4xx responses were the expected ones: 400, 401, 403, 404, 409 and 429.
- No 5xx responses from the app.
- T-02's B1 answer ("Do you smoke?" No) was confirmed saved after signing back in: `GET /api/p/episodes/<B1 id>` returns it.

## GIFs

- `~/Downloads/S15-T19-twins-name-step-r3.gif`: the twins' shared-mobile sign-in.
  1. Sign in to Poised, then Continue, then the date of birth.
  2. The name step: the red message, with First name focused.
  3. `Lee`/`Someone` gives "…don't match… 4 tries left".
  4. A 101-character name is cut at 100.
  5. ` LEE `/`TWINR3` ends on Lee's home only.

## Hand checks for Rahul

- **R-6:**
  1. As Sam Superadmin, `/admin/trusts/<Trust B>` → **Suspend trust**.
  2. Sam HomeR3's home (sign in by 07700 900542, 17 5 1980) should list only the Hospital A1 card, and B1 episode addresses should be refused.
  3. Re-activate Trust B afterwards.
