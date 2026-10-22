# S04 manual test plan: the web app on React + Vite (Next.js removed)

**Changed after verification 1 (12 Oct):** a page whose code can't be downloaded (e.g. after a deploy, while the app is open) used to show a raw "Unexpected Application Error!" screen. Now the app reloads once by itself; if that doesn't help, it shows **Something went wrong** with **Reload** and **Go to the start page**.
- **New case T-19** checks it.
- **For this retest, run:** T-19; T-01, T-08 and T-15 (the app's start-up and every address, because the root route changed); T-12 at phone size; R-01.

**Changed after test report 1 (12 Oct):** no code changed. Corrections to the plan:
- **T-16 step 4 was wrong.** It expected Back to return a long list to where you'd scrolled. The Next.js app didn't do that either: I ran the pre-S04 build beside this one, and both open the Episodes list at the top after Back (the rows load after the page appears, so there's nothing to scroll to yet). Step 4 now expects the top, the same as before S04.
- **T-05 step 3:** a new Question Set has no page, so click **Add Page** first; **Add content:** appears after that.
- **T-06 step 2:** new questions are already **Required**; check that the tick is on, rather than ticking it.
- **T-14 step 3:** the Patient answers tab shows what the patient's **Yes** produces (the disclosure, e.g. "Patient smokes"), not the word "Yes".
- **For this retest, run:** T-16 (all of it), and T-14 step 5 only if you can (Rahul may check Print by hand instead). Everything else passed in report 1.

**What changed in S04:**
- **The whole web app was moved from Next.js to React + Vite + React Router.** Every screen's code was moved, so **this is a regression test of every screen**. Nothing should look or behave differently, with two exceptions:
  - **The name:** the app now says **Poised** where it showed the old name (the browser tab title, the header beside the hospital picker, the sign-in page's heading, and the patient page's dark top bar).
  - **The "not found" page** is the app's own: a heading **404** and the text **"This page could not be found."** Next.js showed nearly the same text.
- **The URLs are all the same as before.** Every address in this plan should open the same screen, including after a reload or when pasted into a new tab.
- **Gone:** the small Next.js "N" tools button and the "Issues" badge at the bottom left in development. Errors now appear only in the browser console.

**How to find the IDs in addresses:** open the screen from the app, then copy its address from the address bar. Hospital A's ID is always `00000000-0000-4000-8000-00000000000a`, and Hospital B's is `00000000-0000-4000-8000-00000000000b`.

## 1. Setup

**Rahul runs these in his own terminal before testing, and leaves the last one running.** If `make dev` was already running, he stops it (Ctrl-C) and starts it again, because the web app has changed completely. Run `npm install` once first, because the packages changed.

```sh
npm install
docker compose up -d db
make migrate seed
make dev
```

- **The app:** http://localhost:3000. If it doesn't load, stop and report it. Don't try to start or fix the servers.
- **Sign-in:** the sign-in page lists the users; click a name. There's no password. **Sign out** is at the top right (in the editor, it's under the round initials button).

  | User | Hospital | Can do |
  | --- | --- | --- |
  | Alex Author | Hospital A | write and publish questionnaires; not a clinician |
  | Val Viewer | Hospital A | see questionnaires, read only |
  | Cara Clinician | Hospital A | patients, episodes, validation, POA Summary |
  | Bea Author | Hospital B | Hospital B only |

- **Data rules:**
  - Use made-up text only. Never enter a real person's details.
  - Name what you create `S04 Test<time>`, e.g. the questionnaire `S04 Test1405`, and a patient with first name `S04` and last name `Test1405`.
  - Don't change questionnaires or episodes you didn't create. The database may already hold "E2E …" and "Sweep …" questionnaires from automated tests; leave them alone.
- **Terms:**
  - an **HQ** is a health questionnaire;
  - a **Question Set** is a section of it;
  - an **episode** is one patient's HQ, from the link to the POA Summary.
- **Phone size:** "phone size" means a 390 × 844 viewport. Use DevTools' device toolbar (iPhone 12 Pro or 14). If that doesn't work with your tools, a 390 × 844 same-origin iframe is acceptable; say so in the report.
- **The patient's link:** open it in a private window, or sign every staff user out first.
- **Never clear site storage** (local or session storage, cookies) for `localhost:3000`. Other local apps on this machine keep data there.
- **Console and network:** keep DevTools open on every screen, and report any red console error or yellow warning with the case ID.
  - **Expected, not errors:**
    - a `401 (Unauthorized)` from `/api/me` when you open the app while signed out;
    - a `404` or `403` that a case causes on purpose (it says so).
  - **Known tool quirk:** the Claude in Chrome network log shows every `204 No Content` as **503**. Ignore a 503 on sign-out, delete, publish, submit or complete review.

## 2. Test cases

Do them in order: T-04 to T-09 make the questionnaire that T-10 to T-15 use.

### T-01: the app is the new one, and the name is Poised

1. Open http://localhost:3000/login while signed out.
2. Look at the browser tab's title.
   - **Expected:** **Poised**, with the blue tick icon.
3. Look at the page.
   - **Expected:** the heading **Poised**, the text "Development sign-in: choose a user.", and a card for each seeded user with their email address.
4. View the page source (right-click → View Page Source, or `view-source:http://localhost:3000/login`).
   - **Expected:** a short page with `<div id="root">` and a script `/src/main.tsx`. **No** `_next` anywhere in it.
5. Check that the old name appears nowhere on screen, in T-01 to T-15. (It may appear in questionnaire *content* someone wrote; that's fine.)

### T-02: sign-in and the start page

1. On the sign-in page, click **Alex Author**.
   - **Expected:** you land on `/h/<Hospital A>/questionnaires`, the Questionnaires list. The header shows **Poised**, the hospital picker showing **Hospital A**, a **Questionnaires** button (highlighted), "Alex Author" and **Sign out**. There's no **Episodes** button (Alex isn't a clinician).
2. Open http://localhost:3000/ (the bare address).
   - **Expected:** a short spinner, then the Questionnaires list again.
3. Click **Sign out**.
   - **Expected:** the sign-in page.
4. Sign in as **Cara Clinician**.
   - **Expected:** you land on `/h/<Hospital A>/episodes`. The header has **Episodes** (highlighted) and no **Questionnaires**.
5. Sign out, then open http://localhost:3000/ while signed out.
   - **Expected:** the sign-in page. (One 401 from `/api/me` in the console is expected.)

### T-03: a signed-out deep link goes to sign-in

1. Signed out, open `http://localhost:3000/h/00000000-0000-4000-8000-00000000000a/questionnaires`.
   - **Expected:** a short spinner, then the sign-in page (`/login`). No hospital data shows on the way.

### T-04: questionnaires list and a new questionnaire

1. Sign in as **Alex Author**.
2. On the Questionnaires list, type part of an existing name into the search box.
   - **Expected:** the list narrows to matching rows. Clear the search.
3. Click **New Questionnaire**, name it `S04 Test<time>`, give it any description, and create it.
   - **Expected:** the editor opens at `/h/<Hospital A>/questionnaires/<id>`, full screen (no Poised header bar; the editor has its own header with a **Questionnaires** link). The badge says **v1 · draft**.
4. Click the **Questionnaires** link at the top left.
   - **Expected:** back on the list; your questionnaire is in it, as a draft. Click its row.
   - **Expected:** the editor opens again.

### T-05: the editor's Structure tree and canvas

In your `S04 Test<time>` draft:
1. Under **Structure** on the left, click **Add Question Set**.
   - **Expected:** "Question Set 1" appears and opens on the right. The address gains `?set=<id>`.
2. Rename it to `About you` (use the pencil next to its name on the canvas, then confirm).
3. Click **Add Page**. Then, using **Add content:**, add a **Yes/No** question. Set its text to `Do you smoke?`.
4. Add a second Yes/No question: `Do you vape?`.
5. Wait a moment, then **reload the page**.
   - **Expected:** the editor comes back with **About you** still open (the address still has `?set=…`), and both questions are there.
6. Add a second Question Set and name it `Your health`. Click between the two in the Structure tree.
   - **Expected:** the canvas switches each time, and the address's `?set=` changes to match.
7. Use the browser's **Back** button once.
   - **Expected:** you leave the editor (to the Questionnaires list or wherever you were before it opened). Clicking between Question Sets doesn't add Back steps; that's how it worked before.
8. Use **Forward**.
   - **Expected:** the editor again, on the Question Set you last had open.

### T-06: Settings and Logic

1. In **About you**, click the `Do you vape?` card.
   - **Expected:** the left column shows the **Settings | Disclosures | Logic** tabs for it.
2. In **Settings**, check that **Required** is ticked (new questions start as Required). Leave it ticked.
3. Open the **Logic** tab. In the **Builder**, make the question show only when `Do you smoke?` is **Yes**, and save it.
   - **Expected:** the `Do you vape?` card gets an orange left border (it's conditional). The **Code** tab shows the condition as an expression mentioning the smoking question.
4. Reload the page.
   - **Expected:** the condition and the Required tick are still there.
5. Close the settings with the panel's close button.

### T-07: Disclosures

1. Click the `Do you smoke?` card, and open the **Disclosures** tab.
2. Add a disclosure for the **Yes** answer, with any code you can pick from the code picker (e.g. search "smok").
   - **Expected:** the disclosure is listed. The card shows a green link chip with **1**.
3. Reload the page.
   - **Expected:** the chip and the disclosure are still there.

### T-08: Preview and test cases

1. With **About you** open, click **Preview** in the editor's header.
   - **Expected:** the preview opens at `/h/<Hospital A>/chapters/<id>/preview`, full width, with the Poised header bar back at the top. It shows the questions as a patient sees them.
2. Answer `Do you smoke?` **Yes**.
   - **Expected:** `Do you vape?` appears. The clinical outputs show the disclosure you added in T-07.
3. Switch the **Patient / Clinician** view a few times, and change the screen size.
   - **Expected:** the layout changes as before, with no console errors.
4. Under **Test cases**, save the current answers as a test case named `Smoker`, then run all test cases.
   - **Expected:** `Smoker` passes.
5. **Reload** the preview.
   - **Expected:** the same preview comes back, with the `Smoker` test case listed.
6. Click **Edit** in the preview.
   - **Expected:** the editor opens with **About you** selected (`?set=` in the address).
7. Click **Preview** again, then use the browser's **Back** button.
   - **Expected:** back in the editor, on **About you**.

### T-09: publish and versions

1. In the editor, click **Publish** and confirm.
   - **Expected:** the badge becomes **v1 · published**, a **Read only** badge appears, and editing is no longer possible.
2. Click **Create new version**.
   - **Expected:** the badge says **v2 · draft**, and the content is the same as v1. The Structure tree starts with nothing selected.
3. Go back to the Questionnaires list.
   - **Expected:** your questionnaire shows its new version and status as before.
4. Sign out, sign in as **Val Viewer**, and open your questionnaire from the list.
   - **Expected:** it opens **Read only**, and Val can't change it.

### T-10: episodes list and a new episode

1. Sign out, and sign in as **Cara Clinician**.
   - **Expected:** the Episodes list.
2. Use the status filter (**All statuses**), and pick a status, then clear it.
   - **Expected:** the list filters, as before.
3. Click **New episode**. Create a new patient `S04 Test<time>` (made-up details: any date of birth, any sex) on **your** published questionnaire `S04 Test<time>` (version 1), with any procedure, anaesthetic and consultant. Click **Create episode**.
   - **Expected:** the episode page opens at `/h/<Hospital A>/episodes/<id>`.
4. Click **Episodes** (the link at the top) to go back to the list, then click your episode's row.
   - **Expected:** the episode page opens again.

### T-11: the episode page

1. On your episode's page, find the **Patient link** field.
   - **Expected:** it starts with `http://localhost:3000/p/`. Copy it for T-12.
2. Check the Question Set list.
   - **Expected:** **About you** and **Your health** are listed, with nothing answered yet.
3. Reload the page.
   - **Expected:** the same episode, the same link.

### T-12: the patient's link, at phone size (390 × 844)

In a private window (or with all staff signed out), at phone size:
1. Open the patient link from T-11.
   - **Expected:** the patient page, with a dark top bar saying **Poised**, the patient's first name, and the Question Sets to answer. Nothing is cut off at 390 px wide; there's no sideways scrolling.
2. Open **About you**. Answer `Do you smoke?` **Yes**.
   - **Expected:** `Do you vape?` appears. Answer it **No**. The page shows the answers saved.
3. **Reload** the page.
   - **Expected:** the same patient page comes back, and your answers are kept.
4. Go back to the section list, answer **Your health** if it has questions, then click **Send my answers** and confirm.
   - **Expected:** a thank-you message.
5. Reload.
   - **Expected:** still the thank-you message.
6. Change the last few characters of the link and open it.
   - **Expected:** a red error message on the patient page (as before S04), **not** the 404 page and not a blank page. (A 404 for `/api/p/…` in the console is expected.)

### T-13: validating a Question Set

As **Cara Clinician**, at desktop size:
1. Open your episode again (reload it if it was open).
   - **Expected:** it shows the patient has sent their answers.
2. Click the button to validate **About you**.
   - **Expected:** the validate screen opens at `/h/<Hospital A>/episodes/<eid>/sets/<cid>`, with the patient's answers.
3. **Reload** this screen.
   - **Expected:** the same screen and answers.
4. Change `Do you smoke?` to **No**, and click **Validate**.
   - **Expected:** you return to the episode page, and **About you** shows as validated.
5. Validate **Your health** the same way (no changes), then use the browser's **Back** button.
   - **Expected:** Back goes to the previous screen, not to an error page.
6. On the episode page, click **Complete HQ review** if it's offered.

### T-14: the POA Summary

1. On the episode page, click **POA Summary**.
   - **Expected:** the POA Summary at `/h/<Hospital A>/episodes/<eid>/poa`, with **Episodes** and **Episode** links at the top, the patient's name, and the tabs **Validated summary** (selected) and **Patient answers**.
2. In **Validated summary**, look at **About you**.
   - **Expected:** the smoking answer is the clinician's **No**, marked as changed by the clinician.
3. Click the **Patient answers** tab.
   - **Expected:** the patient's original answer (what their **Yes** produces, e.g. the disclosure "Patient smokes"), with no change marked.
4. Change the episode's status (the **Status** picker), if it's offered here or on the episode page.
   - **Expected:** an event is added under **General notes**.
5. Click **Print**.
   - **Expected:** the browser's print dialog opens, and the preview has no Print button or other screen-only controls in it. Cancel it.
6. **Reload** the POA Summary.
   - **Expected:** the same summary.
7. Click the **Episode** link, then **Back**, then the **Episodes** link.
   - **Expected:** each goes where it says; Back returns to the POA Summary.

### T-15: every address reloads, and works pasted into a new tab

For each address below (use the IDs from your own data), (a) reload it, and (b) paste it into a **new tab** in the same window. Use the user given.

| Address | User | Expected |
| --- | --- | --- |
| `/login` | signed out | sign-in page |
| `/h/<A>/questionnaires` | Alex | Questionnaires list |
| `/h/<A>/questionnaires/<qid>?set=<cid>` | Alex | editor, with that Question Set open |
| `/h/<A>/chapters/<cid>/preview` | Alex | preview of that Question Set |
| `/h/<A>/episodes` | Cara | Episodes list |
| `/h/<A>/episodes/<eid>` | Cara | the episode |
| `/h/<A>/episodes/<eid>/sets/<cid>` | Cara | validate screen |
| `/h/<A>/episodes/<eid>/poa` | Cara | POA Summary |
| `/p/<token>` | private window, phone size | patient page |

- **Expected for every row:** the screen opens directly, with no 404, no blank page, no sign-in prompt (except signed out), and no console errors.

### T-16: Back, Forward and scrolling

As **Cara Clinician**:
1. Episodes list → your episode → **POA Summary**.
2. Press **Back** twice.
   - **Expected:** the episode, then the Episodes list.
3. Press **Forward** twice.
   - **Expected:** the episode, then the POA Summary.
4. On a long page (the Episodes list as Cara), scroll down, open an episode from it, then press **Back**.
   - **Expected:** the newly opened episode starts at the top (not part-way down). After Back, the list opens **at the top** too. That's how it worked before S04 (the list's rows load after the page appears); don't report it.
5. On the episode, scroll down a little, click **POA Summary**, then press **Back**.
   - **Expected:** the POA Summary starts at the top; Back shows the episode, and nothing breaks.

### T-17: the not-found page

Signed in as **Alex Author**, open each of these:
1. `http://localhost:3000/nope`
2. `http://localhost:3000/h/00000000-0000-4000-8000-00000000000a` (the hospital alone; it was never a page)
3. `http://localhost:3000/h/00000000-0000-4000-8000-00000000000a/nope`
4. `http://localhost:3000/h/00000000-0000-4000-8000-00000000000a/episodes/anything/nope`
5. `http://localhost:3000/p` (no token)
   - **Expected for each:** a plain page with a heading **404** and **"This page could not be found."**, the tab title **Poised**, and no console errors. Reloading shows the same.
6. Press **Back**.
   - **Expected:** the previous page.

### T-18: refused paths (another hospital, wrong role)

1. Sign in as **Bea Author** (Hospital B).
   - **Expected:** Bea lands on Hospital B's Questionnaires list. The hospital picker shows only **Hospital B**.
2. As Bea, open `http://localhost:3000/h/00000000-0000-4000-8000-00000000000a/questionnaires`.
   - **Expected:** "You don't have access to this hospital." No Hospital A data shows.
3. As Bea, open your questionnaire's editor address from Hospital A (`/h/<A>/questionnaires/<qid>`).
   - **Expected:** "You don't have access to this hospital."
4. As Bea, open your episode's POA Summary address (`/h/<A>/episodes/<eid>/poa`).
   - **Expected:** "You don't have access to this hospital." No patient details show.
5. Sign in as **Alex Author** (not a clinician) and open `/h/<A>/episodes/<eid>`.
   - **Expected:** no patient details show; an error message (as before S04). A 403 in the console is expected.

### T-19: a page whose code can't be downloaded (new after verification 1)

This needs Chrome DevTools' request blocking. In current Chrome it's DevTools → **Request conditions** (older Chrome: ⋮ → More tools → Network request blocking): add the rule `*Editor*` set to block, and tick **Disable cache** in the Network tab. If your tools can't block a request, mark T-19 **Blocked** and say so; Rahul will check it by hand.

1. Sign in as **Alex Author**, and open the Questionnaires list in a **new tab** (so the editor's code hasn't been loaded in this tab yet).
2. Turn on the `*Editor*` blocking rule (and **Disable cache**), then reload the Questionnaires list.
3. Click any questionnaire in the list.
   - **Expected:** the page reloads by itself once (a short spinner), then shows **Something went wrong**, "This page couldn't be loaded. Check your connection, then reload.", a **Reload** button and a **Go to the start page** link. It must **not** keep reloading, and it must not show "Unexpected Application Error!". (Console errors about the blocked file are expected.)
4. Turn the blocking rule off, then click **Reload**.
   - **Expected:** the editor opens for that questionnaire.
5. Block `*Editor*` again, go back to the Questionnaires list, wait 15 seconds, then click a questionnaire again.
   - **Expected:** the same as step 3: one reload, then the message.
6. Click **Go to the start page**.
   - **Expected:** a short spinner, then the Questionnaires list. Turn request blocking off when done.

## 3. Regression (earlier sessions)

These re-check behaviour from earlier sessions that runs through the moved screens.

- **R-01 (S02a): two editors on one Question Set.** As Alex, open your v2 draft's **About you** in two tabs. Change a question's text in tab A and wait. Then change something in tab B.
  - **Expected:** tab B shows **"Someone else changed this Question Set"** with **Reload**. Reload shows tab A's change.
- **R-02 (S02a): two tabs validating the same Question Set.** Make a second episode (T-10, T-12) and have the patient send their answers. As Cara, open its **About you** validate screen in two tabs. Change an answer in tab A and wait for it to save. Then change an answer in tab B.
  - **Expected:** tab B says the answers were changed somewhere else, with **Reload**. Nothing tab B sent is saved.
- **R-03 (S02a): the patient's link on two "devices".** Open a fresh patient link in two phone-size tabs. Answer in tab A and wait for it to save. Then change an answer in tab B.
  - **Expected:** tab B shows "Your answers weren't saved" with **Reload**.
- **R-04 (S01): a duplicate questionnaire name is refused.** As Alex, create a questionnaire with exactly the same name as your `S04 Test<time>`.
  - **Expected:** a clear message that the name is taken (a 409 in the console is expected); nothing is created.
- **R-05 (S01): delete a draft.** As Alex, make a throwaway draft `S04 Delete<time>` and delete it from the list (**Delete questionnaire?** → confirm).
  - **Expected:** it disappears from the list.

## 4. Out of scope (don't report these)

- **Looks:** colours, spacing and layout that were the same before S04. Only report a screen that looks *different* from before, or broken.
- **Hospital switching between two hospitals:** no seeded user belongs to two hospitals, so the picker can't switch. Trusts and multi-hospital staff come in S08.
- **New features or URLs:** `/o/…` addresses (S08), Amplify and AWS (S06).
- **The questionnaire content itself** (e.g. gaps in HJE Full HQ).
- **Speed:** a short blank moment the first time each screen opens is expected (each screen's code is loaded when first opened). Report only a screen that stays blank.
- **The old name in code comments** or in docs; only on-screen text counts (T-01).
