# S20: Every status, cancel and re-open

**Plan:** C3 ([c3-episode-lifecycle.md](../../plans/c3-episode-lifecycle.md)), part 3 of 5.
**Requirements:** EPI-01 (the admitted and discharged dates), EPI-03, EPI-07, and the statuses in [saas-requirements.md §5.2](../../saas-requirements.md).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S19 verified.

**Ask Rahul first:**
1. **The transition table.** Recommended moves and who may make them (role ladder: clinician < super clinician < admin):

   | From | To | Who |
   | --- | --- | --- |
   | `hq_not_complete` | `ready_for_review` | The patient's submit, or staff submitting on their behalf (as today) |
   | `ready_for_review` | `ready_for_poa` | Completing the review (as today) |
   | `ready_for_poa`, `poa_complete`, `on_hold`, `not_ready`, `ready_for_admission` | any other of these five | A clinician (as today) |
   | `ready_for_admission` | `admitted` | A clinician, with the admission date |
   | `admitted` | `discharged` | A clinician, with the discharge date |
   | any status before `discharged` | `cancelled` | A clinician, with a reason |
   | `ready_for_poa` … `ready_for_admission` | `ready_for_review` (**re-open**) | A super clinician, with a reason |

   - `recovery` (PROMs, C9) isn't built.
   - `archived` is S22's.
   - Auto-triage's moves are C6's.
2. **Correcting a mistake** (e.g. Admitted pressed by error):
   - (a) A super clinician can move an episode one step back, with a reason.
   - (b) No going back except re-open; anything else is fixed by cancelling.

   **Recommended: (a),** limited to `admitted → ready_for_admission` and `discharged → admitted`.
3. **Cancel reasons:**
   - a fixed list: procedure cancelled, patient withdrew, moved to another hospital, other;
   - "Other" needs a sentence.

   **Recommended: yes.** A cancelled episode can't be un-cancelled; it can only be archived (S22).
4. **When is a reason required?**
   - Recommended: required for **cancel, re-open and a step back**.
   - Optional for every other manual move.
5. **What cancelling does to the patient** (decided 13 Oct: closing an episode ends its links):
   - their invite link and patient sessions for that episode end;
   - follow-up jobs stop;
   - their home shows the episode as "Cancelled by the hospital", with no form.

   **Recommended: yes.**
6. **Re-open** (EPI-07) clears the review stamp and unlocks the clinician's answers. The answers and validations stay, and the episode is back in Ready for review.
   - The patient's form stays sent: re-opening doesn't ask the patient again.
   - **Recommended: yes.** Re-opening an auto-triage is C6.

## Build

- **The transition table in Go** (`internal/episode`):
  - `allowed[from] → []to`, with the least role and whether a reason is required.
  - Unit-tested exhaustively, test-first: every pair of statuses is checked.
  - Submit and complete-review go through it too.
- **Status events (EPI-03):** every change records who, when, from, to and why, and automatic changes name their rule. Either structured columns on `episode_events` or a new `episode_status_events`. General notes still show a readable line.
- **An episode revision.** Writes send the revision the page loaded, and a stale one gets 409. This fixes two items in [pending.md](../../pending.md) §1a:
  - two PATCHes overwriting each other's ASA grade;
  - the 409 that compared against the wrong status.
- **Routes:**
  - `PATCH …/episodes/{eid}/status` (to, reason, date where needed);
  - `POST …/episodes/{eid}/cancel`;
  - `POST …/episodes/{eid}/reopen`.

  Each gets the five integration cases, plus **every illegal transition → 409**, **a role without the right → 403**, **cancel without a reason → 400**, and **a stale revision → 409**. All are audited (S11).
- **Episodes:** `admitted_at`, `discharged_at` and `cancel_reason`, plus the new status keys in the CHECK and the labels.
- **Screens:**
  - the status control shows only the moves this user may make from here;
  - the Admit and Discharge dialogs ask for the date, and the Cancel and Re-open dialogs ask for the reason;
  - a status history on the episode page.
- **The patient side of cancel,** as answered.
- **End-to-end:** **J9b**, cancel with a reason. J9 (the full lifecycle) comes in S22, once archive exists.

## Not in this session

- Archive, un-archive and auto-archive (S22).
- Tasks and worklists (S21).
- Status messages to the patient (S21).
- Auto-triage (C6), `recovery` (C9).

## Manual test focus

1. Walk an episode from creation to Discharged, entering the admission and discharge dates. The history shows each move, who made it and when.
2. The status control offers only legal moves. A move made in a stale tab gets a clear "changed since you opened it".
3. **Two tabs:** set the Nurse ASA in one and the Anaesthetist ASA in the other. Neither is lost; the second gets the stale message and keeps its input.
4. Cancel with each reason. "Other" needs a sentence. The patient's link stops working, and their home shows the episode as cancelled.
5. Re-open as Sue Super: the review is unlocked and the reason is in the history. As Cara Clinician, there's no Re-open.
6. A step back (if answered (a)) works only for a super clinician, with a reason.
7. Trust B staff can't change Trust A's episode statuses.
