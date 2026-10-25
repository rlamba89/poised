# S33 test report 4 (retest after verification 1)

**9 passed, 0 failed, 0 blocked.**

- **Environment:**
  - Code: commit `fdc963b` plus the uncommitted S33 work, so the working tree isn't clean.
  - When and where: Sun 25 Oct 2026, 19:32–19:49 GMT, at http://localhost:3000 (API :8080, run by the conductor's `make dev`, freshly migrated and seeded).
  - Browser: Chrome ("trk" profile) through Claude in Chrome. Automatic mode.
- **Narrow widths:** the window can't be resized, so I tested 390px in a same-origin iframe. Its `innerWidth` was exactly 390.
- **Scope (the builder's request):** the new T-16, T-17 and T-18, then a re-run of T-02, T-03, T-05, T-07 and T-13, and a quick T-14 at 390px.
- **Test data I made:**
  - **Platform (Sam), all retired at the end:**
    - Capital leaflet 1933 and Capital video 1934;
    - Knee video 1936 and Fasting leaflet 1939;
    - five "MT S33 193x …" accepted-link videos;
    - "MT S33 1945 status active" (from the T-18 probe).
  - **Trust A (Ada):**
    - Retired: "Capital leaflet A 1955", "MT S33 2005 double 🩺 leaflet" and "MT S33 2007 probe".
    - Left active: "A hip video 1956" (Hip, Recovery, 12:30) and "A who-to-call leaflet 1957".
    - To give T-07 step 3 a "nothing yet" start, I retired an earlier round's "A who-to-call leaflet 1851".
  - **Trust A (Alex):** a draft questionnaire, "MT S33 1950 starter q", containing "Medicines you take regularly".
  - **Starter questions:** I retired "Medicines you take regularly" and offered it again. All 11 are offered at the end.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-16 Capital letters in a link | Pass | See the detail below the table. |
| T-17 An old error doesn't come back | Pass | See the detail below the table. Note the off-screen message under "Found while exploring". |
| T-18 New item can't be added already retired | Pass | `400 {"error":"A new item is always active."}`. Nothing named "Retired at birth" in the list. |
| T-02 Add a platform video with tags | Pass | See the detail below the table. |
| T-03 Non-YouTube/Vimeo links refused | Pass | See the detail below the table. |
| T-05 Leaflet; edit; retire and bring back | Pass | See the detail below the table. |
| T-07 Trust A video and leaflet | Pass | See the detail below the table. |
| T-13 Retire a starter question | Pass | See the detail below the table. |
| T-14 Phone size (390px) | Pass | See the detail below the table. |

**T-16 in detail**

- **Step 1 (Sam):** `Https://WWW.Trust-A.example/Leaflets/Capital.pdf` was added with no "Something went wrong". The row's link reads `https://www.trust-a.example/Leaflets/Capital.pdf`: scheme and host lower case, path capitals kept.
- **Step 2:** `HTTPS://WWW.YOUTUBE.COM/watch?v=aqz-KE-bpKQ` was added and shown as `https://www.youtube.com/watch?v=aqz-KE-bpKQ`.
- **Step 3:** an Edit to `Https://Vimeo.com/76979871` was saved as `https://vimeo.com/76979871`. I confirmed the stored values through `GET /api/admin/content`.
- **Step 4 (Ada):** the same leaflet was added the same way, and its href is lower-cased.
- **Step 5:**
  - `HTTP://www.youtube.com/watch?v=…` on a video got the plain "Only YouTube or Vimeo links…" message.
  - `HTTP://example.org/a.pdf` on a leaflet got "Enter the leaflet's web address, starting https://".
- **Step 6:** the link input is `type="url"` and `autocapitalize="none"`, plus `autocorrect="off"` and `spellcheck="false"`.
- **Step 7:** I retired everything I added.

**T-17 in detail**

- **Step 1:**
  - With the plan's fetch override, Retire on "Capital leaflet A 1955" showed the red "Test: this retire was refused.".
  - The item was unchanged and the override removed itself.
- **Step 2:** Edit opened the form and the red message went. Cancel brought no message back.
- **Step 3:** I re-armed the override and got the refusal again on "A hip video 1956". Then **Add video** → **Cancel**: no old message. A plain Add video → Cancel before that showed none either.
- **Step 4:** the real Retire gave "Capital leaflet A 1955 is retired: it isn't offered any more." The row shows Retired with **Bring back**.

**T-02 in detail**

- "Knee video 1936 is added.": 3:20, tags KNEE (typed + Enter) and RECOVERY (**+ Recovery**).
- The link opens `https://youtu.be/aqz-KE-bpKQ` in a new tab (`target=_blank`, `rel=noopener noreferrer`).
- The checklist's Recovery went from "nothing yet" to covered (8 → 7 of 11).

**T-03 in detail**

- **Refused links:** Dailymotion, example.org .mp4, `http://` YouTube and a YouTube channel link each showed the plain YouTube-or-Vimeo message (POST 400). Nothing was added.
- **Accepted links:** YouTube watch, youtu.be, embed, Vimeo, the Vimeo private link and player.vimeo.com were each added, stored as typed.

**T-05 in detail**

- **The leaflet:** "Fasting leaflet 1939" was added: Leaflet, length "—", FASTING. The leaflet form has no Length field.
- **The edit:** it showed 3:20 and both tags. I set 4:05, removed Knee and pressed Enter in Length: "Knee video 1936 is saved.", showing 4:05 and RECOVERY.
- **Retire:** "… is retired: it isn't offered any more.". The row went to the bottom (row 30 of 30), greyed, with **Retired** and **Bring back**. The checklist's Recovery went back to "nothing yet".
- **Bring back:** "… is offered again." and Recovery was covered again.
- **Retired again** at the end.

**T-07 in detail**

- **The video:** "A hip video 1956" (Vimeo private link, 12:05, Hip) was added under **Trust A's items**.
- **The leaflet:** "A who-to-call leaflet 1957" (**+ Who to call**) was added. Who to call went from "nothing yet" to covered (7 → 6 of 11).
- **The edit:** + Recovery and 12:30 gave "… is saved.", showing 12:30, HIP and RECOVERY.
- **A bad link on Edit:** `https://example.org/film.mp4` got the same plain message. After Cancel, the row was unchanged.

**T-13 in detail**

1. As Alex, in the new draft "MT S33 1950 starter q", **From library** inserted "Medicines you take regularly" and showed "All changes saved".
2. As Sam, **Retire** gave the green "“Medicines you take regularly” is retired: Insert from library no longer offers it. Questionnaires that already have it keep it.". The row went last, greyed, **Retired**, with **Offer again**.
3. As Alex, after a reload the question was still there, unchanged, with the ID `regular_medicines`.
4. **From library**:
   - Searching "medicines" gave "No library questions match."
   - An empty search listed the other 10 starter questions.
5. As Sam, **Offer again** gave "“Medicines you take regularly” is offered again.". As Alex, **From library** offered it again for "medicines".

**T-14 in detail**

- **Ada at 390px:**
  - The page is 390px wide.
  - The Add video fields stack at 324px wide (Title, link, Length, Tags), and the Cancel and Add video buttons fit.
  - **Trust A's items** (820px) and **From Preopsis** (680px) each scroll in their own 358px box.
  - None of the 151 badges is clipped or spills out of its cell, including all 50 **Retired** badges.
- **Sam at 390px:**
  - The page is 390px wide.
  - The top bar scrolls on its own (358px box, 562px of contents). **PLATFORM ADMIN**, **Trusts**, **Content library** and **Sign out** read in full.
  - The items (820px) and **Starter questions** (760px) tables scroll in 358px boxes.
  - **MEASUREMENTS** reads in full on both rows (label 99/99px), including with the table scrolled fully right.
  - None of the 77 badges is clipped.

## Failures

None. Verification 1's two must/should fixes are confirmed:
- capital letters in a link are saved, lower-cased;
- a stale error doesn't come back after Edit or Add → Cancel.

Its optional point 3 is also in: a POST with `status: "retired"` is now a 400.

## Found while exploring

- **Minor (usability), the T-17 message can be off-screen.**
  - **Steps:** open Trust A's list (it's long after several rounds), scroll down, and click Retire on an item lower down while the PATCH fails.
  - **What happens:** the red message appears at the top of the page, above the viewport (its top was at −277px). The page doesn't scroll to it, so the admin sees nothing happen: the row stays as it was and there is no visible feedback.
  - **Successes:** the green "… is retired" message is also at the top, but there the row itself visibly changes.
  - **Suggestion:** consider scrolling the message into view, or showing a failure next to the row. It isn't a plan failure, because the plan only asks that the message appears.
