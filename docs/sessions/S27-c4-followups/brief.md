# S27: Follow-ups after plan C4

**Plan:** none of its own. These are Rahul's 24 Oct decisions after plan C4 ([decisions.md](../../decisions.md) §2, 24 Oct), built before plan C5.
**Requirements:** PRE-01–PRE-04 as changed on 24 Oct; §6 point 7 of [saas-requirements.md](../../saas-requirements.md#6-pre-filling-from-earlier-hqs-agreed-need-proposed-design); the S21 and S22 items settled in [pending.md](../../pending.md).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S26 verified (plan C4 done).

**Ask Rahul first:**
1. **The patient's message for answers from another trust.** **Recommended:** S25's opening card names the source hospital when any of the set's answers come from another trust: "We've filled in 9 answers from your questionnaire at Hospital B1 on 3 Mar 2026. Please check them." It keeps **Check each answer** and **Everything is still correct**, and adds **Start fresh**.
2. **What Start fresh clears.** **Recommended:** the answers pre-filled in **that Question Set** (from any trust), so the set opens empty, and it's recorded with the button, as S25 records the others. It's offered only while nothing in the set has been changed, and it can't be undone.
   - *Or:* every pre-filled answer in the whole HQ, from the first card.
3. **Staff filling in on the patient's behalf.** There's no question for the patient now, so the reason for leaving other trusts out has gone. **Recommended:** on-behalf filling also copies other trusts' answers, and staff see the same card and markers.
   - *Or:* keep on-behalf filling to the same trust only.
4. **S26's consent table and route** (`patient_prefill_consents`, `POST …/prefill-consent`, migration 00026). Nothing is pushed yet. **Recommended:** remove them with migration 00027, along with the consent screen. Answers already copied stay.
5. **Clickable answers on the POA Summary.** **Recommended:** each answer (the patient's and the validated one) and each disclosure link opens Validate for that Question Set (`…/episodes/:eid/sets/:cid`), scrolled to that question and highlighted briefly. The Back button returns to the same place on the POA Summary.

## Build

- **Cross-trust pre-fill without asking** (`internal/prefill`, `httpapi/prefill.go`):
  - remove `crossTrustPrefill` from the gate and from the trust's settings (`internal/settings`, the Settings page, the superadmin console), with a migration that drops any stored value;
  - another trust's answers are always used, with the same matching (a platform library key with choices mapped by option text, else exact wording with every safeguard), one trust per query, by that trust's `org_id`;
  - a suspended trust's answers still aren't used;
  - keep the `prefill.read` audit row under the source trust, and no trust-B identifiers in trust A's data or responses (S26's tests stay);
  - answers 1–4 as agreed.
- **The archived episode** (S22 leftovers):
  - the patient's home shows a sent-then-archived questionnaire as **Sent** with a read-only **View** (today it's "Closed by the hospital" and 403);
  - an archived episode hides the **NOT RESPONDING** badge.
- **The default Cancelled message:** neutral wording, since Cancel is allowed from any status ("…has cancelled your pre-operative assessment…"). It's off by default and editable (`settings.go`, `episode.StatusMessage`).
- **The format names in the publish refusal:** the S23 lock names a text format in words ("phone number", "NHS number", "UK postcode", "email address") instead of its raw pattern. "NHS number" and "UK postcode" stay hardcoded (a recorded A-20 exception).
- **Clickable answers on the POA Summary** (answer 5), on both its tabs.
- **Integration tests:**
  - another trust's answers are copied with no setting and no consent;
  - Start fresh clears the set and is recorded;
  - on-behalf filling, as answered;
  - the archived View is read-only (a save is refused);
  - the five cases for any new or changed route.
- **End-to-end:** J16b changes to the no-question flow (a patient with a Trust B history opens a Trust A HQ, sees the card naming Hospital B1, and uses Start fresh on one set). J9 (archive) checks the read-only View.

## Not in this session

- The print-designed POA Summary PDF (plan C9).
- An account merge for one person's two accounts (later; C10).
- IG/DPO sign-off on copying across trusts (before go-live, C10).

## Manual test focus

1. A patient with only a Trust B history opens a Trust A HQ at phone size: answers are filled with no question first, and the card names Hospital B1. Start fresh empties that set. Nothing in Trust A's responses names Trust B's records.
2. The clinician's markers still say where each answer came from, and how it matched.
3. Trust settings and the superadmin console no longer show `crossTrustPrefill`.
4. Archive a sent episode: the patient sees Sent with a read-only View, and the badge is gone.
5. Cancel an episode in each status: the message reads correctly.
6. Change a published question's format to phone number: the refusal says "phone number".
7. On the POA Summary, click a patient answer, a validated answer and a disclosure link: each opens Validate at that question, and Back returns.
8. Export CSV and Print, checked by the tester as files (the CSV in `~/Downloads`, and Print rendered to PDF), as the new tester rule says.
