# S14 test report 2: patient invites and magic-link sign-in (retest)

**Summary: 8 passed, 0 failed, 0 blocked** (the new T-15 to T-18, plus a quick T-02, T-04, T-06 step 6 and T-07, as the top of test-plan.md asks).

**Environment:**
- HEAD `14981f1`, with S14's changes (round 2: patient sessions in the database, `no-store`, `no-referrer`) uncommitted in the working tree. The tree also has planning-doc changes that aren't part of S14.
- 13 Oct 2026, 13:16–13:30 BST.
- http://localhost:3000 and http://tester.localhost:3000, served by the conductor's `make dev` (MESSAGING=capture).
- Claude in Chrome, automatic mode.

**How the phone size was done:** as in report 1, every patient step ran in a **390 × 844 iframe** served from the same origin as the page under test. "Reload" means reloading that iframe's own page.

**Round 2 data:** every surname ends in `R2`, and the mobiles are 07700 900201–205. The emails are `s14.r2.a`, `b`, `f` and `g@example.org`. Because several cases share M1–M3, I gave each episode its own mobile or email:

| Episode | Contact | Used for |
| --- | --- | --- |
| Sam Ended R2 | M1 + `s14.r2.f` | T-15 |
| Sam LockEnds R2 | M2 | T-16 |
| Sam CopiedCookie R2 | M3 | T-17 |
| Sam Invite R2 | M4 + `s14.r2.a` | T-02 |
| Sam Locked R2 | M5 | T-03/T-04 |
| Sam Reload R2 | `s14.r2.g` only | T-06 |
| Sam Phone R2 | `s14.r2.b` | T-07 |

