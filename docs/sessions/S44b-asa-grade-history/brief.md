# S44b: The anaesthetist's ASA grade as a history

**Plan:** C6 ([c6-scores-asa-triage.md](../../plans/c6-scores-asa-triage.md)), a follow-up session added on 27 Oct, before S45.
**Requirements:** Rahul's answer of 27 Oct to S43's open item "The anaesthetist's ASA grade cleared after Ready for admission" ([answers.md](answers.md); decisions.md, 27 Oct).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S44a verified.

**What exists today:** the anaesthetist's grade is one column, `episodes.anaesthetist_asa` (1–6), set or cleared through the episode PATCH (`internal/httpapi/episodes.go`) from the POA Summary. S43's rule action `requireAnaesthetistAsa` (`episodes.anaesthetist_asa_required_by`) refuses a move into Ready for admission while no grade is saved. But once there, the grade can be cleared, and Admit is still accepted; a cancelled episode's grade can't be added, so its Undo stays greyed out.

**Rahul's answer (27 Oct):** nobody may change or clear a grade once given, in any status, because that's too risky. Grades are **incremental**: each one is kept with who gave it and when; a later one is added on top, and the earlier stays visible.

**Ask Rahul first:** nothing more; the details were answered on 27 Oct:
- **A reason for a re-grade:** the first grade needs none; every later grade must give one (e.g. "New echo result").
- **Entered in error:** a grade given by mistake can be marked "entered in error", with a reason. It stays visible, struck through, with who marked it, when and why. It no longer counts, and the one before it (if any) counts again.

## Build (test-first)

- **Migration:** a table of grades per episode: grade (1–6), who, when, reason (required unless it's the episode's first), and entered-in-error (who, when, reason). Carry each existing `episodes.anaesthetist_asa` over as the episode's first grade (no user: "Recorded before the history"), then drop the column, or keep it only if something must still read it (say which in the build notes). The grade that **counts** is the latest one not entered in error.
- **Routes** (each audited, IDs only, and with the 5 cases: happy path, no sign-in, wrong role, another trust's episode, invalid input; the same role as can set the grade today):
  - **add a grade:** 422 without a reason when the episode already has a grade that counts; a stale view (a grade added in another tab meanwhile) → 409, as the app's other stale saves;
  - **mark a grade entered in error:** a reason is required; once marked, it can't be unmarked;
  - **no edit and no delete** of a grade, anywhere. The episode PATCH no longer accepts the grade (400 if sent).
- **S43's requirement reads the grade that counts.** Marking the **only** counting grade entered in error is refused (409, "Add the correct grade first, or move the episode back from Ready for admission.") while a rule requires the grade and the episode is at Ready for admission or later. This closes all three gaps from pending.md: the grade can't go missing, so Admit can't be reached without one, and a cancelled episode's Undo keeps working.
- **The POA Summary** (and anywhere else that shows the anaesthetist's grade: the episode page's requirement line, the clinician's summary, the CSV export if it has one): the grade that counts, and a history list underneath: grade, who, when, reason; entered-in-error ones struck through, with who, when and why. Add a grade (the reason box appears and is required once a grade exists); "Entered in error" on each grade, with its reason.
- **E2E:** extend J11b: the anaesthetist's grade is added, re-graded with a reason, and an error marked; Ready for admission and Admit as before.
- **Docs:** pending.md (strike the S43 item), decisions.md, CLAUDE.md's content model (one line), the C6 plan's Status.

## Not in this session

- A history for any other clinician entry (only the anaesthetist's ASA grade is decided).
- The suggested ASA (worked out from answers; unchanged).

## Manual test focus

1. Add a first grade (no reason asked), then a second (a reason required); both show, and the latest counts.
2. Mark the latest entered in error: it's struck through, and the earlier one counts again.
3. At Ready for admission with a rule requiring the grade: marking the only grade in error is refused, with the message; Admit works.
4. No way to edit or delete a grade, in the app or through the API (the episode PATCH refuses it).
5. Another trust's clinician gets a 404; a hospital A2 clinician can't see A1's grades.
