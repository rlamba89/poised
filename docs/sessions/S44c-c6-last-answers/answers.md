# S44c: answers

Asked by Claude in chat on 28 Oct 2026, after the S44a–S44b run, from the open items in pending.md. Answered by Rahul the same day.

1. **A chained follow-up on the patient's own screen** (B Yes shows A, A Yes shows C; B back to No: the screen still showed C, but the send dropped A and C): **clear on hide.** The patient's form clears an answer as soon as its question is hidden, so the screen and what's stored always agree, even though a patient who flips a question loses the follow-up's answers.
2. **A held auto-triage message lost after Undo or un-archive** (the HQ's "wait for Checked"): **keep it held.** When a change lands back on the auto-triaged status, the message is still held, so Checked sends it.
3. **A High STOP-Bang and the default "Low risk" rule:** **hold them back.** In the platform's default rules, a High STOP-Bang stops the Low risk auto-triage, so a clinician sees the patient first. Authors can still change it per HQ; only new HQs (and new versions of HQs without rules) get the defaults.
4. **STOP-Bang for a patient whose sex is "Other":** **Incomplete**, as for "Not known".
5. **A question the patient never saw** (a clinician's correction on Validate shows a set the patient's form hid): Rahul: it's obvious the clinician answers it, and nothing goes back to the patient. **The app doesn't need a "patient answered" line for it:** the clinician is asked to answer it, and if it isn't answered the review can't go forward (S44 already refuses Complete review while a shown set isn't validated).
6. **Re-adding the anaesthetist's grade that already counts** (e.g. a second anaesthetist agreeing): **allow it** (as built in S44b: "confirmed as 3", with a reason).

**How Rahul wants questions asked** (28 Oct): in simple English, short, with a little context (what the thing is, and an everyday example), then one simple question. No dense or technical wording.
