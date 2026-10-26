# S39 test report 3 (retest after verification 1)

**Summary: 14 checks re-run, all passed. 0 failed, 0 blocked. Verification 1's F1 ("No ASA grade" overridden by adding a code) and F2 (Emergency tick lost when a code is removed) are both fixed.**

- **Code:** `c1f6c69`, plus S39's uncommitted changes and the fixes for verification 1, all in the working tree (not committed; 44 changed or new paths).
- **When:** 26 Oct 2026, 19:41–19:52.
- **Where:** http://localhost:3000. The conductor ran `make dev` on freshly migrated and seeded data. I tested in Chrome through Claude in Chrome, at desktop size.
- **Data made for this round:**
  - HQ **MT S39 R3 1945 Retest** v1, published, made as Alex Author. Question Set 1 has one Select Many question:
    - Statement 1 has two disclosures. The first is COPD + Cough + E11.9 with No ASA grade (made in T-03b). The second is Smoker, **ASA IIE (Current smoker)** (made in T-05c).
    - Statement 2 has E11.9, ASA III (Poorly controlled diabetes).
  - Episode **Rory MT S39 R3 2000 Retest**: Cara Clinician, Hospital A1, made-up email `mt-s39-r3-2000@example.test`, made-up date of birth 14/03/1961. The patient answered through the invite link with Statements 1 and 2 and sent. Cara then validated with Statement 2 unticked.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-03b (new) | Pass | COPD fills III. I chose **No ASA grade**, then added Cough: the grade stays empty and No ASA grade stays selected. I added ICD10 E11.9: its three diabetes examples joined the box, nothing was filled in, No ASA grade stayed selected, and the box stayed plain (not the yellow "must choose" box). Save output was accepted, and the disclosure saved with **no** ASA badge. |
| T-05c (new) | Pass | COPD (III), Emergency (E) ticked, then Smoker added: still III with E. Removing COPD gives **II**, with the Emergency tick **still on**, "ASA II: Current smoker (Adults)" selected and the reason line "Current smoker". It saved as **ASA IIE (Current smoker)**, and it was the same after a reload. |
| Verification fix: picking a grade clears "No ASA grade" | Pass | E11.9, No ASA grade chosen, then **IV** picked from the grade list: No ASA grade is no longer selected, no radio is selected, and there's no reason line. |
| T-01 (quick) | Pass | The result row says "· ASA III". Clicking it fills in III, selects "ASA III: COPD (Adults)" and gives the reason line "COPD" (from the ASA guide). |
| T-02 (quick) | Pass | E11.9 alone: the grade is empty, the yellow box lists the three examples, and Save is refused with "Choose the ASA grade from the ASA guide, or No ASA grade.". Choosing No ASA grade removes the red message. |
| T-04 step 5 / T-06 step 3 (quick) | Pass | From IV, picking **III** from the list selects "Poorly controlled diabetes" and gives its reason. It saved as ASA III (Poorly controlled diabetes). |
| T-08 (quick) | Pass | Clinician preview, Statement 1: "ASA IIE / Smoker (77176002) – Current smoker". Statements 1 + 2: "ASA IIIE", with the diabetes reason, then "Smoker (77176002) – Current smoker (emergency)". |
| Episode page before sending | Pass | Before the patient sent, the Health questionnaire card showed no suggested ASA line, and the page had no "ASA" text. |
| T-10 (episode page) | Pass | After sending: "**Suggested ASA IIIE**: Type 2 diabetes mellitus without complications (E11.9) – Poorly controlled diabetes; Smoker (77176002) – Current smoker (emergency) (the anaesthetist confirms)". |
| T-10 (POA Summary) | Pass | Validated tab: "Suggested ASA IIIE", both reasons, "Suggested from the answers…", then the nurse's and anaesthetist's grade boxes. The Patient answers tab shows the same suggestion and reasons, without the boxes. |
| T-11 | Pass | The diabetes reason on the Patient answers tab opens Validate for Question Set 1 at `?q=q_kfwtxq`. The question is highlighted in yellow, and there's a "← POA Summary" link. |
| T-12 | Pass | Validated with Statement 2 unticked. The episode page says "Suggested ASA IIE: Smoker (77176002) – Current smoker". The Validated tab says IIE (Smoker). The Patient answers tab still says IIIE with both reasons. |
| T-16 | Pass | Loading `…/poa?tab=patient` directly opens that tab, with its suggestion. |
| R-05 (part) | Pass | The patient's pages (the set list, the set and the thank-you page) show no ASA, anaesthetist or ASA-reason text. Checked at desktop width only. |

Not re-run: T-03, T-05, T-05b, T-07, T-09, T-13, T-14, T-15, T-17 and R-01 to R-04. They passed in report 1 or 2, and verification 1's fixes don't touch them.

## Failures

None.

## Found while exploring

- **A saved "No ASA grade" isn't shown as chosen when the disclosure is opened again.** I opened the T-03b disclosure (COPD + Cough + E11.9, saved with No ASA grade) with Edit. The guide box lists all four examples, but nothing is selected, not even No ASA grade. Adding Smoker then turns the box yellow and asks for a choice again; nothing is filled in. So the author's earlier answer isn't remembered, and they're asked again. Nothing wrong is saved. I cancelled, and the disclosure is unchanged. This matches the build notes ("opening an older disclosure with no grade shows the box but doesn't require a choice"), so it isn't counted as a failure. The builder may want to decide whether "No ASA grade" should be stored, so it shows on reopening.
- **"(emergency)" marks only a reason below the top grade.** When the emergency disclosure is itself at the top grade, its line has no "(emergency)", so the reader can't tell which disclosure made the grade E:
  - Statement 1 alone gives "ASA IIE / Smoker (77176002) – Current smoker".
  - The same is true of the episode page's "Suggested ASA IIE" after validation.
  - With diabetes at III, the smoker line does say "(emergency)".

  `asa.ts` documents this ("one below the top grade is listed with '(emergency)'"), so it's by design. Mentioned only for the builder to confirm.

## Console and network errors

- No console errors or warnings on the episode page or the POA Summary after a reload, which is when tracking started.
- Every `/api/` call returned 200. Two `GET /api/me` calls show as "pending" because I navigated away straight after the reload.

## GIFs

- `~/Downloads/S39-R3-T03b-T05c.gif`: T-03b (No ASA grade stands while Cough and E11.9 are added) and T-05c (removing COPD keeps E, giving II with Current smoker, saved as ASA IIE).
- `~/Downloads/S39-R3-summary.gif`: the episode page and the POA Summary's two tabs after validation (IIE validated, IIIE patient).

## Hand checks for Rahul

- R-05 in a real 390×844 window. As before, this round was at desktop width only.
