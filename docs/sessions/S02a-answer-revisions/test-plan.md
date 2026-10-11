# S02a manual test plan: a stale save is refused, not silently applied

**Changed after verification 2 (11 Oct):** the patient page was changed again.
- **The bug it fixes:** a section that was already **Complete** still counted as complete after the patient changed an answer whose save failed. So the HQ could be sent without the change.
- **Now:**
  - a section with a change that hasn't saved shows **In progress** (**Continue**), and **Send my answers** stays disabled until the change saves;
  - the section's save label says **Not saved** (in red) after a failed save, instead of "Saved";
  - **Reload** on the "weren't saved" card clears any old error message, and shows it's loading.
- **T-12 has new steps 9–13 for this.**
- **For this retest, run:**
  - T-12 (all of it), T-04 and T-06 (the patient card and Reload);
  - T-07 (X-1);
  - R-01 and R-02.

**Changed after verification 1 (9 Oct):** the patient page was changed to fix a bug the verifier found. After a save failed (e.g. no network), the answers were never sent again, yet the section could look complete.
- **New case T-12** checks the fix with DevTools' **Offline** switch.
- **On the section list, "Complete", "In progress" and "Send my answers" now go by what the server has.** A section whose last save failed shows as not started until a save succeeds. Its answers stay on screen when you reopen it.
- **For this retest, run:**
  - T-03 to T-06 and T-12 (the patient page);
  - T-07 (X-1);
  - R-01 and R-02.
  The clinician page's code didn't change after report 1.

