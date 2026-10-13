# S14 test report 3: patient invites and magic-link sign-in (retest after verification 1)

**Summary: 5 passed, 0 failed, 0 blocked.** I ran the new T-19 and T-20, plus a quick T-04, T-07 and T-15, as the top of test-plan.md asks. I also explored the on-behalf form under re-renders and Resend more generally. Neither turned up a failure.

**Environment:**
- HEAD `14981f1`, with S14's changes (round 3: the "Not saved" box, Resend sends before it revokes, strict `MESSAGING`, the on-behalf reload dependency, the neutral placeholder) uncommitted in the working tree. The tree also has planning-doc changes that aren't part of S14.
- 13 Oct 2026, 13:51–14:04 BST.
- http://localhost:3000, served by the conductor's `make dev` (MESSAGING=capture).
- Claude in Chrome, automatic mode.

**How the phone size was done:** as in reports 1 and 2, every patient step ran in a **390 × 844 iframe** served from the same origin as the page under test. "Reload" means reloading that iframe's page.

**Round 3 data:** every surname ends in `R3`; the mobiles are 07700 900301–304; the emails are `s14.r3.b`, `f` and `g@example.org`.

| Episode | Contact | Used for |
| --- | --- | --- |
| Sam Lost R3 | M4 | T-19 |
| Sam Locked R3 | M2 | T-04 (T-03's lock) |
| Sam Ended R3 | M1 + `s14.r3.f` | T-15 |
| Sam Phone R3 | `s14.r3.b` | T-07 |
| Sam Typing R3 | `s14.r3.g` | on-behalf exploration, Resend by email |
| Sam Nobody R3 | none | Resend with nothing to send to |

**Where I used tools other than the UI:**
- **Creating episodes:** I created *Sam Lost R3* through the New episode form. The other five I created with `fetch` to the same create-episode API the form uses, signed in as Cara. Creating episodes wasn't under retest.
- **Reading links:** mostly with `curl /api/qa/messages?to=…`. For T-04 I used the captured-messages page itself.
- **Locking the T-04 link:** I did it with curl (Continue, then 5 wrong dates of birth: 401 ×4, then 403). A right date after that also got 403 "locked".
- **Database:** read-only `select`s on `episode_answers` to confirm what was saved and what wasn't.
- **The old and new email links in the Resend exploration:** checked with curl `POST …/continue`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-19 A change refused because the session ended is named | Pass | Lost R3 was created through the UI ("Invite sent by text message."). On the phone: link → Continue → right date → About you → "Do you smoke?" **No** ("Saved"; the DB had `q_smokes: o_no`, rev 1). Cara clicked **Resend invite**. Changing to **Yes** on the phone gave **"You've been signed out"** with the orange **Not saved** box: "Your latest change in About you wasn't saved. When you open the link again, please check your answers in that section.", and under it "For your security, you've been signed out. Please open the link in your text message or email again." The DB still had `o_no`, rev 1. The new link → Continue → right date → About you showed **No** (step 5). I answered every question in both sections (2 of 2 complete), Cara resent again, then **Send my answers** → confirm gave "You've been signed out" + **Not saved** "Your answers haven't been sent yet. When you open the link again, check them and tap Send my answers." Cara's episode, reloaded: still **HQ NOT COMPLETE**. scrollWidth 388. |
| T-20 The captured-messages placeholder | Pass | The **To** box's placeholder reads "A mobile number (as stored, with its country code) or an email address". There's no UK number anywhere at the top of the page. |
| T-04 Resend (quick) | Pass | Locked R3 showed the red **"The link is locked"** box ("…wrongly 5 times (13/10/2026, 13:56)…"). Clicking **Resend invite** gave "Invite sent again by text message.", the red box went, Last sent updated and the General note was added. On the captured-messages page (UI), filtering by `+447700900302` showed 2 texts, newest first, with different links. The old link → Continue gave **"This link doesn't work"**. The new link → `17 5 1980` opened the questionnaire. |
| T-07 Staff fill in on behalf (quick) | Pass | The page was at `/patient-form`, with the blue box "You're filling in Sam Phone R3's health questionnaire for them…", and only About you and Your health. **Send the answers** → confirm gave "Sent for Sam Phone R3 — The answers are with the team for review." On the episode: **Ready for review**, "Sent by Cara Clinician on the patient's behalf on 13/10/2026, 14:00.", both sets marked "Patient's answers entered by Cara Clinician", the note "HQ completed on the patient's behalf", and the invite card gone. The Validate page for About you had "The patient's answers here were entered by Cara Clinician on the patient's behalf." The POA Summary's Patient answers tab had "Patient (entered by Cara Clinician) on 13/10/2026, 14:00" on both patient sets. |
| T-15 A Resend ends an open session (quick) | Pass | Ended R3: link → Continue → right date → About you → **Yes**, then All sections (About you **In progress**). Cara clicked Resend ("Invite sent again by text message and email."). Reloading the phone gave **"You've been signed out"**. Because it was a reload and not a change, there was correctly **no** Not saved box. The new link → right date → About you showed **Yes**. |

## Exploration (beyond the plan)

**The on-behalf form doesn't reload or drop answers when the layout re-renders** (verification finding 5). On Sam Typing R3's `/patient-form`, as Cara:
- I answered "Do you smoke?" Yes, then Next, then alcohol Sometimes.
- Then I triggered re-renders of the hospital layout: I opened and closed (Escape) the **Hospital** selector in the header, and resized the browser window twice. I waited about 5 seconds after each.
- The form stayed on page 2 with **Sometimes** selected and showed "Saved". **No new `GET …/patient-form`** request appeared after I cleared the network log, so the form didn't reload.
- I then changed answers quickly: alcohol Often, then Never, then Previous, then smoke No. All of them saved, ending at `{"q_smokes":"o_no","q_alcohol":"o_never"}`, rev 2.
- One limit: the Trust A demo HQ's patient sections have only radio questions; the one text field, in **Nurse checks**, is clinician-only. So "typing in several fields" here means several radio answers across two pages. I didn't find a published Trust A HQ with a free-text **patient** question.

**Resend, generally:**
- **Double-clicking Resend invite** (Locked R3) gave exactly one resend: one POST in the network log, one note and one new text.
- **Email only** (Typing R3): "Invite sent again by email."; one note. The new email link's Continue gave 200 "Hospital A1". The old email link gave "This link doesn't work".
- **No contact details** (Nobody R3): the card says "No invite has been sent. The General notes say why." **Send invite** is disabled. The note reads "No invite sent. The patient has no mobile or email."
- **After the answers are sent** (Phone R3): the invite card is gone, so Resend isn't offered.
- **A Resend that fails** (the old link should keep working) can't be shown here: capture mode never fails. The builder's integration test covers it.

**Two staff tabs editing the same on-behalf set:**
- Tab A had Typing R3 at rev 2. Tab B, a second tab as Cara, changed smoke to Yes (rev 3).
- Tab A then changed alcohol, and got a clear **"Your answers weren't saved — These answers were changed somewhere else since you opened them. Reload to see the latest; your last change was not saved."** with a **Reload** button.
- Reload showed the latest answers. The 409 is handled well.

**The Not saved box only shows when something was really refused.** A Resend followed by just opening a section (client-side) and then a reload gives a plain "You've been signed out", with no Not saved box. That's correct.

## Found while exploring

None of these is a failure.

1. **The "Not saved" box doesn't survive a reload.** Reloading the "You've been signed out" page after a refused change shows the plain signed-out text, and the hospital's name is gone from the top bar. That's acceptable: the patient has already seen the box, and the reload just restarts at a page that knows nothing about the old session. Mentioned for awareness only.
2. **Wording nit (cosmetic, optional):** the test plan shows "About you" in bold inside the Not saved box; the app shows it as plain text. The meaning is clear either way.
3. **Tooling notes, not app bugs:** as in reports 1 and 2, Claude in Chrome sometimes dropped the first click or keystrokes after a page load, especially in a tab that wasn't the window's selected tab. Examples: the first Resend click on Ended R3, a Day box that stayed empty, and the captured-messages **To** box. Each worked on a retry. Because of this, I couldn't tell whether Enter in the **To** box submits the filter; clicking **Show** works.

## Console and network errors

- Network, as expected:
  - 401 on the ended sessions (T-19, T-15);
  - 403 on the locked link (curl);
  - 409 on the stale on-behalf save in the two-tab exploration (handled by the page);
  - the on-behalf submit logged as 503, which is the extension showing a 204 (known quirk).
- The doubled GETs in the network log are React StrictMode in dev, not the app reloading.
- Console: no errors or warnings were captured on the staff tab or on the phone tab after I started tracking. The phone pages ran inside an iframe, whose console the extension may not capture.

## Recordings

- `~/Downloads/S14-T19-not-saved-box.gif` (T-19 at phone size): answering "Do you smoke?", the **Not saved** box after Cara's Resend, the new link showing **No**, filling in both sections, and the Send refused with "Your answers haven't been sent yet". Cara's Resends happened in another tab.
- There were no failures to record.

STATUS: ALL_PASSED
