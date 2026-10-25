# S30 manual test plan: the follow-ups after S29

**For the tester:** you've never seen the code. Use only the browser (and, where a step says so, the browser's JavaScript console or your page-scripting tool). Report each case as Pass, Fail (with what you saw) or Blocked (why). The brief is [brief.md](brief.md).

## What this session changed
1. **The plain network message for any server error without the API's own message.** When something between the browser and the API answers a 5xx error with a body that isn't the API's (on AWS, the gateway's `{"message":"Internal Server Error"}`), every page now says **"We couldn't reach the server. Check your connection and try again."** Before, it said "Something went wrong. Please try again.". An error that carries the API's own message (a 409, a 422, even a 500) still shows that message.
2. **The "answers in this section were cleared" card hides only the network message.** After **Start fresh**, if the questionnaire can't be loaded again, the card shows with no red banner (as in S29). But if **Reload** is then refused for another reason, for example because the episode was cancelled meanwhile, the red banner says why, above the card.
3. **J8 (an automated journey) reads only its own episode's texts.** There's nothing to test in the browser.

## Setup

**Rahul's terminal** (the conductor does this in an automatic run):
```sh
docker compose up -d db
make migrate seed      # no new migration in S30; seeds the demo data
make dev               # leave it running
```

**Open:** http://localhost:3000

**Staff sign-in:** the stub login page lists the seeded users; click a name to sign in. Sign out (top right) before signing in as someone else, or use a second browser profile or an incognito window. The users you need:
- **Cara Clinician**: a clinician at Hospital A1 (Trust A). She creates episodes, fills in for patients and cancels episodes.
- **Bea Author**: a super clinician of the whole of Trust B. She creates Trust B episodes at **Hospital B1** (pick Hospital B1 in the hospital picker).
- **Sam Superadmin**: the platform admin. He opens the testers' pages.

**Patients:** there's no patient sign-in page. A new episode sends the patient an invite email, which is captured, not sent.
- Open **http://localhost:3000/qa/messages** (sign in as **Sam Superadmin** if it asks).
- Find the newest email to the patient's address, and open its link (`/i/…`).
- Press **Continue**, then enter the patient's date of birth.
- **Test patient pages at phone size (390 × 844):** use Chrome's device toolbar, or a window that narrow.

**A person with a Trust B history (used by T-04 and T-05):**
1. As **Bea** (Hospital B1), create an episode on the **Demo pre-op HQ** for a new patient: first name **Sam**, a surname you'll recognise (e.g. "S30 T04"), date of birth **17/05/1980**, and a fresh made-up email (e.g. `s30.t04@example.test`).
2. As the patient (open that invite link, at phone size), answer:
   - **About you:** smoker "I used to smoke", alcohol 6, height 170, weight 82;
   - **Your health:** diabetes No, snoring **Yes**, pregnant No, sick after an anaesthetic No, problems No, family No.

   Then **send**.
3. As **Cara** (Hospital A1), create an episode on the **Demo follow-up HQ** for a new patient with the **same** first name, surname, date of birth and email.

Opening that Trust A episode fills both sections from Trust B, whether as the patient through its invite, or as Cara with **Fill in for the patient**. Each section then opens on a card with **Check each answer**, **Everything is still correct** and **Start fresh** (Start fresh asks "Yes, start fresh" first).

**Making the server misbehave in the browser:** the app sends every request through the page's `fetch`. Run these in the browser's JavaScript console (DevTools → Console), or with your page-scripting tool, on the page. A full page reload (F5) undoes any of them.
- **(A) Every request gets the gateway's JSON 502:**
  ```js
  window.fetch = () => Promise.resolve(new Response('{"message":"Internal Server Error"}', { status: 502, headers: { "Content-Type": "application/json" } }))
  ```
- **(B) Every request gets the API's own refusal (a 409 with its message):**
  ```js
  window.fetch = () => Promise.resolve(new Response('{"error":"Made-up refusal from the API."}', { status: 409, headers: { "Content-Type": "application/json" } }))
  ```
- **(C) Reads that can be switched off and on (changes still go through):**
  ```js
  window.__offline = false; const f0 = window.fetch; window.fetch = (u, i) => (window.__offline && !(i && i.method && i.method !== "GET")) ? Promise.reject(new TypeError("Failed to fetch")) : f0(u, i)
  ```
  Then `window.__offline = true` makes every read fail, and `window.__offline = false` puts them back, without reloading the page.

If you can't use the console or a scripting tool, mark the cases that need it **Blocked (needs a changed fetch)**. An automated journey covers them.

## Test cases

### The plain network message

**T-01. A staff page and the gateway's JSON 502**
1. As **Cara**, open any episode, then its **POA Summary**.
2. Run **(A)** in the console.
3. In **General notes** press **Add comment**, type anything, and press **Save comment**.
4. **Expected:** a red message **"We couldn't reach the server. Check your connection and try again."** Not "Something went wrong", and not "Failed to fetch".
5. Still without reloading, use the app's own links to move to another page (e.g. the top bar's **Episodes**, or the episode's name). **Expected:** where that page can't load, it shows the same plain message, never "Something went wrong".
6. Reload (F5). **Expected:** the page works again, and **Save comment** saves.