**Where I used tools other than the UI:**
- **Creating episodes:** I created *Sam Ended R2* through the New episode form. The other six I created with `fetch` to the same create-episode API the form uses, signed in as Cara, to save time. Creating episodes wasn't under retest.
- **Reading links:** I read invite links with `curl /api/qa/messages?to=…`. For T-04 I also used the captured-messages page itself.
- **Locking the T-03/T-04 link:** I did it with curl (5 wrong dates of birth), recording each response's headers for T-18.
- **T-17:** curl, plus one cookie injection into the tester browser, explained in that row.
- **T-18:** response headers checked with curl, and with a wrapper on the page's own `fetch` that recorded `Cache-Control` on the app's real save request.
- **Database:** one read-only look at `patient_sessions`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-15 Resend ends an open session | Pass | Patient signed in and answered "Do you smoke?" = Yes (Saved). Cara clicked Resend invite and got "Invite sent again by text message and email." Reloading the phone showed **"You've been signed out"** ("For your security…"), with 2 × 401 on `GET /api/p/episodes/…`. The new link → Continue → right date opened the questionnaire, and About you still had **Yes**. |
| T-16 Lock ends an open session | Pass | Phone signed in on localhost. In the fresh browser (tester.localhost), wrong dates gave 4 → 3 → 2 → 1 tries, then **"This link is locked"**. Back on the phone, changing an answer straight away gave **"You've been signed out"** (the PUT got 401, so nothing was saved), and so did a reload. Cara's episode showed the red **"The link is locked"** box ("…5 times (13/10/2026, 13:20)…"). After Resend, the new text's link opened the questionnaire. |
| T-17 Sign out ends a copied cookie | Pass | See "How T-17 was run" below. The decoded payload is `{"sub":"d704e521-…","aud":["patient"],"exp":…,"iat":…}`. Its `sub` is the `patient_sessions` id, **not** the patient record id (`cdd5b7fb-…`). Before Sign out, the copied value got 200. After tapping **Sign out** in the questionnaire's top bar ("You've signed out…"), the same copied value got **401** "For your security, you've been signed out. Please open the link in your text message or email again." on GET, and 401 on PUT answers and POST submit. The row's `ended_at` was set at the Sign out time. |
| T-18 No cache, no referrer | Pass | `Cache-Control: no-store` was on every `/api/p` response I checked: continue 200 and 404 (dead link); dob 200, 401 (×4) and 403 (locked); GET episode 200 and 401; PUT answers 200 (the app's own save, seen through the `fetch` wrapper) and 401; submit 401; logout 204. On `/i/<token>`, `document.querySelector('meta[name=referrer]').content` gives **`no-referrer`**, and the page source has `<meta name="referrer" content="no-referrer" />`. |
| T-02 Link, 4 wrong, right, fill, send (quick) | Pass | Before Continue the first screen showed only "Your health questionnaire…" and Continue. After it: "Hospital A1 needs your date of birth…", with tries going 4 → 3 → 2 → 1. The right date moved the page to `/home/episodes/…`, which shows "Hello Sam.", About you and Your health (no Nurse checks), and no DOB, mobile or email. scrollWidth was 386. Filling both sections and Send my answers → confirm gave "Thank you, Sam". On Cara's page: **Ready for review**, the invite card gone, and the note "HQ completed by the patient". |
| T-04 Resend (quick) | Pass | Clicking Resend gave "Invite sent again by text message.", the red box went, Last sent became 13:23 and the note was added. The captured-messages page (UI, filtered to `+447700900205`) showed 2 texts, newest first, with different links. The old link → Continue gave **"This link doesn't work"**. The new link with the right date opened the questionnaire. |
| T-06 step 6 Sign out then reload | Pass | After a reload the patient was still signed in. Sign out showed "You've been signed out — You've signed out. To come back, open the link…". After a reload it still said signed out ("For your security…"), with no questionnaire. |
| T-07 Staff fill in on behalf (quick) | Pass | The page was at `/patient-form`, with the blue box "You're filling in Sam Phone R2's health questionnaire for them…". Sending gave "Sent for Sam Phone R2". On the episode: Ready for review, "Sent by Cara Clinician on the patient's behalf on 13/10/2026, 13:27.", both sets marked "Patient's answers entered by Cara Clinician", and the note "HQ completed on the patient's behalf". The Validate page had the orange box. The POA Patient answers tab said "Patient (entered by Cara Clinician) on 13/10/2026, 13:27". |

**How T-17 was run.** Claude in Chrome can't read an HttpOnly cookie, so I couldn't copy it out of the browser. Instead:
1. I opened CopiedCookie R2's link with curl (Continue, then the right date) and kept the `poised_patient` value from `Set-Cookie`.
2. I put that value into the tester.localhost browser (`document.cookie`, path `/api/p`) and opened `/home/episodes/<id>` there. The questionnaire showed, so the browser and my copy were the same session.
3. I tapped **Sign out** in the browser, then replayed the copied value with curl.

The `Set-Cookie` also showed `Path=/api/p; Max-Age=1800; HttpOnly; SameSite=Strict`. It has no `Secure`, which is correct here: the code adds `Secure` when the app URL is https.

## Failures

None.

## Found while exploring

None of these is a failure.

1. **Sign out ends only that browser's session, as you'd want.** I opened two sessions on the same link with curl. Signing out A gave A 401 and left B at 200; signing out B then gave B 401. A Resend or a lock ends every session on the link, as T-15 and T-16 showed.
2. **Opening a second link in the same browser doesn't end the first session on the server.** For Rahul's awareness; low risk. On the phone I opened Ended R2's new link and then LockEnds R2's link. The browser's cookie was replaced, but Ended R2's session row (`bd2d0f8f…`) has no `ended_at`, so it stays valid until it has been idle for 30 minutes.
   - Only someone who had already copied that cookie could use it.
   - Rows ended by a Resend or a lock also keep `ended_at` empty. They're refused through the link's revoked or locked state instead, which T-15 and T-16 confirm works.
3. **What the patient sees when a session ends.** Changing an answer after a lock goes straight to "You've been signed out", and the answer isn't saved. No message says the answer was lost. That's acceptable, since the patient must reopen the link anyway.
4. **Tooling notes, not app bugs:**
   - Claude in Chrome often dropped the first click after a page load or a script call, as in report 1.
   - In a background tab (`document.visibilityState` = hidden), the "Send the answers?" dialog's closing animation paused with its overlay still showing, so a click on "Back to the episode" didn't land. The dialog closed by itself a few seconds later, and the click then worked. This was the browser pausing a hidden tab's animations, not the app.

## Console and network errors

- No console errors or warnings on the patient pages or the staff pages during this round.
- Network: only the expected ones:
  - 401s after a wrong date of birth, and on the ended sessions in T-15, T-16 and T-06;
  - 404 for the old link in T-04;
  - submit and logout logged as 503, which is the extension showing a 204 (known quirk).

## Recordings

- `~/Downloads/S14-T15-resend-ends-session.gif`: T-15 at phone size. It shows the link, Continue, the date of birth, answering About you and All sections; Cara's Resend happened in the other tab. The reload shows "You've been signed out", and the new link opens with the answer kept.
- No failures to record.

STATUS: ALL_PASSED
