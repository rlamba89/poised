# S14: Patient invites and magic-link sign-in

**Plan:** C2 ([c2-patients-signin.md](../../plans/c2-patients-signin.md)), part 2 of 4.
**Requirements:** PAT-03, PAT-04, PAT-05, PAT-09, A-4.
**Depends on:** S13 verified.

**Ask Rahul first:** nothing new. The SMS wording follows PAT-03's example, and the hospital's name comes from the hospital record.

## Build

- **`Deps.SMS` and `Deps.Email`,** with recording fakes.
- **`MESSAGING=capture`** (locally and in QA): messages go to `sent_messages`. Two dev/QA-only ways to read them:
  - `GET /api/qa/messages?to=`;
  - a small **captured-messages page**, for the tester.
  - CDK refuses capture mode on any other stage.
- **Creating an episode sends the invite** by SMS and/or email. Staff can resend it.
- **`login_links`:**
  - only the hash of a 32-byte token is stored;
  - an invite link works until the episode closes.
- **The patient flow, `/i/:token`:**
  1. A **Continue** button sends a POST.
  2. The patient enters their date of birth.
  3. A session cookie is set, with a 30-minute idle timeout.
  - **5 wrong dates of birth lock the link.**
  - **The link alone returns no data.**
- **The patient fills in and submits the HQ** through the new patient API. This replaces `/p/:token`.
- **Staff can complete the HQ on the patient's behalf** (PAT-09). The answers are marked "entered by <staff>".
- **E2E:** J7 and J7b, and J3 updated.

## Manual test focus

At **phone size**:
1. Staff create an episode, and the captured SMS and email both appear.
2. Open the link → Continue → a wrong date of birth 4 times → the right one → fill in and submit.
3. 5 wrong dates of birth → locked.
4. Resend the invite.
5. Open the link in a fresh browser profile or private window: no data shows before the date of birth.
6. Reload and use the back button in the middle of the flow.
7. **Staff complete on behalf;** the "entered by" mark shows.

The idle timeout is checked by an automated test only (the plan says so).