**T-02. A patient page and the gateway's JSON 502 (phone size)**
1. As **Cara**, create an episode on the **Demo pre-op HQ** for a new patient ("S30 T02", date of birth 17/05/1980, a fresh email). Open its invite as the patient (phone size).
2. Open **About you** and answer one question. Run **(A)** in the console.
3. Change that answer.
4. **Expected:** a red message "We couldn't reach the server. Check your connection and try again." No "Something went wrong", and nothing wider than the screen.
5. Reload (F5). **Expected:** the page works again (the last change may be lost; that's fine).

**T-03. The API's own message still wins**
1. On the patient page from T-02 (phone size), run **(B)** in the console, then change an answer.
2. **Expected:** the message **"Made-up refusal from the API."**, not the network message. (The page may show it as "Your answers weren't saved" with Reload; that's the usual handling of a 409.)
3. Reload (F5). As **Cara**, on any episode's POA Summary, run **(B)**, then **Add comment** → type → **Save comment**. **Expected:** "Made-up refusal from the API." in red.
4. *(A real refusal.)* Open T-02's invite link in a fresh window (phone size) → **Continue** → enter a **wrong** date of birth (e.g. 01/01/1990). **Expected:** the app's usual wrong-date-of-birth message, not the network message.

### The cleared card

**T-04. Start fresh whose reload fails: only the card (phone size)**
1. Make a person with a Trust B history ("S30 T04"; see Setup), and open the Trust A invite as the patient (phone size).
2. Open **About you**. Run **(C)**, then `window.__offline = true`.
3. Press **Start fresh** → **Yes, start fresh**.
4. **Expected:** the card **"The answers in this section were cleared"**, with "The questionnaire couldn't be loaded again just now. Please try again." and **Reload**. There is **no red banner** above it, and nothing is wider than the screen.
5. Press **Reload** while still offline. **Expected:** still only the card, with no red banner.
6. Run `window.__offline = false`, then press **Reload**. **Expected:** About you opens **empty**, with no card and no banner.
7. *(The gateway's JSON over the card.)* Reload (F5) and open **Your health**. Run **(C)** again, but before setting it offline, replace it with the gateway's 502 for reads only:
   ```js
   const f1 = window.fetch; window.fetch = (u, i) => (i && i.method && i.method !== "GET") ? f1(u, i) : Promise.resolve(new Response('{"message":"Internal Server Error"}', { status: 502, headers: { "Content-Type": "application/json" } }))
   ```
   Press **Start fresh** → **Yes, start fresh**. **Expected:** the cleared card only, with no red banner. That 502 counts as the network message.

**T-05. A Reload refused because the episode was cancelled: the message shows**
1. Make another person with a Trust B history ("S30 T05"). **Don't** open their Trust A invite as the patient.
2. As **Cara**, open T05's Trust A episode → **Fill in for the patient** → **Your health** → **Check**. The card appears.
3. Run **(C)**, then `window.__offline = true`. Press **Start fresh** → **Yes, start fresh**.
4. **Expected:** the cleared card, with **no** red banner.
5. In a **second tab** (same browser, still Cara), open the same episode and **cancel** it (any reason, e.g. "Patient withdrew").
6. Back in the first tab (don't reload it), run `window.__offline = false`, then press **Reload** on the card.
7. **Expected:** a red banner **"This questionnaire was cancelled by the hospital, so there's nothing more to fill in. Please contact the hospital if you have any questions."** above the cleared card. The card stays.

*Note:* a patient who came in through **that** episode's invite is signed out when the episode is cancelled (as designed since S20). So the patient's version of T-05 shows the "signed out" page, not the cancelled banner. That isn't a failure.

## Regression

- **R-01 (S29):** with no misbehaving fetch, make every request fail: `window.fetch = () => Promise.reject(new TypeError("Failed to fetch"))`. Then save a comment on a POA Summary. **Expected:** "We couldn't reach the server…", never "Failed to fetch".
- **R-02 (S27/S28):** a plain **Start fresh** with nothing failing: the section opens empty, with no card and no banner. After a reload (F5) it's still empty.
- **R-03 (S20):** the patient opens a cancelled episode's form (e.g. T-05's episode, through a new invite to another episode for the same person, or the patient's home). **Expected:** the cancelled message, as before.
- **R-04 (S16):** as **Sam Superadmin**, open `/qa/jobs`, choose **Now + 2 days**, and press **Run jobs**. **Expected:** "Ran N jobs as of …", and the captured-messages page lists reminders for unanswered episodes. (This is the page J8 drives.)

## Out of scope (don't report)
- An action ("What") filter on the Audit page (Rahul: later).
- The other S29 findings in `docs/pending.md`: the old label on a refused pre-fill press, the `reportedBmi` tidy-ups, and the `EpisodeList` tidy-up.
- Rahul's own hand checks (he'll do them in one go later), and API Gateway's real 5xx on AWS (local only).
- The "Failed to fetch" text that the browser's own console prints for a blocked request. Only the page's wording counts.
