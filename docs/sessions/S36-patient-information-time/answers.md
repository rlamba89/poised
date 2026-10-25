# S36 answers

Asked by the conductor in chat, 25 Oct 2026 (Rahul).

1. **Where patients see an episode's Information:** **(a)** a section on the episode's card on their home page.
2. **A trust changes a procedure's videos or leaflets:** **(b), not the recommendation: the list is copied onto the episode when it's made**, so a booked patient keeps the list from when they were booked.
3. **Estimated time:** **(a)** 15 seconds per question the patient sees, plus each video's length, rounded up to whole minutes. **Rahul: the seconds per question must be configurable** (a setting in Go's `internal/config`, read from the environment with 15 as the default, and sent to the web app with `GET /api/config`), not hardcoded.
