# S46 test plan: observations

**Changed after test report 1 (29 Oct):** F-1 to F-3 are fixed (ECG and urinalysis save; T-03 step 4 is now refused at once with no warning; T-10 step 4's "-1" is kept and refused). Also: the ranges boxes on the Settings page are plain text boxes; "Measured at" is the time of Save unless you change it; "1e2" is refused as not a number; a stale tab's Correct or Entered in error shows the message and reloads the list. **Restart `make dev` before retesting** (the Go fix for F-1 needs it).

**Changed after test report 2 (29 Oct):** F-4 is fixed: after a correction or an entered in error is saved (or Back is clicked), no "Say why…" message stays under the reading. A range with a missing, empty or non-number limit is refused naming its row ("Weight: give all four limits as numbers."). The Measured at box says "The time you save it, unless you change it." until you change it. **Restart `make dev` before retesting again** (the ranges message is Go's).

**What's new:** clinicians record readings on an episode (height, weight, blood pressure, heart rate, SpO₂, temperature, respiratory rate, blood glucose, peak flow, and ECG and urinalysis notes). A reading outside the usual range warns but can still be saved; outside the hard limits it's refused. A reading is never changed or deleted: it can be **corrected** (a new value, with a reason) or marked **entered in error** (with a reason), and the old one stays in the history, struck through. The episode page and the POA Summary show the **hospital BMI**, worked out from the latest height and weight. A trust's admins can change the ranges for their trust on its Settings page. Patients never see any of this.

**Changed after test report 5 (31 Oct):** **Restart `make dev` before retesting** (the fixes are in Go too). What to retest: **T-22** (two admins editing the ranges), **T-23** (a note too long), **T-24** (leaving a page mid-request in the Audit log), **T-25** (observations off, no refused rows), and T-09, T-10, T-16, T-18, T-19 and R-05 as regressions.

**Changed after verification 2 (31 Oct):** web code only, so `make dev` needn't restart. What to retest: **T-19** (a Settings page left open), **T-20** (a five-digit year), **T-21** (the POA Summary when readings don't load), and T-10, T-13, T-15, T-18 and R-04 as regressions.

**Changed after verification 1 (30 Oct):** **Restart `make dev` before retesting** (the fixes are in Go too). What to retest:
- **T-13 below:** a correction outside the usual range now asks first.
- **T-14:** marking a correction entered in error doesn't bring the old value back, and the form says so.
- **T-15:** a time before the patient's date of birth is refused.
- **T-16:** a long unbroken note wraps.
- **T-17:** switching POA Summary tabs writes no audit rows.
- **T-18:** a trust that changes one range saves only that one.

## Setup

In Rahul's own terminal, from the repo root:

```sh
docker compose up -d db
make migrate seed
make dev
```

Leave `make dev` running. Open **http://localhost:3000**. The sign-in page lists the seeded users; click one to sign in as them (Sign out, top right, to switch). The users for this plan:

| User | Who |
| --- | --- |
| **Cara Clinician** | clinician, Hospital A1 only |
| **Val Clinician** | clinician, Hospital A2 only (same trust) |
| **Bea Author** | super clinician, the whole of Trust B |
| **Ada Admin** | admin, the whole of Trust A (Settings) |
| **Sue Super** | super clinician, Hospital A1 only |
| **Sam Superadmin** | platform admin (the console at `/admin`) |

**Test data:** made-up patients only, e.g. surname `MT S46 <time>`. To make an episode: as Cara, Hospital A1 → **Episodes** → **New episode**: first name, surname, date of birth (an adult), sex, mobile `07700 900` + 3 digits, an email like `mt-s46-<time>@example.com`, any published HQ (e.g. **Demo pre-op HQ**), **Create**. The episode's page opens; the **Observations** card is below **Tasks**.

**Patient reported BMI** (for T-06): the patient's invite is on **http://localhost:3000/qa/messages** (no sign-in needed locally). Open the newest invite to the episode's email at phone size (390×844), type the date of birth, and in the Demo pre-op HQ answer the height and weight questions (e.g. 170 cm, 82 kg). Sending isn't needed.

## Test cases

### T-01 Record each kind (OBS-01, OBS-02)
1. As Cara, open a new episode. The Observations card says "No readings yet." and "Hospital BMI: Not measured yet (needs a height and a weight)".
2. **Add a reading** → **Height** → 175 → **Save reading**. Then each of: Weight 80.4; Blood pressure 128 / 82; Heart rate 72; SpO₂ 97; Temperature 36.8; Respiratory rate 16; Blood glucose 5.6; Peak flow 450; ECG "Sinus rhythm (made-up)"; Urinalysis "NAD (made-up)".
3. **Expected:** each appears at the top of the list, newest first, as "Weight: 80.4 kg", "Blood pressure: 128/82 mmHg", "Blood glucose: 5.6 mmol/L", "Temperature: 36.8 °C", the ECG's text, and so on, each with "Measured <date, time> · entered by Cara Clinician". The labels show the units ("Weight (kg)", "Systolic (mmHg)"). After Save the same kind stays chosen and the boxes are empty, ready for another reading.
4. **Hospital BMI** shows 26.3 (80.4 / 1.75²) with when it was measured.

### T-02 An earlier time (OBS-02)
1. Add a weight with **Measured at** set to yesterday, 08:00.
2. **Expected:** it's listed below today's readings (newest measured first) with yesterday's time. The hospital BMI still uses today's weight (the latest measured).
3. Try **Measured at** tomorrow. **Expected:** refused: "The time it was measured can't be in the future."

### T-03 Soft warnings and hard limits (OBS-03)
1. Heart rate **220** → Save reading. **Expected:** a yellow box: "Heart rate 220 beats/min is outside the usual range (50 to 120). Save it anyway?" Nothing is saved yet. **Change it** closes the box; **Save anyway** saves it.
2. Weight **-3**. **Expected:** no warning box; refused in red: "Weight must be from 1 to 400 kg."
3. Heart rate **400** → refused ("Heart rate must be from 20 to 300 beats/min."). Heart rate **72.5** → refused ("Heart rate: enter a whole number.").
4. Blood pressure 80 / **120**. **Expected:** at once, with no "usual range" box first: "The diastolic must be lower than the systolic."
5. Empty boxes, or "eighty". **Expected:** "Enter the weight." / "Weight: enter a number." "82,5" (comma) is read as 82.5.
6. An ECG with an empty note. **Expected:** "Enter the ECG note."

### T-04 Correct a reading (OBS-02)
1. On T-01's weight, **Correct**. Type 78.0, leave the reason empty, **Save correction**. **Expected:** "Say why the reading is corrected."
2. Reason "Made-up: scale not zeroed" → Save correction.
3. **Expected:** the old "Weight: 80.4 kg" stays, struck through, with a **Corrected** badge and "Corrected to 78.0 kg"; the new one reads "corrected by Cara Clinician: Made-up: scale not zeroed" and keeps the old one's time. The old one has no Correct or Entered in error buttons. **Hospital BMI** follows (25.5).
4. Correct the correction (e.g. 78.2, a reason). **Expected:** works; both earlier values stay in the history.

### T-05 Entered in error (OBS-02)
1. On a blood pressure, **Entered in error**; Mark it with an empty reason. **Expected:** "Say why the reading was entered in error."
2. Reason "Made-up: another patient's" → **Mark entered in error**.
3. **Expected:** struck through, an **Entered in error** badge, "Entered in error: marked by Cara Clinician on …: Made-up: another patient's". No buttons on it any more.
4. Mark the latest **height** entered in error. **Expected:** the hospital BMI uses the earlier height if there is one, else says "Not measured yet…".

### T-06 POA Summary (OBS-04)
1. On an episode with readings and the patient's own height and weight (Setup), open **POA Summary**.
2. **Expected:** "Patient reported BMI" (e.g. 28.4) and below it "Hospital BMI" (e.g. "26.3 (measured …)"), on both tabs. **Print** keeps both.
3. An episode with no readings: Hospital BMI says "Not measured yet (needs a height and a weight)".

### T-07 Who can see and record (tenancy)
1. As **Sue Super** (Hospital A1), open Cara's episode. **Expected:** the card works the same (any clinician at the hospital records).
2. As **Val Clinician** (Hospital A2), paste that episode's URL (change nothing else). **Expected:** "You don't have access to this hospital." or Episode not found; no readings.
3. As **Bea Author** (Trust B), paste the URL. **Expected:** no access; nothing from Trust A shows.
4. As the **patient** (the invite link, phone size): nothing about observations anywhere.

### T-08 Cancelled and archived episodes
1. Cancel an episode with readings (Cancel episode, a reason). **Expected:** the readings and hospital BMI still show; no Add a reading, no Correct or Entered in error buttons.

### T-09 Observations switched off
1. As **Sam Superadmin**, `/admin` → Trust A → turn **Observations** off for **Hospital A1** → save.
2. As Cara, open an episode. **Expected:** no Observations card; the POA Summary has no Hospital BMI line. Other cards are unchanged.
3. Turn it back on. **Expected:** the card and the readings are back.

### T-10 The trust's own ranges (S46 answers 1)
1. As **Ada Admin**, **Settings** → the whole trust → **Observations** section → **Observation ranges**: a row per measurement with "Usual from", "Usual to", "Refused below", "Refused above", greyed out with "Use Preopsis's default" ticked.
2. Untick it; set **Heart rate**: usual 50 to 100, refused below 30 and above 150. Save.
3. As Cara: heart rate 110 warns ("…usual range (50 to 100)"); 160 is refused ("Heart rate must be from 30 to 150 beats/min.").
4. As Ada, try heart rate "Refused below" **-1** → Save. **Expected:** refused: "Heart rate: a limit can't be below 0." Try usual from 20 with refused below 30. **Expected:** "Heart rate: the usual range must be inside its hard limits, lowest first." Try SpO₂ refused above 101. **Expected:** "SpO₂: a limit can't be above 100%."
5. As Ada, pick **Hospital A1** in the settings picker. **Expected:** Observation ranges shows "Set for the whole trust", not editable.
6. Tick **Use Preopsis's default** again and save: Cara's heart rate 110 no longer warns.

### T-11 Reload, deep link, Back
1. On an episode with readings, reload. **Expected:** the same list, struck-through ones included, and the same BMI.
2. Open POA Summary, then Back. **Expected:** the episode page with the card.

### T-12 Phone size, for a clinician on a small screen (390×844)
1. As Cara at 390×844, add a blood pressure and correct it. **Expected:** the boxes wrap; nothing is cut off; the buttons are reachable.

### T-13 A correction outside the usual range (OBS-03, verification 1)
1. As Cara, on a weight of 80: **Correct**, type **380**, a reason, **Save correction**.
2. **Expected:** in the form, a yellow box: "Weight 380 kg is outside the usual range (35 to 200). Save it anyway?" Nothing is saved, and the Hospital BMI hasn't changed. **Change it** closes the box and keeps what's typed.
3. Change the value, go on to **Save anyway**. **Expected:** saved, and the BMI follows. A value inside the range (e.g. 78) saves with no box. A value outside the hard limits (e.g. 500) gets no box, and is refused in red.

### T-14 A correction marked entered in error (verification 1)
1. On a reading that's a correction (it reads "corrected by …"), click **Entered in error**.
2. **Expected:** the form says "The value it corrected won't count again. To put a value back, use Correct." A reading that isn't a correction doesn't say this.
3. Mark it, with a reason. **Expected:** both it and the value it corrected are struck through and nothing counts: with no other weight, the Hospital BMI says "Not measured yet…". This is what Rahul chose; it's not a fault. Adding a new reading counts again.
4. On another correction, **Correct** it again instead. **Expected:** the new value counts.

### T-15 Measured at before the date of birth (verification 1)
1. Add a weight with **Measured at** in the year **1926** (the patient is younger than that) or **0026**.
2. **Expected:** refused in red: "The time it was measured can't be before the patient's date of birth." Nothing is saved.
3. A time on the patient's date of birth itself saves.

### T-16 A long unbroken note (test report 3)
1. Add an ECG with a note of 400 "x" characters and no spaces.
2. **Expected:** it wraps inside the card, all of it readable, at desktop width and 390×844. The buttons stay on screen.

### T-17 POA Summary tabs and the audit log (verification 1)
1. As Cara, open a POA Summary, then switch between its two tabs a few times.
2. **Expected:** the Hospital BMI line stays on both tabs, with no gap while switching.
3. As Ada, **Audit**: no extra "Opened an episode's observations" rows from the tab switches.

### T-18 One changed range (verification 1)
1. As Ada, **Settings** → Observation ranges: untick the default, change only **Heart rate**, Save.
2. **Expected:** saved. `GET /api/o/<trust>/settings`'s `observations.ranges` holds only `heartRate`. The other rows still show Preopsis's numbers.
3. Tick **Use Preopsis's default** again and save, to put it back.

### T-19 A Settings page left open (verification 2)
1. As Ada, give the trust its own Heart rate range (T-10) and Save. Save again with no edits. **Expected:** "Nothing has changed."
2. Open the same Settings page in a second tab. There, change Heart rate **Usual to** to 105 and Save.
3. Back in the first tab, without reloading, change only "Mark not responding on day", and Save.
4. Reload. **Expected:** Heart rate Usual to is still **105**; the other setting is saved.
5. Put both back to the default.

### T-20 A five-digit year (verification 2)
1. As Cara, add a weight, and in **Measured at** type the year as **20266**. Save reading.
2. **Expected:** refused in red: "When was it measured? Enter the date and time." Nothing is saved, and there's no console error.

### T-21 The POA Summary when readings don't load (verification 2)
1. If your tools can make `…/observations` fail on a POA Summary: **Expected:** the Hospital BMI row says "Couldn't be loaded". If they can't, mark it Blocked: a J12 test covers it.
2. With observations off (T-09): still no row.

### T-22 Two admins editing the ranges (test report 5)
1. As Ada, give the trust its own Heart rate range. Open Settings in two tabs.
2. Tab 2: change Heart rate **Usual to**, and Save.
3. Tab 1, no reload: change only **Weight Usual to**, and Save.
4. **Expected:** refused: "Someone else changed the observation ranges. Reload to see them, then make your change again." Nothing is stored from tab 1. Reload: Heart rate has tab 2's value, Weight its old one. Making the change again then saves.
5. Repeat with both tabs changing the **same** row: the second is refused too. And in a stale tab, tick **Use Preopsis's default**: refused too.
6. In a stale tab, change only another setting (T-19): it still saves.
7. Put everything back to the default.

### T-23 A note too long (test report 4)
1. Paste a note of about 2,200 characters into an ECG, **Save reading**.
2. **Expected:** "The note is too long (2000 characters at most)." Nothing is saved, and all of what was pasted is still in the box.
3. A note of exactly 2,000 characters saves, whole. On **Correct**, the same limit.

### T-24 Leaving a page mid-request (test report 5)
1. If your tools can abort a request while it runs: **Expected:** the Audit page shows that row as **Cancelled** (grey), not Refused (500). If they can't, mark it Blocked: `TestAuditCallerLeft` covers it.
2. Older rows from before this change still say Refused (500): that's expected, as the log is never changed.

### T-25 Observations off, no refused rows (verification 2)
1. Turn Observations off for Hospital A1 (T-09). As Cara, open an episode and its POA Summary.
2. **Expected:** no card, no Hospital BMI row, and no `…/observations` request in the network log. As Ada, **Audit** has no new refused "Opened an episode's observations" rows.
3. Turn it back on.

## Regression

- **R-01 (S44b):** on the POA Summary, add an anaesthetist's ASA grade, then mark it entered in error with a reason: works as before.
- **R-02 (S43):** a task added on the episode still shows, completes and re-opens (the Tasks card above Observations).
- **R-03 (S38/S28):** the POA Summary's Patient reported BMI is the same as before for the patient's height and weight.
- **R-04 (S10):** Ada changes another setting (e.g. "Mark not responding on day") and saves: it's saved; the Observation ranges row is unchanged.
- **R-05 (S11):** as Ada, **Audit**: the readings show as "Recorded an observation", "Corrected an observation", "Marked an observation entered in error" and "Opened an episode's observations", naming the episode, with no values.

## Out of scope

- Assessments, STOP-Bang from observations, and which BMI feeds the scores (S47).
- The ranges' numbers themselves: they're made up until a clinician checks them.
- Units other than mmol/L and °C (nothing converts yet).
- Observations on printed documents (C9) and charts or trends.
- J7's occasional failure (a random mobile seen before; pending.md).
