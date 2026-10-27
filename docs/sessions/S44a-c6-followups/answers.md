# S44a: answers

Asked by Claude in chat on 27 Oct 2026, after the S38–S44 run, from the open items in pending.md. Answered by Rahul the same day.

1. **A score input hidden by a condition** (a question or a whole Question Set): **it adds 0 points.** A shown question left unanswered still makes the score Incomplete.
2. **A follow-up set a clinician's correction brings back,** after the patient hid it before sending: **it starts empty.** On send, keep only the answers the patient's form showed. Rahul: the clinician answers the follow-up during validation, and nothing goes back to the patient.
3. **A patient set's condition on a clinician-only score:** **Publish refuses it**, with a message saying how to fix it.
4. **The thank-you page:** **one neutral wording** for every send, promising no review.
5. **An Incomplete score from clinician-entered inputs stops auto-triage:** **rightly so**: the clinician enters them and carries on. But **tell the author** when designing the HQ that auto-triage won't work for it, naming the score and the inputs, and let them go ahead anyway.
6. **The rules' suggested-status line:** **hide it once the episode reaches or passes that status** (and when cancelled or archived).
7. **The anaesthetist's ASA grade:** see [S44b's answers](../S44b-asa-grade-history/answers.md).
8. **S39's "E" rule changed draft test cases' expected lines:** for now, **update the drafts automatically** (local, made-up data). For production, see 9.
9. **Production policy, when a release changes our clinical logic so live HQs would give different results:** **both** a regression check before every release (every trust's saved test cases, published and draft, run against the new code, and each changed HQ reported, signed off and told to the trust) **and pinning** for changes that affect results (a published HQ keeps the logic it was published with until the trust publishes a new version). For C10, not this session.
10. **The defects found by the verifiers:** fix **all four** before S45 (S42's message sent twice, S44's Complete review race, S41's rules save racing Publish, S38's score picker offering repeating-group questions).
11. **Packaging:** **two sessions before S45:** S44a (this one) and S44b (the ASA grade history).
12. **S45's evaluate calls under database locks:** **measure in S45, then decide.** If a call takes more than about 1 s on QA, the builder asks before moving the calls out of the locks.
13. **The local database's huge audit log:** **reset the local database, and harden the test.** *Claude's correction, the same day:* don't cap what a "Listed episodes" audit row records, because the audit log must stay complete. Only the test reads the CSV as a stream.
14. **Small improvements:** **all four:** the two clearer messages, the ASA guide nudges for authors, the suggested ASA in the right places, and the age in the hospital's time zone.
15. **Saved option lists:** **keep copies** (as built).
16. **"Also when" for authors' own bands:** **add it to S44a.**
