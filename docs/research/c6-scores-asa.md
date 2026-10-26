# C6 research: scores, ASA, auto-triage and flags

*26 Oct 2026. Desk research for plan C6 ([c6-scores-asa-triage.md](../plans/c6-scores-asa-triage.md)), to answer the "Ask Rahul first" questions in [S38](../sessions/S38-scores-across-sets/brief.md) (Q1), [S39](../sessions/S39-suggested-asa/brief.md) (Q1), [S42](../sessions/S42-auto-triage/brief.md) (Q1–Q2) and [S43](../sessions/S43-rule-actions/brief.md) (Q1). Docs only. Sources are listed at the end; the numbers in brackets point to them.*

**Rahul's direction (26 Oct):** scores must be configurable, automatic and not rigid. They may be formula-based, but clinicians must never write formulas, and building them must be easy. Published scores follow the book (e.g. STOP-Bang as published). The result must be exactly right and as good as the mature products, or better. Auto-triage by default means straight to Ready, and the patient's messages go out, configurable per HQ.

The recommendations are at the end ([§5](#5-recommendations)), in plain words.

---

## 1. What we found, in short

1. **Mature UK pre-op products sell scores as built-in, vendor-maintained algorithms, not as things trusts build.** One major UK product claims "over 250 clinically validated algorithms" producing ASA, STOP-BANG, P-POSSUM, SORT, frailty, cardiac and lung risk; its scores "can be turned on or off" per organisation and "additional scores can also be added" (by the vendor's specialists) [1]. Another UK product says ASA grade, risk scores and suggested tasks are "triggered by the patient responses", and that trusts can customise questions and triggers, but it "recommended that minimal customisation is requested" [2]. Both are registered medical devices with the MHRA [1][2].
2. **General form builders all use the same no-code pattern: points on each answer, plus "add N points if …" rules.** Jotform puts a "calculation value" on each option and totals them in a widget [9]. Typeform's calculator adds points "If" an answer is given [10]. REDCap and SurveyJS need a typed formula (`sum()`, `if()`, `iif()`, `age()`) for anything beyond option points [11][12]. FHIR Questionnaire puts a weight on each answer (`itemWeight`, formerly `ordinalValue`) and a FHIRPath `calculatedExpression` for the total; even HL7's own PHQ-9 example needed fixing [13].
3. **"By the book" is about fixed definitions, versions and licences, not only arithmetic.**
   - STOP-Bang has **two published versions**: the original used neck circumference > 40 cm [14]; the updated one uses 43 cm for men and 41 cm for women (shirt collar 17 or 16 inches) [15]. Its high-risk band isn't only "total 5–8": a STOP score of 2 or more **plus** male sex, BMI > 35 or a large neck is high risk too [15][16]. A plain "total ≤ max" band can't express that.
   - Some published scores aren't sums. P-POSSUM and SORT are logistic equations that need lab values, an ECG and operative severity [17][27]; the Clinical Frailty Scale is a clinician's judgement on a 1–9 picture scale [18]. They need built-in calculators, not points lines.
   - **Licences:** STOP-Bang is owned by Dr Frances Chung and University Health Network (UHN); use beyond education needs a licence (stopbang.ca) [19]. The Clinical Frailty Scale is © Rockwood/Dalhousie; commercial use is reviewed for a licence [18].
   - Curated libraries lock the content: REDCap's Shared Library is reviewed for accuracy and copyright, and an imported instrument can't be changed [20]. MDCalc has each calculator written by a named contributor and checked by a peer reviewer [21].
   - **Missing answers matter:** REDCap warns that `sum()` silently skips blanks, while `+` gives no result [11]. A score that counts a missing answer as 0 under-states risk.
4. **ASA is a clinical judgement, and "suggest, a clinician confirms" is the norm.**
   - The ASA statement (last amended 13 Dec 2020; republished in *Anesthesiology Open*, Jan 2026 [3]) says its examples are guides "including but not limited to" [4]. It says the system alone doesn't predict risk [3][5], and the **final grade is assigned by the anaesthesiologist on the day of anaesthesia care** [6][7].
   - Agreement between raters is only moderate [8]. **Adding institution-specific examples to the ASA ones raised clinic-to-day-of-surgery agreement from 74% to 91% in adults** [7].
   - **No published code-to-ASA mapping exists.** ASA is graded by *severity and control*, not by diagnosis: well-controlled diabetes is II and poorly controlled is III; a heart attack more than 3 months ago is III and a recent one is IV [4]. So one ICD-10 or SNOMED code can't fix the grade.
   - Machine-learning models predict "ASA 1–2 vs 3+" for triage, not the exact grade. The best UK one (GP Connect data, Aire Logic's Smart PreOp) has 0.95 precision for finding low-risk patients [22].
   - GIRFT calls ASA grades "subjective" and recommends a validated tool to support triage [23].
5. **Fully automatic "straight to Ready, tell the patient" is the stated aim, but most UK services still keep a short human step.**
   - NHS England's guide says triage "will be supervised by registered healthcare professionals" and "no clinical decision-making will be made by non-clinical staff alone" [24].
   - GIRFT's low-risk pathway ends in "Suitable for surgery? Yes → inform waiting list suitable for surgery", after an electronic pre-op assessment (ePOA) [23].
   - In the UK examples, the POA team decides who is fit to proceed (Chelsea and Westminster, about 180 patients a month skip POA) [25], or low-risk patients get a short telephone check (East Kent, "low touch", with "no touch" as the aim) [26]. The two UK products above sort patients into folders, tags and task lists for a clinician [1][2].
   - CPOC's guidance has most patients seen by a nurse, with telephone or video for low-risk patients, by criteria the local POA leads set [28].
6. **Flags are prospective warnings that stay until someone clears them.**
   - In Epic, FYI flags are Active or Inactive and are "marked inactive when no longer pertinent" [29][30].
   - FHIR's `Flag` has the same status values and an active period, and should be kept "concise and limited to high-priority items" [31].
   - One UK pre-op product uses tags such as "refer to the anaesthetist" or "pharmacy review", and each tag filters a worklist for that team [2].

---

## 2. Scores built from patient data

### 2.1 How the mature products do it

| Product / tool | Who builds the score | How inputs like age, sex and BMI get in | Notes |
|---|---|---|---|
| UK pre-op product A (VitalHub's Synopsis) [1] | The vendor ("250+ algorithms"); trusts switch scores on or off; the vendor adds new ones | From the questionnaire and "lab data from other systems" | ASA, STOP-BANG, P-POSSUM, POSPOM, SORT, frailty, cardiac, lung, mortality. MHRA-registered, DCB0129, CE |
| UK pre-op product B (Ultramed's Ultraprep / MyPreOp) [2] | The vendor, "developed collaboratively with Trusts" | From the branching questionnaire | ASA, Rockwood frailty and suggested tasks are "triggered by the patient responses"; "customisation of triggers"; MHRA-registered |
| Aire Logic's Smart PreOp [22] | A trained ML model | GP record via GP Connect, procedure, demographics | Sorts patients into ASA 1–2 or 3+, plus 30-day mortality |
| Jotform [9] | The author | Option "calculation values"; a calculation widget | No patient profile |
| Typeform [10] | The author | "If <answer> then add N to score" | One score per form |
| REDCap [11] | The author types `sum()`, `if()`, `datediff()` | `datediff([dob],"today","y")`; "today" moves on every save | Shared Library instruments are locked [20] |
| SurveyJS [12] | The author types expressions (`iif`, `age`), or the developer adds functions | Survey variables; `calculatedValues` | What we use today |
| FHIR SDC [13] | The author: `itemWeight` per answer plus a FHIRPath `calculatedExpression` | `launchContext` (patient) | Spec still settling (FHIR-37508) |
| MDCalc [21] | Editorial: a named contributor writes it, a peer reviewer checks it | The clinician types the inputs | Evidence rating since Dec 2025 |

**What the UI patterns have in common:** points on each answer, plus "add N points if <fact> <compare> <value>" lines. No mainstream tool asks a clinician to write `iif(...)`. Where formulas exist (REDCap, SurveyJS, FHIR), the people who write them are research coordinators or developers.

### 2.2 What "by the book" requires

- **The exact published items, cut-offs and bands, with the version named.** For STOP-Bang that means the 2008 original [14] or the updated 43/41 cm neck version [15], and its extra high-risk rule [16]. An author-built copy can drift (40 or 43 cm, "> 50" or "≥ 50", the high-risk rule left out).
- **The published worked examples as test cases**, so a change is caught before publishing (we already have test cases, PRV-05, and ASA-04).
- **A source and licence note on each score** (SCR-04 already says this). STOP-Bang needs a UHN licence for commercial clinical use [19].
- **A clear rule for missing inputs:** a score with a missing input is *incomplete*, and must never be read as low risk.
- **Age on a fixed date:** the age on the day the patient answered (stored with the answers), not "today" at view time, so a score never changes later [11].
- **Equation and judgement scores** (P-POSSUM, SORT, DASI's VO2 estimate, the CFS) need named calculators in `packages/clinical`, checked against the paper. Points lines can't express them [17][18][27].

### 2.3 What exists in our code today

- An option carries a `score`. A Calculation of kind `score` totals the options' scores (`calcFormOf` in `fields.ts`).
- Bands have only `max` (`bandsOf`/`bandFor`).
- The patient's age and sex are survey variables (`patientAge`, `patientSex`, `respondent.ts`); the age is computed from today's date.
- A custom formula is possible, but it means typing.

So the missing pieces for STOP-Bang are: points from age, sex and BMI; a band condition beyond `max`; an "incomplete" state; and the age fixed at the answer date.

---

## 3. ASA physical status

### 3.1 The official statement

The ASA Physical Status Classification was approved on 15 Oct 2014, last amended on 13 Dec 2020 (which added the paediatric and obstetric examples), and republished in *Anesthesiology Open* in January 2026 [3][4][5]. Its examples are "including but not limited to". The text below is condensed from the copy the Canadian Anesthesiologists' Society reproduced with permission in 2025 [4].

| Class | Definition | Adult examples | Children | Pregnancy |
|---|---|---|---|---|
| I | Normal healthy patient | Healthy, non-smoking, no or minimal alcohol | Healthy, normal BMI percentile for age | — |
| II | Mild systemic disease | No substantive functional limits: current smoker, social drinker, pregnancy, obesity (30 < BMI < 40), well-controlled DM/HTN, mild lung disease | Asymptomatic congenital heart disease, well-controlled dysrhythmias, asthma without exacerbation, well-controlled epilepsy, non-insulin DM, abnormal BMI percentile, mild/moderate OSA, cancer in remission, autism with mild limits | Normal pregnancy; well-controlled gestational HTN; pre-eclampsia without severe features; diet-controlled gestational DM |
| III | Severe systemic disease | Substantive functional limits; one or more moderate to severe diseases: poorly controlled DM or HTN, COPD, BMI ≥ 40, active hepatitis, alcohol dependence, pacemaker, moderately reduced EF, ESRD on regular dialysis, MI/CVA/TIA/CAD-stents more than 3 months ago | Uncorrected stable congenital heart abnormality, asthma with exacerbation, poorly controlled epilepsy, insulin-dependent DM, morbid obesity, malnutrition, severe OSA, cancer, renal failure, muscular dystrophy, CF, organ transplant, symptomatic hydrocephalus, premature infant PCA < 60 weeks, difficult airway, long-term TPN, full-term infant < 6 weeks | Pre-eclampsia with severe features; gestational DM with complications or high insulin needs; thrombophilia needing anticoagulation |
| IV | Severe systemic disease that is a constant threat to life | MI/CVA/TIA/CAD-stents within 3 months, ongoing cardiac ischaemia or severe valve dysfunction, severely reduced EF, shock, sepsis, DIC, ARD or ESRD not on regular dialysis | Symptomatic congenital heart abnormality, heart failure, active sequelae of prematurity, shock, sepsis, AICD, ventilator dependence, severe trauma, advanced cancer | Pre-eclampsia with HELLP or other adverse event; peripartum cardiomyopathy with EF < 40%; uncorrected or decompensated heart disease |
| V | Moribund, not expected to survive without the operation | Ruptured aneurysm, massive trauma, intracranial bleed with mass effect, ischaemic bowel with significant cardiac pathology or multi-organ dysfunction | ECMO, respiratory failure or arrest, malignant hypertension, decompensated heart failure, hepatic encephalopathy | Uterine rupture |
| VI | Brain-dead organ donor | — | — | — |

**E** means emergency: delay would significantly increase the threat to life or a body part [4].

**Who assigns it:** "the final assignment of Physical Status classification is made on the day of anesthesia care by the anesthesiologist after evaluating the patient", although it "may initially be determined at various times during the preoperative assessment" [6][7]. Departments may write institution-specific examples to supplement the ASA-approved ones; doing so raised agreement from 74% to 91% (weighted κ 0.62 → 0.86) [7].

**In the UK:** GIRFT says ASA grades "can be subjective", that ASA 3 covers a broad range of patients, and recommends a validated tool such as SORT alongside it. Its tools are "only as an example" and are to be adapted locally [23].

### 3.2 Can ASA be derived from codes?

- **Codes for the grade itself exist** (SNOMED CT 413495001–413499007 for ASA 1–5; LOINC 97816-3) [32]. They're useful for exporting the grade, not for working it out.
- **No standard maps diagnosis codes to ASA.** The grade depends on control, severity, timing and function (see the table above), which a diagnosis code doesn't carry.
- **Research models** predict the low/high split, not the exact grade:
  - Vanderbilt, from ICD-9 codes and other data: AUC 0.884 [33];
  - a multicentre random forest: anaesthetists still beat it in patients with several conditions [34];
  - the UK GP Connect model (2025): 0.95 precision for ASA 1–2 [22].
- **The products "categorise" against ASA, and humans decide the pathway.** Product A's triage tool "will automatically calculate the patient's risk and categorise them against a particular ASA score, this with numerous other scores is used by the triage team to determine the pathway" [1].

**Conclusion:** suggest the ASA with reasons, and let a clinician confirm it. Our model already supports this: the suggested ASA is separate from the nurse's and the anaesthetist's grades. A code alone can give only a *starting* grade, and for many conditions it needs the severity chosen.

### 3.3 Inter-rater reliability

- A systematic review found moderate agreement overall [8].
- Pre-hospital and in-hospital physicians agreed moderately to substantially (κw 0.47–0.89); with the same information it rose to 0.65–0.93 [8].
- In a paediatric cancer setting, agreement between raters was near zero [8].

A consistent, explained suggestion helps make grading more consistent.

---

## 4. Auto-triage and flags in mature products

### 4.1 Auto-triage

| Source | What happens to a low-risk patient | Human step? |
|---|---|---|
| NHS England guide (2023, updated May 2025) [24] | Low-risk patients are "fit to proceed (the waiting well)" and are "contacted to hear the outcome of their triage" | Yes: "supervised by registered healthcare professionals" |
| GIRFT POA guidance (2023) [23] | Low risk → electronic POA → "Suitable for surgery? Yes → inform waiting list" | The ePOA includes "formal electronic pre-operative assessment", consent and minimum nursing documentation |
| CPOC (2021) [28] | Telephone or video POA for low-risk patients | Yes: a nurse-led consultation; local leads set the criteria |
| Chelsea and Westminster (Palantir, Isla) [25] | About 180 patients a month no longer need POA | The POA team decides from the answers |
| East Kent (RAG tool) [26] | "Low touch": a short nurse telephone check; "no touch" is the aim | Yes, for now |
| RCoA leads' day (2020) [35] | Green → "proceed to surgery"; amber → nurse-led; red → notes review ± anaesthetic clinic | Varies |
| UK product A [1] | Scores categorise; "used by the triage team to determine the pathway" | Yes |
| UK product B [2] | Folders ("needs attention" and others), tags and suggested tasks for the reviewing clinician | Yes |

**Configurable?** Yes: both UK products let trusts customise triggers and pathways [1][2], and GIRFT says its tools should be "adapted to reflect localised policies" [23].

**Regulation:** software that triages or scores risk is likely a medical device (MHRA; EU MDR Rule 11 class IIa) [36]. Both UK products are registered [1][2].

### 4.2 Flags

- **One UK pre-op product:** tags such as "refer to the anaesthetist" and "pharmacy review" mark a patient for a team, and "filters can be used to identify all patients with a particular tag", giving each team its own worklist. Each patient's timeline records every tag change [2].
- **Epic:** FYI flags are Active or Inactive; a user deactivates one "when no longer pertinent". They don't expire by themselves [29][30].
- **FHIR `Flag`:** status active, inactive or entered-in-error, with a period. A flag is a warning, not a clinical record; keep flags few and high-priority [31].
- **Queensland's pre-anaesthetic triage:** an amber patient must not stay amber indefinitely [37]. Open flags need someone to own them.

---

## 5. Recommendations

Plain words first. The recommended option comes first each time.

### 5.1 S38 Q1: points for age, sex, BMI and earlier answers

**A. "Add points when…" lines, plus STOP-Bang ready-made and locked (recommended).**

- The score form gets a list of lines. The author picks from menus and never types a formula.
- The things to choose from: **Patient** (age on the day they answered, sex), **Measurements** (BMI, height, weight), **Answers** (questions in this set and in earlier sets), and **Other scores** (another score's band).
- The editor writes the formula behind the scenes. Options keep their points.
- A band can say "also High when…", which STOP-Bang's high-risk rule needs.
- If a score's input is missing, the score shows **"Incomplete"**, and rules never treat it as low risk.
- **STOP-Bang (updated version) is the first ready-made score.** It is built from the same lines but locked, with its source, version, licence note and the published examples as test cases.
- The rest of the library (Apfel, RCRI, Caprini, and so on) and the equation or judgement scores (P-POSSUM, SORT, DASI, CFS, as named calculators) follow in C7.

**B. The lines only; every ready-made score waits for C7.** This is what the brief has now. It's simpler, but an author can build STOP-Bang slightly wrong (40 or 43 cm, the missing high-risk rule).

**C. The author types a formula.** This breaks Rahul's rule. It stays only as the hidden "custom" kind for us.

*What the author sees (A):*

> **Score: STOP-Bang** · *Ready-made, locked. Source: Chung 2016 (UHN). Licence: UHN.*
> Snoring loudly? Yes **1 pt** · Tired in the day? Yes **1 pt** · … (from the options)
> ➕ Add **1** point when **Patient › Age** is **over** **50**
> ➕ Add **1** point when **Patient › Sex** is **Male**
> ➕ Add **1** point when **Measurements › BMI** is **over** **35**
> ➕ Add **1** point when **Set 2 › Neck size** is **Large**
> Bands: Low 0–2 · Intermediate 3–4 · High 5–8, **also High when** STOP part ≥ 2 **and** (Male **or** BMI > 35 **or** neck Large)
> *Try it: age 51, male, BMI 30, snores → 3, Intermediate ✓*

*What it means for S38:*
- Q1 = A.
- Q2 changes from (a) to "STOP-Bang now, the rest in C7".
- Add to the build: lines; the "also when" band conditions; the "Incomplete" state; age fixed at the answer date; the locked library entry with its test cases.
- **Before S38 starts:** check the STOP-Bang licence with UHN [19]. Until it's agreed, the ready-made STOP-Bang stays in the platform library on test data only.

### 5.2 S39 Q1: where a code's usual ASA grade comes from

**A. An "ASA guide" file built from the ASA's own examples (recommended).**

- Each ASA example (adults, children, pregnancy, 2020 statement) is listed with its grade and the codes it covers.
- Where the grade depends on severity, the file holds both choices, e.g. diabetes: "well controlled → II", "poorly controlled → III".
- When an author adds a code to a disclosure:
  - **one match** fills in the grade and the example's words as the reason;
  - **several matches** ask the author to pick;
  - **no match** leaves the grade empty.
- The grade and the reason are copied into the disclosure, so a published HQ never changes.
- A clinician checks the file before go-live. A trust's own examples (a screen) come later; they're what raised agreement to 91% [7].
- ASA from facts other than codes uses the bands we already have, e.g. the library's BMI Calculation ships with "30 to under 40 → ASA II" and "40 or more → ASA III".
- Everywhere, the screen says **"Suggested ASA"** with its reasons; the nurse and the anaesthetist set the real grade.

**B. A file with one grade per code** (the brief's (a)). Simplest, but it gives a silent wrong grade for diabetes, high blood pressure, heart attacks (more or less than 3 months ago) and other severity-dependent codes.

**C. A superadmin screen to set a grade per code** (the brief's (b)). The same problem as B, plus a screen.

*What the author sees (A), after adding "Type 2 diabetes (E11)" to the answer "Yes, I take insulin":*

> **ASA from the ASA guide:** ◯ **II**, "well-controlled diabetes" ◉ **III**, "poorly controlled diabetes" *(choose one; you can change it)*

*What the clinician sees on the POA Summary:*

> **Suggested ASA III**: Type 2 diabetes, poorly controlled (E11); BMI 41 (BMI ≥ 40). *Suggested from the answers; the anaesthetist confirms.*

*What it means for S39:*
- The seed file holds examples with grade choices, not one grade per code. The editor shows the choice when a code is ambiguous.
- `codes.asa` becomes optional; the file is the source.
- Add the BMI bands with ASA to the starter library.

### 5.3 Auto-triage (S42 Q1–Q2)

**A. A setting on each HQ, "When the rules say low risk:", defaulting to "Ready for admission and tell the patient now" (recommended; Rahul's view).**

The setting has three choices:
1. **Ready for admission now, and send the patient's message now** (the default);
2. Ready for admission now, and send the message after a clinician clicks Checked;
3. Suggest only: the episode stays in Ready for review with "Rules suggest: Ready for admission", accepted with one click.

The default comes with guards that can't be switched off:
- the server's check ran (S41);
- no score is Incomplete;
- there are no amber or red review flags;
- the HQ's rules passed their test cases at publish.

After the move:
- the episode still goes on the **Auto-triaged** list for a look afterwards, and Checked removes it (S42 Q1 (a) stays);
- the message says what happens next, not "fit for surgery". For example: "Thank you. Your answers mean you don't need a pre-op appointment. We'll contact you with your date."

Before a trust turns the default on, its clinical safety officer signs it off (DCB0160), and the hazard log names it.

**B. Ready now, message after Checked** (S42's current recommendation). Safer, and closer to how most UK services work today [24][25][26]. It stays available as choice 2.

**C. Suggest only.** The most cautious option, and closest to the two big UK products [1][2]. It stays available as choice 3.

*Why A is still recommended:* the clinicians decide in advance by writing, testing and signing off the rules, and a trust that wants a human step picks 2 or 3 per HQ. Our default is ahead of most of the UK market [24][25][26], so the guards and the sign-off are not optional.

*What it means for S42:* add the per-HQ setting (stored with the rules, S40) with its three choices. Q2 becomes "per HQ, default send now". Q1 stays (a).

### 5.4 Flags (S43 Q1)

**A. A flag stays until a clinician clears it with one click and a short reason (recommended).**

- Who, when and why are recorded, e.g. "Anaesthetic review booked 3 Nov".
- Each flag name filters the episode list (as tags do in [2]).
- If the patient sends again and the rule fires again, a cleared flag comes back.

**B. The flag clears itself when the rule's task is done.** This could be a per-rule option later. A task and a flag can mean different things.

**C. Flags expire after some days.** Not recommended: a warning would disappear with nobody deciding [29][31][37].

*What the clinician sees (A):*

> 🔴 **Anaesthetic review** (rule: High risk) · **Clear…** → *Why?* "Seen by Dr Patel, 3 Nov" → *Cleared by Cara Clinician, 4 Nov 10:12*

*What it means for S43:* Q1 = A, plus a reason box, a list filter per flag name, and re-raising a cleared flag on a re-send.

---

## 6. Sources

1. VitalHub (Synopsis), G-Cloud 15 service definition, Jan 2026. https://assets.applytosupply.digitalmarketplace.service.gov.uk/g-cloud-15/documents/721733/214669371000433-service-definition-document-2026-01-29-1636.pdf
2. Ultramed, Ultraprep product specification (G-Cloud 15), Sep 2025. https://assets.applytosupply.digitalmarketplace.service.gov.uk/g-cloud-15/documents/702415/799143724896306-service-definition-document-2025-12-18-1101.pdf
3. ASA Committee of Oversight: Economics, "American Society of Anesthesiologists Statement on ASA Physical Status Classification System", *Anesthesiology Open*, 1 Jan 2026, doi:10.1097/ao9.0000000000000002 (listing). https://resources.wfsahq.org/standards-and-guidelines/american-society-of-anesthesiologists-statement-on-asa-physical-status-classification-system/
4. Canadian Anesthesiologists' Society, Appendix 2 (2025): ASA PS definitions and ASA-approved examples, reproduced with permission. https://www.cas.ca/CASAssets/Documents/Practice-Resources/Guidelines/Appendix-2_2025.pdf
5. OpenAnesthesia, "ASA physical status classification" (updated Jan 2024). https://www.openanesthesia.org/keywords/asa-physical-status-classification/
6. ASA statement text as reproduced by the Kentucky Board of Dentistry. https://dentistry.ky.gov/Documents/ASA_Classification_System.pdf
7. Abouleish AE et al., "Improving agreement of ASA physical status class between pre-anesthesia screening and day of surgery by adding institutional-specific and ASA-approved examples", *Perioperative Medicine* 2020;9:34. https://pmc.ncbi.nlm.nih.gov/articles/PMC7677831/
8. Inter-rater reliability: systematic review https://pmc.ncbi.nlm.nih.gov/articles/PMC4800940 ; Tønsager et al., BMC Anesthesiology 2020 https://uis.brage.unit.no/uis-xmlui/handle/11250/2672656?show=full ; Tollinche et al. 2020 (paediatric cancer) https://synapse.mskcc.org/synapse/works/123099
9. Jotform, "How to assign calculation values". https://www.jotform.com/help/301-how-to-assign-calculation-value
10. Typeform Help Center, "Calculator". https://www.typeform.com/help/calculator/
11. REDCap calculated fields (UConn) https://health.uconn.edu/clinical-research-center/?p=4794 ; "syntax limits and the recalculation trap" https://www.casrai.org/guides/redcap-calculated-fields-syntax-limits-and-the-recalculation-trap
12. SurveyJS, "Conditional logic and dynamic texts" https://surveyjs.io/form-library/documentation/design-survey/conditional-logic ; "Custom functions in expressions" https://surveyjs.io/form-library/examples/use-custom-functions-in-expressions/documentation
13. HL7 FHIR SDC, ordinal-value example https://hl7.org/fhir/uv/sdc/Questionnaire-questionnaire-sdc-profile-example-ordinal-value.html ; FHIR-37508 https://jira.hl7.org/browse/FHIR-37508 ; itemWeight https://packages2.fhir.org/xig/resource/hl7.fhir.uv.extensions.r5%7C5.3.0/StructureDefinition/StructureDefinition-itemWeight.json
14. Chung F et al., "STOP questionnaire: a tool to screen patients for obstructive sleep apnea", *Anesthesiology* 2008;108:812–21 (original STOP-Bang; neck > 40 cm). *Known publication; not re-fetched in this research.*
15. Updated STOP-Bang questionnaire (43 cm / 41 cm neck; scoring), e.g. https://www.entnet.org/wp-content/uploads/2021/04/rhinoplasty_stop-bang-v1.pdf and https://www.psychdb.com/_media/sleep/stop-bang.pdf
16. Chung F, Abdullah HR, Liao P, "STOP-Bang Questionnaire: A Practical Approach to Screen for Obstructive Sleep Apnea", *Chest* 2016;149(3):631–8. https://pubmed.ncbi.nlm.nih.gov/26378880/
17. POSSUM and P-POSSUM equations: review https://www.ncbi.nlm.nih.gov/pmc/articles/PMC10779462/ ; Prytherch et al. 1998 https://pubmed.ncbi.nlm.nih.gov/9752863/
18. Clinical Frailty Scale, Dalhousie: https://www.dal.ca/sites/gmr/our-tools/clinical-frailty-scale.html ; permission for use https://dal.ca/sites/gmr/our-tools/permission-for-use.html
19. STOP-Bang ownership and licence (Dr F Chung and UHN; licence beyond education via stopbang.ca), as stated on reproductions: https://www.beaconhealthsystem.org/wp-content/uploads/2023/11/Stop-Bang-Questionnaire.pdf ; https://site.thoracic.org/assemblies/srn/sleep-related-questionnaires/stop-bang
20. REDCap Shared Library: https://projectredcap.org/?p=11 ; import guide (no changes to imported instruments) https://redcap.smhs.gwu.edu/sites/g/files/zaskib651/files/2021-07/Import%20from%20the%20REDCap%20Shared%20Library%20Adaptive%20Instruments.pdf
21. MDCalc FAQ, contributors and reviewers https://www.mdcalc.com/faq/about-clinical-decision-support-tools/what-does-it-mean-that-a-calculator-has-been-contributed-by-and-reviewed-by-someone ; quality rating system https://blog.mdcalc.com/post/quality-rating-system
22. Kotzé A et al., "Development, external validation and integration into clinical workflow of machine learning models to support pre-operative assessment in the UK", *Anaesthesia* 2026;81(2):201–212 (online Sep 2025). https://eprints.whiterose.ac.uk/232273
23. GIRFT, Pre-operative assessment guidance, July 2023. https://gettingitrightfirsttime.co.uk/wp-content/uploads/2023/07/GIRFT-Pre-operative-assessment-guidance-FINAL-V1-July-2023.pdf
24. NHS England, "Early screening, triaging, risk assessment and health optimisation in perioperative pathways" (May 2023, updated May 2025). https://www.england.nhs.uk/long-read/earlier-screening-risk-assessment-and-health-optimisation-in-perioperative-pathways/
25. Chelsea and Westminster NHS FT, digital POA triage case study (2023). https://www.chelwest.nhs.uk/about-us/news/news-archive-old/2023/digital-preoperative-assessment-triage-at-chelsea-and-westminster-hospital-nhs-foundation-trust-published-as-exemplary-case-study-by-nhs-england
26. "Smart waiting lists: from one-size-fits-all to personalised pre-op care" (East Kent RAG tool). https://healthcare-in-europe.com/en/news/perioperative-patient-pathways-smart-waiting-list.html
27. SORT, Surgical Outcome Risk Tool. https://www.sortsurgery.com (as recommended in [23])
28. CPOC, "Preoperative assessment and optimisation for adult surgery" (2021). https://cpoc.org.uk/media/2071 (quoted through search; the PDF wasn't reachable directly)
29. Epic open.epic, PATIENT_FYI_FLAGS table. https://open.epic.com/EHITables/GetTable/PATIENT_FYI_FLAGS.htm
30. Alberta Connect Care manual, "Flags (FYI)". https://manual.connect-care.ca/tools/flags
31. HL7 FHIR R4, Flag resource. https://hl7.org/fhir/R4/Flag.html
32. ASA codes: LOINC 97816-3 https://loinc.org/97816-3 ; SNOMED CT ASA value set https://hl7.org/fhir/us/hai/STU2/ValueSet-2.16.840.1.113883.13.10.html
33. "Data-Driven System for Perioperative Acuity Prediction" (Vanderbilt). https://ir.vanderbilt.edu/items/b3d5c76a-a8e7-48ef-9980-1a38553a7392
34. "Development and validation of a machine learning ASA score" (J Clin Anesth 2023). https://einstein.elsevierpure.com/en/publications/development-and-validation-of-a-machine-learning-asa-score-to-ide
35. Simpson, Perioperative Medicine Leads Day, RCoA/CPOC, 2020 (green/amber/red pre-assessment). https://www.cpoc.org.uk/sites/cpoc/files/documents/2020-03/Simpson%20-%20Periop%20Med%20Leads%20Day%20RCoA.pdf (quoted through search)
36. MHRA, "Medical device stand-alone software including apps". https://assets.publishing.service.gov.uk/media/6abf6c8c943e890cb55e5e52/Medical_device_stand-alone_software_including_apps__including_IVDMDs.pdf (from [docs/market](../market/tender-research.md))
37. Queensland Clinical Excellence, triage guidelines for pre-anaesthetic evaluation. https://clinicalexcellence.qld.gov.au/sites/default/files/docs/resourses/pre-anaesthetic-evaluation-framework/triage/triage-guidelines-pre-anaesthetic-eval.pdf
