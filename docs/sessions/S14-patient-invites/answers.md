# S14: answers

Asked by the conductor (`/run-sessions`) on 12 Oct 2026, before the run started.

1. **What the patient API returns before and after the date-of-birth check.**
   - **Answer (recommended):** before the check, only the hospital's name and the Continue/date-of-birth screen. After it, the HQ, without the date of birth or any other identifier the patient didn't type.

## Asked during the run (13 Oct, by the conductor, during test round 1)

2. **After the date-of-birth check, may the patient see their own first name, full name, age and sex?** (Built that way; never the DOB, NHS number, hospital number, mobile, email or record ID.)
   - **Answer: yes, as built** (recommended). The check has passed, so it's the patient; the name reassures them, and age and sex drive the questions.
3. **Invite links with no expiry date** (as long as the episode exists; DOB check, 5-attempt lock, a resend replaces older links).
   - **Answer: OK until C3** (recommended). When C3 adds episode statuses, closing an episode ends its links.
4. **Patient sessions are stateless cookies** (found by test report 1): Sign out only clears the browser's cookie, and a Resend or a lock doesn't end a session already opened.
   - **Answer: fix now** (recommended). Tie each session to its invite in the database so Sign out, Resend and a lock end it at once (one DB check per patient request; still no server memory, so Lambda-safe). Also add `Cache-Control: no-store` on patient data and a `Referrer-Policy` on the `/i/<token>` page.
5. **After test report 4 (the 4-round limit):** the patient's browser Back or a reload within ~0.6 s of typing loses that last input, silently (fix: send the waiting save on page unload, e.g. `pagehide` + a `keepalive` fetch).
   - **Answer: skip it for now; it's an edge case** (Rahul, 13 Oct). Log it in `pending.md` as a follow-up; verify and commit S14 as is.
