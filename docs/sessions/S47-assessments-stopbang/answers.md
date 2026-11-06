# S47 answers

Rahul, 5 Nov 2026 (asked at the start of `/build-session S47`). All four are the recommended options.

1. **A clinician adds an assessment** from the hospital's list on the episode page ("Add assessment → STOP-Bang") (option a). Adding them automatically, or per procedure, can follow once trusts say which procedures need which assessments.
2. **STOP-Bang's neck item is answered Yes or No by the clinician**, in the published wording (collar over 43 cm for men, 41 cm for women) (option a). There's no new neck observation.
3. **A trust setting, `obs_precedence`, on by default, chooses which BMI feeds the score** (option a): on, the hospital BMI when there is one, else the patient's; off, the patient's. Each item says which it used.
4. **An HQ that already holds STOP-Bang opens the assessment with every item filled, and the clinician still confirms** (option a). Confirm records who and when.

## After verification 3 (Rahul, 6 Nov 2026)

All three are the recommended options. They're to be built into S47 before it's committed ([verify.md](verify.md), verification 3).

5. **STOP-Bang's pre-fill uses the clinician's copy of a Question Set as soon as they've started it,** not only once it's validated, like the rest of the episode page. This changes the brief's "the clinician's validated copy".
6. **A confirmed assessment says when a value it used has changed since** (e.g. "The hospital BMI is now 24.7 (37 when confirmed)"). Nothing changes until a clinician confirms again.
7. **Age and sex are frozen at Confirm** (stored with the confirmed answers), so a later correction to the patient record never moves a confirmed score. The card says when the record has changed.

## After test report 4 (Rahul, 6 Nov 2026)

8. **When the only change since Confirm is the patient's record (date of birth or sex), leave it as built** (the recommended option). The yellow box stays and the score stays as confirmed until a clinician changes another item and confirms again. This should be rare. A button that takes the new values is in pending.md if clinicians ask for one.
