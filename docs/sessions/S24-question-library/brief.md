# S24: The question library and "Same as…"

**Plan:** C4 ([c4-question-keys-prefill.md](../../plans/c4-question-keys-prefill.md)), part 2 of 4.
**Requirements:** §6 points 2 and 3 of [saas-requirements.md](../../saas-requirements.md#6-pre-filling-from-earlier-hqs-agreed-need-proposed-design); PRE-05 ("Same as…"); PNL-06 changed so library inserts keep their keys.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S23 verified.

**Ask Rahul first:**
1. **The platform's starter library.** Made-up but sensible wording, seeded into every deployment (`org_id` NULL), with readable keys. **Recommended** (confirm or change the list):
   - `smoker` (never, ex-smoker, current), `alcohol_units` (number), `snoring` (yes/no), `ponv_history` (sickness after an anaesthetic, yes/no), `anaesthetic_problems` (yes/no + details), `family_anaesthetic_problems` (yes/no + details);
   - `height` (cm), `weight` (kg, Always ask fresh), `pregnant` (Always ask fresh);
   - `allergies` and `regular_medicines` as repeating panels.
2. **Who looks after the trust's own library?**
   - **Recommended:** whole-trust super clinicians and admins (the people who may change the trust's questionnaires) save a question from the editor with **Save to library**, and remove one from a library window.
   - Platform items are seeded only. A superadmin screen for them comes with C5's content library.
3. **What "Same as…" does to the new question.** A freshly made question has its own generated option values, which never match.
   - **Recommended:** adopting a key replaces the question's `name` **and its options** with the chosen question's (values and wording), and keeps its own title. The author can then reword, and add options, under S23's lock.
   - `linkedFrom` records where it came from (questionnaire and title), for information only.
4. **What counts as compatible** (`keys.Compatible`). **Recommended:** the same kind of answer:
   - single choice (radio, dropdown, yes/no buttons) with single choice; multiple choice with multiple choice;
   - a number with a number of the same unit; a date with a date; text with text;
   - **repeating panels aren't offered for "Same as…"**: they get their keys from the library or a new version.
5. **One `keys.Compatible`, in Go.** The search route returns each candidate with `compatible` and a reason, so the editor never needs a TypeScript copy. **Recommended: yes.**

## Build

- **`library_questions`:** id, `org_id` (NULL = platform), key, the SurveyJS element JSON, tags, created_by, created_at. A key is unique per owner, and a trust item can't reuse a platform key.
  - The platform items from answer 1 are seeded (idempotent), like the Lifebox codes.
  - Routes: `GET /api/o/{oid}/library-questions?q=` (platform + the trust's), `POST` and `DELETE …/{id}`, with the five integration cases.
- **Insert from library** in the editor's Add bar: a searchable list by title and tag. Inserting **keeps the key** (and option values). It's refused if the key is already in this HQ.
- **"Same as…"** on each question's settings:
  - searches the keys in the trust's published HQs (S23's walk) and the library, by title;
  - `POST /api/o/{oid}/question-keys/search` with the question's JSON returns the candidates with `compatible` and a reason;
  - adopting is offered only for compatible ones, and is refused if the key is already in this HQ.
- **`keys.Compatible`**, test-first, as answered.
- **PNL-06 in [requirements.md](../../requirements.md):** library inserts keep their keys (the change agreed in §6).
- **The seed's "Demo pre-op HQ"** (S19) uses library questions, so S25's pre-fill has shared keys to show.

## Not in this session

- Saved groups of questions as library items (the rest of PNL-06), and the content library for videos and leaflets (C5).
- The validated score library (C6/C7).
- Pre-fill (S25).

## Manual test focus

1. Insert Smoker from the library: it keeps the key `smoker`. Insert it again into the same HQ: refused.
2. As Alex Author, Save to library a trust question; it appears for Trust A's authors and never for Trust B's. Cara can't save or remove.
3. Make a new single-choice question, "Same as…" a published one in another HQ: it takes that key and its options, keeps its own title, and the label says "Pre-fills from 1 other HQ".
4. An incompatible candidate (e.g. text for a choice question) is shown greyed with its reason, and can't be adopted.
5. Delete an adopted option and Publish: S23's lock refuses it.
6. Remove a trust library item: questions already inserted from it are unchanged.
