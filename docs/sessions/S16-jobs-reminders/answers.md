# S16: answers

Asked by the conductor (`/run-sessions`) on 12 Oct 2026, before the run started.

1. **Send reminders only between 09:00 and 19:00 in the hospital's time zone?**
   - **Answer: no, send at any time** a reminder is due. (The recommendation had been 09:00–19:00.)
2. **What stops reminders?**
   - **Answer: starting stops them.** Reminders stop once the patient opens the HQ and answers anything. (The recommendation had been that only submitting stops them.)

## Asked during the run (13 Oct, by the conductor, during test round 1)

3. **The jobs write no audit-log rows**; what they did is an "Automatic" note on the episode.
   - **Answer: episode notes are enough** (recommended), as built.
4. **Answers a clinician types in on the patient's behalf count as "started" and stop reminders.**
   - **Answer: yes** (recommended), as built.
5. **A locked episode can still be reached through an S15 sign-in link** (found by test report 2): sign-in links go to the patient's own contact and keep their own 5 tries, while the staff banner says "every link is locked".
   - **Answer: keep it; reword the banner later** (recommended). Sign-in links are a fair way back in. Log the banner wording ("invite links locked") in `pending.md`; no extra round.
