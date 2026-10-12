# S08 test report 4 (retest 3)

**13 passed, 0 failed, 0 blocked.**

This round re-ran:
- report 3's failure, T-25 (step 4 now on an "E2E …" draft);
- the new cases T-28 and T-29;
- T-27 step 1, again during R-02;
- the spot checks T-01, T-02 and T-15 (API);
- all six regression cases, R-01 to R-06.

The root fix for report 3's F-1 works everywhere the plan names. On the "E2E …" fixtures (outputs with no codes and no category), these all work with no console errors:
- Save answers and Run all;
- the Disclosures dialog;
- the Publish check;
- the POA Summary.

The test case is stored as `"Note: Smoker"`, with no stray "undefined". Two small cosmetic things turned up while exploring (below). Neither is a failure of an S08 case.

**Environment**
- Commit `603d3d5`. The working tree is not clean: it holds the uncommitted S08 build and fixes plus planning docs, and this report tests that working tree.
- Run on 12–13 Oct 2026, 23:58 to about 00:15 BST.
- App at http://localhost:3000 (API :8080). The conductor's `make dev` was running, and `/api/health` returned ok.
- Browser: Chrome through Claude in Chrome, in a 1280×773 tab.
  - The phone case ran in a 390×844 iframe.
  - As in reports 2 and 3, the first click after a page load or a dialog closing was sometimes ignored until a screenshot was taken. This is a harness quirk, not an app bug.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-25 Preview read-only | Pass | **Step 4 (report 3's failure):** as Alex on "E2E trust-wide 1210 235623 eaec" → About you → Preview, I chose Yes. Clinical outputs showed Notes "Smoker". I named the test case "MT S08 0001 smoker" and clicked **Save answers**: it was listed. **Run all** gave "All 1 passed". The API showed the chapter at revision 2, with `expected: ["Note: Smoker"]`. On my published v1 ("MT S08 0008 r4"), Alex sees "Read only: test cases can be changed only in a draft version." **Steps 1–3:** Sue Super and Cara Clinician on the same draft see "Read only: only super clinicians and admins of the whole trust can change test cases.". They get no name box, no Save answers and no delete icon, and the header button reads **Open in editor**. For Sue, Run all gives "All 1 passed", and Load fills in Yes and shows "Smoker". |
| T-28 Outputs without codes or a category | Pass | **1:** "Smoker", no crash. **2:** saved, "All 1 passed" (same run as T-25 step 4). **3:** in the editor, the Disclosures of "Yes" list "Smoker" with Edit and Remove, and no error. Edit opens with "No codes." and an empty Category. **4:** Publish opens normally ("Publish version 1? Patients can then be given this version…"), with no problems listed. I cancelled, so the fixture stays a draft. **5:** as Cara, I opened "Sam J5 1210 235620 c5be" → POA Summary. Patient answers → About you shows "Smoker". Validated summary shows "Non-smoker" with "Patient answered 'Yes', now 'No'" (the journey's clinician change). No console errors. |
| T-29 "Delete this?" keeps its text | Pass | A MutationObserver on the dialog recorded only the full text: "Delete this? 'MT r4 delete me' will be deleted, with its options and disclosures. Cancel Delete". It recorded this through the whole fade-out, for both **Cancel** and **Delete**. It never recorded the bare "Cancel Delete". Delete removed the question, and it stayed gone after a reload. |
| T-27 step 1 (re-check) | Pass | During R-02, Cancel on the blocked Publish dialog kept "2 problems to fix first …" while fading. The observer never saw "Patients can then be given this version…". |
| T-01 Cara straight in | Pass | Lands on `/o/…/h/…0a/episodes`. The header shows Trust A, Hospital A1 and a CLINICIAN badge. |
| T-02 Alex's picker | Pass | Trust A, "Super clinician, whole trust", with Hospital A1 and A2. |
| T-15 Leak (API spot check) | Pass | As Cara, `GET /api/o/<Trust A>/h/<A2>/episodes` gives 404 `{"error":"Hospital not found."}`. Her `/api/me` lists only Hospital A1. |
| R-01 J1 Author | Pass | I created "MT S08 0008 r4" with Question Set 1 and Page 1. It has a Yes/No question "MT r4 do you have asthma?", with a Yes disclosure "MT r4 asthma" (Allergies). Everything was kept after a reload, and the Preview shows it (Yes → Notes, ALLERGIES, "MT r4 asthma"). I saved the test case "MT r4 asthma yes". |
| R-02 J2 Publish | Pass | I added "MT r4 asthma details", shown when asthma is Yes, then deleted the asthma question. The header showed "1 logic problem". Publish listed two problems: the missing q_rifnmn, and the test case "MT r4 asthma yes" fails. Undo restored the question. Publish then worked: V1 · PUBLISHED, READ ONLY. **Create new version** gave V2 · DRAFT. |
| R-03 J3 Patient at phone size | Pass | Cara created an episode for Jo Bloggs ("MT r4 hip", HQ "MT S08 0008 r4 v1") and clicked **Copy**. The link opened in a 390×844 frame: "Hello Jo", one Question Set. Yes revealed the conditional question. After Finish this section, **Send my answers** (in-app modal) gave "Thank you, Jo". scrollWidth 390 = clientWidth 390. Cara's view after a reload: READY FOR REVIEW. |
| R-04 J4 Validate | Pass | Changing the answer to No showed "Patient answered: Yes". After Validate and **Complete HQ review**: READY FOR POA. |
| R-05 J5 POA Summary | Pass | Validated summary: "Changed by the clinician … Patient answered 'Yes', now 'No'". Patient answers: "MT r4 asthma". Setting the status to POA complete is logged in General notes. The comment "MT S08 retest4 comment" appears there. |
| R-06 Dates | Pass | "11 December 1974", "17 May 1980", "13/10/2026, 00:05". |

## Failures

None.

## Found while exploring

1. **Cosmetic, probably older than S08: the editor's Disclosures dialog switches to "No disclosures yet." as it fades out.**
   - Steps: as Alex, in a draft's editor, click an option's disclosure badge (e.g. the "1" next to Yes on "Do you smoke?"), then click **Done**.
   - A MutationObserver recorded the dialog going from "Disclosures · Do you smoke? → Yes · Smoker · Edit · Remove …" to "Disclosures · No disclosures yet. · + Add disclosure · Done" during the fade-out. A screenshot caught the faded dialog reading "No disclosures yet.".
   - This is the same family as the T-24, T-27 and T-29 fixes, in a dialog none of them covered.
2. **Cosmetic, probably older than S08: a long test-case name squeezes the PASS/FAIL badge.**
   - I named a test case "MT r4 🚭 non-smoker with a rather long name to see how the list copes with wrapping text".
   - After Run all, its result badge shows only "P.." and its PATIENT badge "P…", while the name itself is truncated with an ellipsis. The badges should keep their width and the name should take the truncation.
3. **Keyboard, minor:** pressing Enter in the Preview's test-case name box doesn't save. You have to click **Save answers**.
4. **Checked and fine:**
   - Double-clicking **Save answers** saves the test case once, with no duplicate.
   - Leading spaces in a test case name are trimmed. Emoji are kept.
   - A failing test case whose expected outputs have no category runs cleanly:
     - I changed the "E2E test case …" draft's No disclosure to "Non-smoker MT changed". The editor requires a category for that save, which is correct, so I chose Allergies.
     - Run all then shows "1 of 2 failed".
     - Publish lists "Test case '…' fails" with no crash and no console error.

## Console and network errors

- None. No console errors or warnings were seen on any page tested (Preview, editor, POA Summary, patient page).
- The only 4xx was the deliberate 404 probe as Cara in T-15.

## Hand checks for Rahul

None. Every confirmation used (Publish, delete question, Send my answers) is an in-app modal.

## GIFs

- `~/Downloads/S08-T28.gif` (happy path): Alex on the "E2E trust-wide …" draft Preview chooses Yes, "Smoker" appears, and he names and saves the test case. Run all then gives "All 1 passed".

## Test data created or changed

- **Questionnaire** "MT S08 0008 r4" (Alex):
  - v1 published, v2 draft;
  - Question Set 1 has the asthma Yes/No question (Yes disclosure "MT r4 asthma", Allergies) and the conditional "MT r4 asthma details";
  - test case "MT r4 asthma yes".
- **"E2E trust-wide 1210 235623 eaec"** (draft): I added the test case "MT S08 0001 smoker".
- **"E2E test case 1210 235630 78cf"** (draft):
  - I added the test case "MT r4 🚭 non-smoker with a rather long name …";
  - I changed the No disclosure to "Non-smoker MT changed" (Allergies). Its two test cases now give "1 of 2 failed".
- **Episode:** Jo Bloggs at Hospital A1, "MT r4 hip", HQ "MT S08 0008 r4 v1".
  - The patient answered Yes ("MT mild, inhaler").
  - Cara validated it, changing the answer to No, and set it to POA complete.
  - Comment: "MT S08 retest4 comment".
