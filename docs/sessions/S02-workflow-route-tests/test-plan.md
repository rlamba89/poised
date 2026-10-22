# S02 manual test plan: episode workflow regression

**Changed after test report 1 (8 Oct):**
- **`make-test-hq.js` now makes its Yes / No questions as the editor does** (Yes and No options). Report 1's F-1 ("Patient answered: false") came from the script using a question type the app never makes. It wasn't an app bug. Re-run T-01 to make a new HQ, and use it for every case.
- **T-07 and T-09** give the exact text expected for the correction.
- **T-12 step 2 and R-04** expect the app's real message for another hospital's address: "You don't have access to this hospital."
- **Setup:** how to test at phone size and "come back" without a private window, and **don't clear site storage**.

**What changed in S02:** very little you can see.
- S02 added automated API → database tests for the episode, patient-link, clinician and sign-in routes.
- **One server bug was fixed:** changing an episode with a field set to `null` crashed the request, and the browser got no answer at all. It's now refused with a message (T-13).
- **Any unexpected server crash** now gets the plain-language "Something went wrong" message instead of a dropped connection. You can't trigger this from the browser.

This plan re-checks the whole episode workflow end to end, especially that **a patient never sees clinician-only content**.

## 1. Setup

**Rahul runs these in his own terminal before testing, and leaves the last one running:**

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
  | Val Viewer | Hospital A | read questionnaires only |
  | Cara Clinician | Hospital A | patients, episodes, validation, POA Summary |
  | Bea Author | Hospital B | write questionnaires in Hospital B only |

- **Seeded patients** (made up, all in Hospital A): Jo Bloggs, Pat Example, Robin Sample.
- **Data rules:**
  - Use made-up text only. Never enter a real person's details.
  - Name new patients `S02 Test<time>`, e.g. first name `S02`, last name `Test1405`.
  - Don't change questionnaires or episodes you didn't create.
- **Terms:**
  - an **HQ** is a health questionnaire;
  - a **Question Set** is a section of it;
  - an **episode** is one patient's HQ, from the link to the POA Summary.
- **The test HQ:** T-01 creates it with a script, because building it in the editor would take a long time. Every question a patient must never see has **STAFF ONLY** in its title, and the "Yes" answer to "Do you smoke?" carries the clinician note **Current smoker**. A patient must never see either.
- **Phone size:** "phone size" means a 390 × 844 viewport. Use DevTools' device toolbar (iPhone 12 Pro or 14), or resize the window. If neither works with your tools, a 390 × 844 same-origin iframe is acceptable; say so in the report.
- **No private window?** Sign every staff user out, and open the patient link in a new tab. To "leave and come back", close the tab and open the link in another new tab.
- **Never clear site storage** (local or session storage, cookies) for `localhost:3000`. Other local apps on this machine keep data there.
- **Console:** keep DevTools open and report any red error with the case ID. These are expected, not errors:
  - two `401 (Unauthorized)` on the sign-in page;
  - `404` responses in T-11 and T-12 (refusals being tested);
  - a `400 (Bad Request)` in T-13 (the fix being tested).
- **Known tool quirk:** the Claude in Chrome network log shows every `204 No Content` response as **503**. Ignore a 503 on a request that the app treats as a success (an autosave, submit, validate, complete review or sign-out). Other codes are reported correctly.

## 2. Test cases

### T-01: Make the test HQ
1. Sign in as **Alex Author**. You're on Hospital A's questionnaire list.
2. Open DevTools → **Console**. Paste the whole contents of `docs/sessions/S02-workflow-route-tests/make-test-hq.js` and press Enter. If Chrome asks you to type "allow pasting" first, do that, then paste again.
3. **Expected:** the console prints `Created and published: S02 test HQ <hh:mm>`. Note that name: the later cases use it.
4. Reload the list. **Expected:** the HQ is listed as published (v1).
5. Open it. **Expected:**
   - three Question Sets: **About you**, **Your health** and **Nurse checks**;
   - it's read-only, with "v1 · published".
6. Sign out.

### T-02: Create an episode for an existing patient
1. Sign in as **Cara Clinician**. **Expected:** you land on **Episodes**.
2. Click **New episode**. Choose **Existing patient**, then **Jo Bloggs**.
3. Open **Health questionnaire**. **Expected:** your S02 test HQ is listed. Questionnaires that are still drafts are not.
4. Choose it. Enter the procedure `Knee replacement` and the consultant `Dr S02`. Click **Create episode**.
5. **Expected:**
   - the episode page opens, with status **HQ not complete**;
   - Procedure "Knee replacement" and Consultant "Dr S02";
   - a **Patient link** with a **Copy** button;
   - the **Episode status** dropdown is disabled;
   - **General notes** say "Episode created", by Cara Clinician, with the time.
6. Click **Copy**, and keep the link (paste it somewhere, such as a note). Note the episode page's address too.