- **The link check is strict in good ways.** Through the API as Ada:
  - `HTTPS://YOUTU.BE/aqz-KE-bpKQ` was stored as `https://youtu.be/aqz-KE-bpKQ`.
  - These were refused with the plain message:
    - `https://www.youtube.com.evil.example/…` (a look-alike host);
    - `https://user@www.youtube.com/…` (a user name in the link);
    - `https://WWW.YOUTUBE.COM:443/…` (an explicit port).
- **T-18 extras:** `status: "RETIRED"` and `status: "bogus"` both get 400 "The status must be active or retired.". `status: "active"` gets 201.
- **Double-click on Add leaflet** created exactly one item. A title with an emoji ("🩺") was fine, and a link typed with leading and trailing spaces was stored trimmed.
- **1Password's overlay** on the New Questionnaire Name field blocked the extension's typing ("Cannot access a chrome-extension:// URL"). I worked around it with form input. This is a tool and browser-profile artefact, not an app bug.

## Console and network errors not tied to a case

- No console errors or warnings (checked on Trust A's page after a reload).
- **Tool artefact, as in reports 2 and 3:** the extension recorded these requests as **503**:
  - every PATCH (Retire, Bring back, Edit);
  - `POST /api/logout`;
  - `POST /api/dev/login`.

  Each one worked in the UI (messages shown, rows changed, sign-out and sign-in happened). This isn't an app bug.
- **The only real 4xx were the expected ones:** the POST 400s in T-03, T-16 and T-18.

## GIFs

- `~/Downloads/S33-T16-T02-retest4.gif`: T-16 (capital-letter leaflet and video, the Vimeo edit, the HTTP refusals) and the T-03 refusals, then T-02 (add a video with a typed tag and **+ Recovery**, Recovery ticked).
- `~/Downloads/S33-T13-retest4.gif`: T-13 (Sam retires "Medicines you take regularly", it moves last with **Offer again**; then Offer again).
- `~/Downloads/S33-T17-retest4.gif`: T-17 (the refused Retire with the red message, Edit → Cancel and Add video → Cancel with no old message, then the real Retire).
