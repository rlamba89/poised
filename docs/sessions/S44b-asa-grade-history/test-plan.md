# S44b manual test plan: the anaesthetist's ASA grade as a history

*Changed after verification 1 (28 Oct): a reason is now required for **every** grade after the episode's first, even after the earlier one was entered in error (T-08 step 4, T-13); the database also refuses deleting a grade.*

*Changed after test report 1 (28 Oct): T-08 and T-13 check "No ASA grades have been added." no longer shows above a struck-through grade; T-14 checks the Entered in error button isn't cut off; new T-15 (the same grade again reads "confirmed as").*

## What changed

The **anaesthetist's ASA grade** on the POA Summary used to be a dropdown with **Save** that could be changed or cleared at any time. Now (Rahul, 27 Oct) **a grade, once given, is never changed or cleared, in any status**:

- **Add grade** adds a grade to a history. The episode's first grade needs no reason. Every later one must say why, even after the earlier ones were entered in error ("Why is the grade changed?", e.g. "New echo result"). The earlier grade stays in the list.
- **The grade that counts** is the latest one not entered in error. It shows as "Anaesthetist ASA grade: ASA …" above the history, with a **Counts** badge on its entry, and in the details at the top of the POA Summary.
- **Entered in error:** each standing grade has a button to mark it entered in error, with a reason. It stays in the list, **struck through**, with "Entered in error: marked by <who> on <when>: <why>". It no longer counts, and the one before it (if any) counts again. A mark can't be taken off.
- **The rule's requirement** (S43: High risk requires the anaesthetist's grade before Ready for admission) reads the grade that counts. At **Ready for admission, Admitted or Discharged**, while a rule requires the grade, the **only** grade that counts can't be marked entered in error: "Add the correct grade first, or move the episode back from Ready for admission."
- There is no way to edit or delete a grade, in the app or the API. The episode's save (PATCH) refuses the anaesthetist's grade. **The nurse's ASA grade is unchanged** (a dropdown with Save, which can be changed or cleared).
- The General notes get a line for each: "Anaesthetist ASA grade set to 3", "Anaesthetist ASA grade changed from 3 to 4: New echo result", "Anaesthetist ASA grade 4 marked entered in error: Wrong patient".

## Setup

**In your own terminal** (the conductor does this in an automatic run):

```sh
docker compose up -d db
make migrate seed
make dev
```

Leave `make dev` running. Open **http://localhost:3000**. Sign in on the stub login page by clicking a name (sign out from the top right to switch):

| User | Role | Where |
| --- | --- | --- |
| **Alex Author** | super clinician | the whole of Trust A (writes and publishes HQs) |
| **Cara Clinician** | clinician | Hospital A1 only |
| **Sue Super** | super clinician | Hospital A1 only (archives) |
| **Val Clinician** | clinician | Hospital A2 only |
| **Bea Author** | super clinician | the whole of Trust B |

- **Captured messages:** texts and emails aren't sent; they're listed at **http://localhost:3000/qa/messages** (newest first).
- **The patient's link:** a new episode with an email sends an invite; open the link from `/qa/messages` in a **phone-sized window (390×844)**, tap **Continue**, enter the date of birth.
- Use **made-up** patients only (e.g. "Sam MT44b <time>", born 17/05/1980, an email such as `mt44b.<time>@example.org`).
- **API checks** (T-09, T-11): run `fetch` from the browser's console on a page of the app while signed in (the session cookie goes with it). The episode's API path is the page's address with `/api` in front, e.g. page `http://localhost:3000/o/<trust>/h/<hospital>/episodes/<id>` → `/api/o/<trust>/h/<hospital>/episodes/<id>`.

### The test HQ (built in T-01)

| Question (Select One, Yes / No, required) | Disclosure on **Yes** |
| --- | --- |
| "Do you smoke?" | SNOMED **77176002** (Smoker): fills in **ASA II** by itself; add a note "Smoker" (category Lifestyle) |
| "Do you have heart failure?" | ICD10 **I50.9**: choose **III**; add a note "Heart failure" (category Cardiovascular) |