**What changed in S02a:**
- **The problem it fixes (S02's test report 2, X-1):**
  1. Two tabs (or two people) open the same Question Set.
  2. One tab changes an answer.
  3. The other tab, still showing the old answer, saves or clicks **Validate**.
  4. Before S02a, the older tab's save quietly replaced the newer answer, and nobody was told.
- **Now:** the older tab's save is refused, and it shows a message with a **Reload** button. Nothing it sent is saved. Reload shows the newer answers.
- **This applies to both answer pages:**
  - the clinician's (validating a Question Set);
  - the patient's link, when it's open on two devices or in two tabs.
- **The message:** "These answers were changed somewhere else since you opened them. Reload to see the latest; your last change was not saved."
- **A tab only finds out when it tries to save.** A stale tab that just sits there shows no message.
- **Also fixed, but you can't trigger it by hand:** changing an episode's details (e.g. an ASA grade) at the same moment as the workflow changed its status could put the old status back. The automated tests cover it. Here, R-03 only checks that normal changes still work.

## 1. Setup

**Rahul runs these in his own terminal before testing, and leaves the last one running.** If `make dev` was already running, he stops it (Ctrl-C) and starts it again, because the API must be rebuilt.

```sh
docker compose up -d db
make migrate seed
make dev
```

- **The app:** http://localhost:3000. If it doesn't load, stop and report it. Don't try to start or fix the servers.
- **Sign-in:** the sign-in page lists the users; click a name. There's no password. **Sign out** is at the top right.

  | User | Hospital | Can do |
  | --- | --- | --- |
  | Alex Author | Hospital A | write and publish questionnaires; not a clinician |
  | Cara Clinician | Hospital A | patients, episodes, validation, POA Summary |
  | Bea Author | Hospital B | Hospital B only |

- **Data rules:**
  - Use made-up text only. Never enter a real person's details.
  - Name new patients `S02a Test<time>`, e.g. first name `S02a`, last name `Test1405`.
  - Don't change questionnaires or episodes you didn't create.
- **Terms:**
  - an **HQ** is a health questionnaire;
  - a **Question Set** is a section of it;
  - an **episode** is one patient's HQ, from the link to the POA Summary.
- **Two tabs:** most cases need the same page open in **two tabs, A and B**.
  - Open the page in tab A, then copy its address into a new tab B.
  - Each case says which tab to use for each step. Wait for **Saved** where a step says so: answers save about half a second after a change.
- **Phone size:** "phone size" means a 390 × 844 viewport. Use DevTools' device toolbar (iPhone 12 Pro or 14) in each tab.
  - If that doesn't work with your tools, a 390 × 844 same-origin iframe is acceptable; say so in the report.
  - For two patient "devices", two such iframes on one page are fine.
- **No private window?** Sign every staff user out before opening the patient link, and use new tabs.
- **Never clear site storage** (local or session storage, cookies) for `localhost:3000`. Other local apps on this machine keep data there.
- **Console and network:** keep DevTools open, and report any red console error with the case ID.
  - **Expected, not errors:**
    - a `409 (Conflict)` on the save that each case refuses on purpose;
    - two `401 (Unauthorized)` on the sign-in page.
  - **A change since S02:** a successful answer save now answers **200** with a revision, instead of 204.
  - **Known tool quirk:** the Claude in Chrome network log shows every `204 No Content` as **503**. Ignore a 503 on sign-out, submit or complete review.

## 2. Test cases

### T-01: Make the test HQ
1. Sign in as **Alex Author**. You're on Hospital A's questionnaire list.
2. Open DevTools → **Console**. Paste the whole contents of `docs/sessions/S02-workflow-route-tests/make-test-hq.js` and press Enter. If Chrome asks you to type "allow pasting" first, do that, then paste again.
3. **Expected:** the console prints `Created and published: S02 test HQ <hh:mm>`. Note that name.
4. Reload the list. **Expected:** the HQ is listed as published (v1), with three Question Sets: **About you**, **Your health** and **Nurse checks**.
5. Sign out.

### T-02: Create an episode, and copy the patient link
1. Sign in as **Cara Clinician**. Click **New episode**, choose **New patient**, and enter a made-up patient (`S02a Test<time>`, any date of birth).
2. Choose your T-01 HQ, and click **Create episode**.
3. **Expected:** the episode page, status **HQ not complete**, with a **Patient link** and a **Copy** button.
4. Copy the link and keep it. Note the episode page's address too. Sign out.

### T-03: The patient's normal use is unchanged (phone size)
1. Open the patient link in a new tab, at phone size.
2. **Expected:** the list of sections, **About you** and **Your health**. Nothing that says **STAFF ONLY** anywhere.
3. Open **About you**. Answer **Do you smoke?** with **No**. Wait for **Saved**.
4. Close the tab. Open the link in a new tab, at phone size.
5. **Expected:** About you shows **In progress**. Open it: smoking is still **No**. No message.
6. Click **All sections** without changing anything. **Expected:** back to the list, with no message.

### T-04: The link open on two devices: the older one is refused (phone size)
1. Open the patient link in **tab A** and **tab B**, both at phone size. In both, open **About you**. Both show smoking **No**.
2. In **tab B**, change smoking to **Yes**. Wait for **Saved**.
3. In **tab A**, which still shows **No**, answer **Do you have angina?** with **Yes**.
4. **Expected in tab A,** within a second or so:
   - the section is replaced by a card titled **Your answers weren't saved**;
   - it shows the message "These answers were changed somewhere else since you opened them. Reload to see the latest; your last change was not saved.";
   - it has a **Reload** button;
   - the card fits the phone screen with no sideways scrolling.
5. In tab A, click **Reload**.
6. **Expected:** tab A shows **About you** again, with smoking **Yes** (tab B's change) and angina **not answered** (tab A's change wasn't saved).
7. In tab A, answer angina **Yes** again and wait for **Saved**. **Expected:** no message: tab A is up to date now.
8. In **tab B** (which hasn't seen tab A's angina answer), click **All sections**, without changing anything.
   - **Expected:** back to the list, with **no** message. A tab that changed nothing doesn't save anything.

### T-05: Two first saves of a section (phone size)
1. In tab A and tab B (at phone size), go to the list and open **Your health**, which has never been answered.
2. In **tab A**, answer its first question. Wait for **Saved**.
3. In **tab B**, answer the same question differently.
4. **Expected in tab B:** the **Your answers weren't saved** card with **Reload**.
5. Click **Reload**. **Expected:** Your health shows tab A's answer.

### T-06: A section saved after the answers were sent (phone size)
1. In **tab A**, complete every section (use **Finish this section**), then **Send my answers** and confirm. **Expected:** the thank-you page.
2. In **tab B**, open a section if one isn't open. If B is on the list, reload it first: it will show the thank-you page, which is also correct. Stop the case there and note it.
3. In tab B, change an answer.
4. **Expected:**
   - the **Your answers weren't saved** card, with "Your answers have already been sent, so they can't be changed here." and **Reload**;
   - **Reload** shows the thank-you page.

### T-07: X-1 exactly: a stale clinician tab's Validate is refused
1. Sign in as **Cara Clinician**. Open your T-02 episode. Click **Validate** next to **About you** (or open that Question Set from the episode page).
2. Copy the address into **tab B**. Both tabs show the same answers.
3. In **tab A**, change **Do you smoke?** to the other answer. **Expected:** "Patient answered: …" appears under it, then **Saved**.
4. In **tab B**, which still shows the old answer, click **Validate**.
5. **Expected in tab B:**
   - a red **Not saved** box with the message "These answers were changed somewhere else since you opened them. Reload to see the latest; your last change was not saved." and a **Reload** button;
   - the save status says **Not saved**;
   - **Validate** is disabled, and the answers can't be changed;
   - tab B stays on the set page; it doesn't go back to the episode.
6. Click **Reload** in tab B.
7. **Expected:** tab B shows **tab A's answer**, and **no** "Validated by" badge.
8. Open the episode page, then **POA Summary**. **Expected:** the smoking answer is tab A's.
9. Back on the episode page, **About you** is **not** marked validated.

### T-08: Two first saves of a clinician set
1. As Cara, open **Nurse checks** (the clinician's own set, never saved) in tab A and tab B.
2. In **tab A**, fill in **blood pressure** (any made-up value). Wait for **Saved**.
3. In **tab B**, fill in blood pressure with a different value.
4. **Expected in tab B:** the **Not saved** box with **Reload**. **Reload** shows tab A's value.

### T-09: Normal clinician use is unchanged (one tab)
Close the extra tabs, so only one is open.
1. Open **About you**. **Expected:** "Patient answered: …" notes where the clinician copy differs from the patient's.
2. Change one answer, then click **Validate**. **Expected:** back on the episode page; About you shows **Validated by Cara Clinician** with the time. No message anywhere.
3. Validate **Your health** and **Nurse checks** the same way. Fill in any required question first.
4. Click **Complete HQ review**. **Expected:** the status is **Ready for POA**, and General notes say "HQ review completed by Cara Clinician".
5. Open **POA Summary**, and look at both of its tabs. **Expected:** the answers you validated, with each correction showing what the patient said.

### T-10: A stale tab after the review is complete
1. Use a new episode: repeat T-02 to T-03 quickly (any answers), and send the patient's answers.
2. As Cara, open **About you** in **tab B** and leave it.
3. In **tab A**, validate every set and **Complete HQ review**.
4. In **tab B**, change an answer.
5. **Expected:**
   - the **Not saved** box, with "The HQ review is already complete, so the answers can't be changed." and **Reload**;
   - **Reload** shows the set read-only, with "The HQ review is complete, so these answers can't be changed."

### T-11: Refusals still work
1. Sign in as **Alex Author**. Paste the address of a T-07 set page. **Expected:** a refusal such as "Only clinicians can see patients and episodes.", and no answers shown.
2. Sign in as **Bea Author**. Paste the same address. **Expected:** "You don't have access to this hospital."

### T-12: A save that fails is sent again, and can't be sent unfinished (phone size)
1. Make a new episode (as in T-02), sign out, and open its patient link at phone size.
2. In DevTools → **Network**, set the throttling menu to **Offline**.
3. Open **About you**. Answer both questions. **Expected:** an error appears at the top (e.g. "Failed to fetch"). That's the failed save.
4. Click **All sections**. **Expected:**
   - About you is **not** marked Complete (it shows **Start**);
   - "0 of 2 sections complete";
   - **Send my answers** is disabled.
5. Set the throttling menu back to **No throttling**. Dismiss the error.
6. Open **About you**. **Expected:** both of your answers are still shown.
7. Click **All sections** without changing anything. **Expected:**
   - a save is sent and succeeds (a `PUT …/answers/…` with **200** in the Network tab);
   - About you now shows **Complete**, and "1 of 2 sections complete".
8. Close the tab, and open the link in a new tab. **Expected:** About you is Complete, with your answers. The server has them.
9. Complete **Your health** too, so the list says "2 of 2 sections complete" and **Send my answers** is enabled. Don't send yet.
10. Set the throttling menu to **Offline**. Open **About you** (**Review**), and change **Do you smoke?** to the other answer.
    - **Expected:** the error at the top, and the label next to **All sections** says **Not saved**, in red.
11. Click **All sections**. **Expected:**
    - About you shows **In progress** (with **Continue**), not Complete;
    - "1 of 2 sections complete";
    - **Send my answers** is disabled.
12. Set throttling back to **No throttling**. Open **About you** (**Continue**).
    - **Expected:** your changed answer is shown, and the label says **Not saved**.
13. Click **All sections**. **Expected:**
    - a `PUT …/answers/…` with **200**;
    - About you is **Complete** again, "2 of 2", and **Send my answers** is enabled;
    - opening the link in a new tab shows your **changed** answer.

## 3. Regression

- **R-01: No clinician content for the patient.** During T-03 to T-06, nothing that says **STAFF ONLY**, and no "Current smoker" note, appears on any patient screen.
- **R-02: Leave and come back keeps answers.** Already covered by T-03 step 5. Also reload the patient tab in the middle of a section: the answers are kept.
- **R-03: Episode details still change.** On a **Ready for POA** episode's **POA Summary**:
  - set the **Nurse ASA grade** to 2, then clear it;
  - change the **Episode status** to **POA complete**.
  - **Expected:** each change is saved, with no message, and General notes record each one.
- **R-04: The editor's own stale-tab check is unchanged.** As Alex, open a **draft** questionnaire's Question Set in two tabs. Change a question title in tab A and let it save, then change something in tab B. **Expected:** tab B gets the editor's existing "Someone else changed this chapter…" message, as before S02a.

## 4. Out of scope (don't report)

- **Not built, on purpose:**
  - "Someone else is looking at this set" (presence);
  - locking a set while someone edits it.
- **Older items already recorded in `docs/pending.md` §3:**
  - the **Saved** label showing before any answer;
  - the staff header wrapping at narrow widths;
  - the New episode drawer leaving a patient when episode creation fails;
  - the patient link's response including the date of birth;
  - the access-error pages having no way back;
  - "Sign out" occasionally not taking on the first click.
- **The 503 that the tool shows for 204 responses.**
- **Complete HQ review is allowed before every set is validated on the server.** The page enables the button only when every visible set is validated, and the server check is planned for C6.