### T-03: Create an episode for a new patient, with refusals
1. On **Episodes**, click **New episode** and choose **New patient**.
2. Enter first name `S02`, last name `Test<time>`, and a date of birth **in the future**. Choose any sex and your S02 test HQ. Click **Create episode**.
3. **Expected:** it's refused with a plain-language message, such as "Enter a valid date of birth.", and the drawer stays open.
4. Correct the date of birth (e.g. 1980-05-01) and click **Create episode**.
5. **Expected:** the episode page opens for S02 Test<time>, with status **HQ not complete**.
6. Back on **Episodes**, the list shows both episodes. **Expected:** the status filter **Ready for review** shows neither of them.

### T-04: The patient fills in part of the HQ at phone size
1. Open a **new private (incognito) window**, so you're not signed in. Set it to phone size (390 × 844).
2. Open the patient link from T-02.
3. **Expected:**
   - a greeting for Jo;
   - only **About you** and **Your health** are listed. **Nurse checks** never appears;
   - nothing needs sideways scrolling.
4. Open **About you**.
5. **Expected:** "Do you smoke?" and "Do you have angina?" are shown.
   - **No text containing STAFF ONLY appears anywhere**, on any screen of this set.
   - "Current smoker" never appears, even after you choose an answer.
6. Answer **Yes** to smoking. Go to the end of this section and finish it, answering **No** to angina.
7. **Expected:** you're back on the list of Question Sets. About you is marked done, and Your health isn't.
8. Use the browser's **Ctrl+F** (Cmd+F) to search the page for `STAFF`. **Expected:** no match.

### T-05: Leave and come back, then submit (phone size)
1. Close the private window. Open a **new** private window at phone size, and open the same patient link.
2. **Expected:** About you is still marked done. Open it: **Yes** and **No** are still selected.
3. Go back to the list and open **Your health**. Answer **No** to diabetes, then finish the section.
4. **Expected:** **Send my answers** is now enabled. It was disabled while a set was unanswered.
5. Click **Send my answers**, then confirm in the dialog.
6. **Expected:** a thank-you page.
7. Reload. **Expected:** the thank-you page again, not the form. The answers can't be changed.
8. Search the thank-you page and the list for `STAFF` and `Current smoker`. **Expected:** no match.

### T-06: A bad patient link (phone size)
1. In the private window, change the last few characters of the patient link and open it.
2. **Expected:** a plain-language message, such as "This link doesn't work. Please check it, or contact the hospital." No questions are shown.
3. Open `http://localhost:3000/p/abc`. **Expected:** the same message.

### T-07: The clinician validates, correcting an answer
1. In the normal window, as **Cara**, open the T-02 episode (from **Episodes**, or its address).
2. **Expected:**
   - the status is **Ready for review**;
   - all three Question Sets are listed. None is validated yet;
   - **Complete HQ review** is disabled, with the hint "Validate every Question Set to complete the HQ review."
3. Click **Validate** on **About you**. **Expected:**
   - it's pre-filled with Jo's answers (Yes to smoking, No to angina);
   - the clinician-only questions (STAFF ONLY: cessation advice, ECG reading, the STAFF ONLY page) **are** shown here, to the clinician;
   - **Current smoker** appears in the Lifestyle summary box.
4. Change angina to **Yes**. **Expected:** "Patient answered: No" appears under it (not "false").
5. Fill in the STAFF ONLY questions with any made-up text, then click **Validate**.
6. **Expected:** back on the episode, About you shows "Validated by Cara Clinician on <date and time>".
7. Validate **Your health** without changing anything.
8. Open **Nurse checks**. **Expected:** only the clinician's questions; there are no patient answers. Fill in the blood pressure and **Validate**.
9. **Expected:** all three show "Validated by Cara Clinician on …", and **Complete HQ review** is enabled.

### T-08: Complete the review; then it's read-only
1. Click **Complete HQ review**.
2. **Expected:** the status is **Ready for POA**.
3. Open **About you** again. **Expected:**
   - it's read-only, with "The HQ review is complete, so these answers can't be changed.";
   - there's no **Validate** button.
4. Reload the episode page. **Expected:** the same status and stamps.
5. Press the browser's **Back** button, then **Forward**. **Expected:** each page shows correctly, with no error.

### T-09: POA Summary, both tabs, and notes
1. On the episode, click **POA Summary**.
2. On **Validated summary**, **expected:**
   - the patient details: Jo Bloggs, TEST0001, Procedure Knee replacement, Consultant Dr S02;
   - **General notes** list, with times: "Episode created" (Cara Clinician), "HQ completed by the patient" (no user), and "HQ review completed by Cara Clinician";
   - **About you** shows "Current smoker". The angina change is listed as **Do you have angina? Patient answered "No", now "Yes"**.
3. Click the **Patient answers** tab. **Expected:**
   - "Current smoker" is shown;
   - there's no correction, and nothing the clinician typed.