**How to add a disclosure:** in the editor, click a question's card; in the right-hand panel open **Disclosures**; **Add** next to an option; **+ Add disclosure**; add a code (the picker has a SNOMED / ICD10 switch) and a note with a category; **Save output**, then **Done**.

So **heart failure Yes** = ASA III → the default **High risk** rule fires (task, flag, and the anaesthetist's grade required before Ready for admission). **Smoker only** = ASA II → **Low risk** fires (auto-triaged to Ready for admission; no requirement).

## Test cases

Do them in order. "The ASA section" means the POA Summary's **Pre operative assessment ASA grades** section, on the **Validated summary** tab.

### T-01: build and publish the test HQ

1. As **Alex**, Questionnaires → **New**: name **"MT S44b <time>"**.
2. Question Set **"About you"** (patient): the two questions in the table, each **required**, with their disclosures.
3. Question Set **"Nurse checks"** (clinician): one text question.
4. **Publish** (the default rules come with a new HQ).

**Expected:** it publishes.

### T-02: a high-risk episode, reviewed

1. As **Cara** (Hospital A1), **New episode** on "MT S44b <time>" for a made-up patient with an email (patient **H**).
2. Open the invite from `/qa/messages` in a **390×844** window. Answer **Smoke: No**, **Heart failure: Yes**. Send.
3. As **Cara**, open the episode: **Validate** each Question Set, then **Complete HQ review**.
4. Open the **POA Summary** and look at the ASA section.

**Expected:** the episode is at **Ready for POA**. In the ASA section: the red line "Ready for admission needs the anaesthetist's ASA grade first (the rule “High risk”)."; the **Nurse ASA grade** dropdown with its **Save**; then **"Anaesthetist ASA grade: No grade yet"**, no history list, and an **Add grade** box with only an **Anaesthetist ASA grade** dropdown (no reason box) and **Add grade**. The top details say "Anaesthetist ASA grade: Not recorded". The status dropdown greys out "Ready for admission (add the anaesthetist's ASA grade first)".

### T-03: the first grade (no reason asked)

1. In the **Add grade** box, click **Add grade** without choosing a grade.
2. Choose **ASA III** and click **Add grade**.
3. Reload the page.

**Expected:**
- Step 1: "Choose an ASA grade." Nothing is added.
- Step 2: "Anaesthetist ASA grade: **ASA III**"; one history entry: **ASA III** with a **Counts** badge and "Given by Cara Clinician on <date, time>" (no reason); an **Entered in error** button on it. The red line turns grey: "The anaesthetist's ASA grade is added, as the rule “High risk” asks." The top details say ASA III. General notes: "Anaesthetist ASA grade set to 3" by Cara. The dropdown in the Add box is empty again, and a **Why is the grade changed?** box has appeared.
- Step 3: all the same after the reload.

### T-04: a re-grade needs a reason; both stay, the latest counts

1. Choose **ASA IV**, leave the reason empty, click **Add grade**.
2. Type **"New echo result"** in **Why is the grade changed?** and click **Add grade**.

**Expected:**
- Step 1: "Say why the grade is changed." Nothing is added; ASA IV stays chosen.
- Step 2: "Anaesthetist ASA grade: **ASA IV**". The history lists **ASA IV** first (newest), with **Counts** and "Given by Cara Clinician on …: New echo result", then **ASA III** (no badge, still shown, with its own Entered in error button). The top details say ASA IV. General notes: "Anaesthetist ASA grade changed from 3 to 4: New echo result".

### T-05: mark the latest entered in error: the earlier counts again

1. On the **ASA IV** entry click **Entered in error**. Click **Mark entered in error** with the box empty.
2. Type **"Typed in the wrong box"** and click **Mark entered in error**.
3. Reload.

**Expected:**
- Step 1: "Say why the grade was entered in error." Nothing changes. (**Back** closes the form.)
- Step 2: ASA IV is still listed but **struck through**, badged **Entered in error**, with "Entered in error: marked by Cara Clinician on <date, time>: Typed in the wrong box", and no button any more (it can't be unmarked). **ASA III** has the **Counts** badge again; "Anaesthetist ASA grade: **ASA III**"; the top details say ASA III; the requirement line stays grey. General notes: "Anaesthetist ASA grade 4 marked entered in error: Typed in the wrong box".
- Step 3: the same after the reload.

### T-06: at Ready for admission the only grade can't go; Admit works

1. Go back to the episode and change the status to **Ready for admission**.
2. On the POA Summary, click **Entered in error** on **ASA III** (the only one that counts), type "Trying it", and click **Mark entered in error**.
3. Click **Back** on the form. Back on the episode page, change the status to **Admitted** (yesterday's date).
4. On the POA Summary try step 2 again. Then do the same once the episode is **Discharged** (yesterday's date or today).

**Expected:**
- Step 1: the move works (the grade counts).
- Step 2: refused, in the form: "Add the correct grade first, or move the episode back from Ready for admission." ASA III still counts and isn't struck through.
- Step 3: Admitted works.
- Step 4: refused the same way at Admitted and at Discharged.

### T-07: with an earlier grade to fall back on, the latest can go

1. **New episode** for patient **H2** on the same HQ; on the phone answer heart failure **Yes**; send. As Cara validate every set and **Complete HQ review**.
2. Add **ASA III**, then **ASA IV** with reason "Second opinion".
3. Change the status to **Ready for admission**.
4. Mark **ASA IV** entered in error ("Wrong patient").
5. Try to mark **ASA III** entered in error.

**Expected:** step 3 works; step 4 works and ASA III counts again; step 5 is refused with "Add the correct grade first, or move the episode back from Ready for admission."

### T-08: before Ready for admission the only grade can be marked; then the move is refused

1. **New episode** for patient **H3** as in T-07, reviewed (Ready for POA). Add **ASA II**.
2. Mark it entered in error ("Wrong patient").
3. Look at the status dropdown on the episode page and try **Ready for admission**.
4. On the POA Summary choose **ASA III** and click **Add grade** with the reason box empty; then type "The correct grade" and click **Add grade**.
5. Try **Ready for admission** again.

**Expected:** step 2 works: ASA II struck through, "Anaesthetist ASA grade: No grade yet", the requirement line red again, and **no** "No ASA grades have been added." line above (the history isn't empty). Step 3: "Ready for admission (add the anaesthetist's ASA grade first)" is greyed out. Step 4: the reason box shows (the episode already has a grade, though entered in error); with it empty: "Say why the grade is changed."; with the reason ASA III is added and counts, with "…: The correct grade"; General notes: "Anaesthetist ASA grade set to 3: The correct grade". Step 5: the move works.

### T-09: no edit, no delete (the app and the API)

1. On any graded episode's POA Summary, look for any way to change or remove an anaesthetist's grade.
2. On that episode's page, open the browser console and run (with your episode's API path and current revision, which `GET` returns as `episode.revision`):
   ```js
   const p = location.pathname.replace(/\/poa$/, "");
   const ep = await (await fetch("/api" + p)).json();
   const r = await fetch("/api" + p, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ anaesthetistAsa: 2, revision: ep.episode.revision }) });
   [r.status, await r.text()]
   ```
3. Repeat with `anaesthetistAsa: 0`, then with `anaesthetistAsa: null`.
4. Try `fetch("/api" + p + "/anaesthetist-asa/" + ep.anaesthetistAsaGrades[0].id, { method: "DELETE" })` and the same with `method: "PATCH"`.
5. Reload the POA Summary.

**Expected:** step 1: only **Add grade** and **Entered in error**; no dropdown holds the anaesthetist's saved grade, no Clear or Delete. Steps 2–3: **400** "The anaesthetist's ASA grade is added on its own (POST …/anaesthetist-asa); a grade given is never changed or cleared." Step 4: 404 or 405. Step 5: the grades and the one that counts are unchanged; no new General notes line.

### T-10: two tabs (a grade added meanwhile)

1. Open the same reviewed episode's POA Summary in two tabs.
2. In tab 1 add a grade (with a reason if one counts).
3. In tab 2, without reloading, choose a different grade (and a reason) and click **Add grade**.
4. Click **Reload** in tab 2's message, then **Add grade** again.

**Expected:** step 3: "This episode changed since you opened it." with **Reload**; the grade and reason you typed stay. Step 4: after Reload the history shows tab 1's grade, your typed values are still there, and the add then works (it counts; tab 1's stays below it).

### T-11: other hospitals and trusts

1. Copy patient H's episode POA Summary address (Hospital A1). Sign in as **Val** (Hospital A2 only) and open it.
2. As Val, in the console on any page of Hospital A2, run `fetch("/api/o/<trust A>/h/<hospital A1>/episodes/<H's id>/anaesthetist-asa", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ grade: 2, reason: "x", revision: 0 }) }).then(r => r.status)`.
3. Sign in as **Bea** (Trust B) and open the same address; repeat the fetch.

**Expected:** Val and Bea can't see A1's episode or its grades (the app's not-found message); both fetches give **404**; H's grades are unchanged when Cara looks again.

### T-12: cancelled and archived episodes are read-only

1. Take patient **H2**'s episode (T-07), or any graded episode: **Cancel** it (any reason).
2. Open its POA Summary.
3. As **Sue**, archive another graded episode and open its POA Summary.

**Expected:** the history is shown (struck-through entries too), but there's no **Add grade** box and no **Entered in error** button, on both. The grade that counts is kept.

### T-13: no rule requires it

1. **New episode** for patient **L** on the same HQ; on the phone answer **Smoke: Yes**, **Heart failure: No**; send (Low risk auto-triages it to **Ready for admission**; no requirement line).
2. On its POA Summary add **ASA II**, then mark it entered in error ("Wrong patient").
3. Add **ASA I** with the reason box empty, then with the reason "Re-checked".

**Expected:** both work (no rule requires the grade, so the only grade can be marked even at Ready for admission). "Anaesthetist ASA grade: No grade yet", and no "No ASA grades have been added." line above the struck-through grade. Step 3: refused with "Say why the grade is changed." until the reason is given; then ASA I counts.

### T-14: print and narrow window

1. On a POA Summary with three grades (one struck through), press **Print** and look at the print preview (if your tools can't open it, say so and skip).
2. Narrow the window to about 500 px wide.

**Expected:** print: the grade that counts and the history (with the strike-through) show; the Add box and the buttons don't. Narrow: the Add box wraps (dropdown, reason, button), nothing overlaps or is cut off badly; every **Entered in error** button reads in full, including beside an entry whose "Given by … : reason" line wraps (test report 1, F1).

### T-15: the same grade again is a confirmation

1. On a reviewed episode whose counting grade is ASA III, add **ASA III** again with the reason "Second anaesthetist agrees".

**Expected:** accepted; the new entry counts and the earlier ASA III stays below it. General notes: "Anaesthetist ASA grade **confirmed as** 3: Second anaesthetist agrees" (not "changed from 3 to 3").

## Regression

- **R-01 (S20):** the **Nurse ASA grade** still saves, changes and clears (choose, **Save**; clear with the ×, **Save**), with General notes "Nurse ASA grade set to 2" / "Nurse ASA grade cleared".
- **R-02 (S43):** a high-risk send still adds the task "Book anaesthetic review", the red **Anaesthetic review** flag and the red requirement line on the episode page ("… Add it on the POA Summary."); Clear the flag with a reason still works.
- **R-03 (S42):** a low-risk send still auto-triages to Ready for admission, with the **Auto-triaged** card and **Checked**.
- **R-04 (S39):** the ASA section still shows the **Suggested ASA** with its reasons, on both tabs; the anaesthetist's and nurse's grades are on the validated tab only.
- **R-05 (S20):** an Undo after Admitted (Undo: back to Ready for admission) works on an episode whose grade counts.

## Out of scope (don't report)

- A history for any other clinician entry, the nurse's grade included (only the anaesthetist's is decided).
- The suggested ASA (worked out from answers; unchanged).
- The General notes word grades as numbers ("set to 3") while the page says "ASA III" (older wording, in pending.md).
- Known cosmetic items at narrow widths listed in pending.md (S43's flag button, the greyed Undo label).