4. Click **Add comment**, type `S02 note one`, and click **Save comment**. **Expected:** it appears in General notes with Cara's name and the time.
5. Add a second comment, `S02 note two`. **Expected:** both comments are listed in order, the first one unchanged. There's no way to edit or delete a note.
6. Try **Save comment** with only spaces. **Expected:** the button stays disabled, or it's refused. No empty note appears.
7. Reload. **Expected:** both comments are still there.
8. Set **Nurse ASA grade** to **ASA II** and click its **Save**. **Expected:** the details show ASA II, and a note says "Nurse ASA grade set to 2".
9. Change **Episode status** to **Ready for admission**. **Expected:** the status changes, and a note "Status changed to Ready for admission" is added.
10. **Expected:** the **Print** button exists. (You needn't print.)

### T-10: Status can't be changed by hand before the review
1. Open the T-03 episode (S02 Test<time>, still **HQ not complete**).
2. **Expected:** the **Episode status** dropdown is disabled, on the episode page and on its POA Summary. The status stays **HQ not complete**.
3. Open its **POA Summary**. **Expected:** it opens with no error, and an orange note says "The HQ review isn't complete yet…".

### T-11: Wrong role: non-clinicians can't see episodes
1. Sign out, and sign in as **Alex Author**. **Expected:** there's no **Episodes** link in the header.
2. Paste the T-02 episode page's address into the address bar.
3. **Expected:** a refusal, such as "Only clinicians can see patients and episodes." No patient details are shown.
4. Repeat steps 2–3 as **Val Viewer**. **Expected:** the same.

### T-12: Another hospital can't see Hospital A's episode
1. Sign out, and sign in as **Bea Author** (Hospital B).
2. Paste the T-02 episode page's address. **Expected:** "You don't have access to this hospital.", and no patient details.
3. In that address, replace Hospital A's id (the part after `/h/`) with Hospital B's id from Bea's own address. Open it.
4. **Expected:** a refusal ("not found", or "only clinicians"). No patient details are shown.

### T-13: A broken status change is refused, not dropped (the S02 fix)
1. Sign in as **Cara Clinician**, and open the T-02 episode page.
2. In DevTools → **Console**, run this, with the episode page's address still open:
   ```js
   const [, , hid, , eid] = location.pathname.split("/");
   const r = await fetch(`/api/h/${hid}/episodes/${eid}`, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: '{"status": null}' });
   console.log(r.status, await r.text());
   ```
3. **Expected:** `400 {"error":"The request could not be read."}`. Before S02, this request got no response at all.
4. Run it again with `'{"procedure": null}'` in place of `'{"status": null}'`. **Expected:** the same 400.
5. Reload the page. **Expected:** the status and procedure are unchanged: Ready for admission, Knee replacement.

## 3. Regression (earlier sessions)

### R-01: Sign in and out (S01)
Sign in as each of the four users in turn. **Expected:** each lands on their own start page with their name in the header, and **Sign out** returns to the sign-in page.

### R-02: Create and save a questionnaire (S01)
As **Alex Author**, click **New Questionnaire**, name it `S02 regression <time>`, and save. Add a Question Set and a page with one Yes / No question. Wait for "All changes saved", then reload. **Expected:** it's all kept. Then delete this draft (⋯ → Delete → confirm).

### R-03: A viewer can't edit (S01)
As **Val Viewer**, open any questionnaire. **Expected:** it opens read-only, and there's no **New Questionnaire** button.

### R-04: Hospital scope for questionnaires (S01)
As **Bea Author**, paste the address of your S02 test HQ (Hospital A). **Expected:** "You don't have access to this hospital.", and nothing from Hospital A is shown.

## 4. Out of scope (don't report)

- **Already recorded from report 1** (in `docs/pending.md` §3; older than S02):
  - a double-click on **Save comment** saves the note twice;
  - the patient link's response includes the date of birth and other details the page doesn't use;
  - **Sign out** occasionally seeming not to sign out (unconfirmed);
  - refusal pages without a header, a blank hospital picker, "No episodes yet." under a filter, "Saved" before any answer, the delete dialog's flicker;
  - an unknown status gets 409 rather than 400;
  - the ASA grade controls work before the review is complete.

- **Things the server leaves to the browser, by design** (decisions.md, 8 Oct):
  - the server would accept "Complete HQ review" before every set is validated, if called directly. The button's rule is the browser's;
  - the clinician view is pre-filled from the patient's answers in the browser.
- **The leftover published "S02 test HQ …"** in Hospital A: published questionnaires can't be deleted.
- **Known gaps listed in `docs/pending.md`**, such as:
  - no patient accounts, search or invitations;
  - a review can't be reopened;
  - the POA Summary lacks Audit log, Work plan, Files and similar sections;
  - "chapter" in some user-facing messages.
- **Branding:** the UI still shows the old name in places.
- **Looks:** spacing, colours and wording, unless something is unreadable or overlaps at phone size.
- **A new patient created when the episode then fails:** if T-03's first try created a patient before refusing, that's older behaviour, not S02's. Note it, but don't fail the case for it.
