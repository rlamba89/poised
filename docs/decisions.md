# Decisions, conventions and lessons

**What this file is:** the project's memory. It replaces the Claude chat history, which doesn't travel to other machines or developers.

- **What it holds:** every decision made in the Claude sessions so far, the reason for each, Rahul's working preferences, and environment gotchas.
- **Keeping it current:** add new decisions at the top of §2 in the same format.
- **Superseded decisions:** mark them as superseded, with the date and what replaced them; never delete them.

Related:
- [CLAUDE.md](../CLAUDE.md): the short hand-over for Claude;
- [plans/README.md](plans/README.md): the roadmap;
- [saas-requirements.md](saas-requirements.md) §0: the SaaS decision table, in more detail.

## 1. Working with Rahul (the product owner)

- **Simplest thing that works.** He pushes back on extra services, steps and abstractions. Explain the cost and the reason up front, especially for managed cloud services.
- **Ask, with a recommendation.** For stack and product choices, give 2–4 options with the recommended one first, then wait. Don't decide alone. But fix routine problems yourself (e.g. "you can fix the docker yourself and then start").
- **Check the official docs before planning** (SurveyJS, AWS, Amplify…). Several plans changed because the docs said otherwise: Amplify supports Next.js ≤ 15 only; Cognito has no native magic link; Aurora Serverless takes about 15 s to wake.
- **He reads Go, not TypeScript.** Keep backend logic in Go, and explain TypeScript he must review.
- **The incumbent is not the architecture reference** (2 Oct). The incumbent (the product he knows well) is evidence of what hospitals need. Never justify a tech choice with "the incumbent does it".
- **Functionality over content fidelity.** Gaps in copied incumbent questionnaire content aren't a priority; working features are.
- **Show the tests running.** He likes a visible-browser run (headed) when checking a flow.
- **Personal account only.** The code lives in his private GitHub (`rlamba89/poised`). Nothing goes to the incumbent's org. Neither the incumbent product's name nor its company's GitHub organisation name may appear anywhere in the repo: file names, code, docs or commit messages (the org name since 2 Oct, the product name since 22 Oct). Say "the incumbent".
- **Messages he'll send on** (e.g. IT tickets) shouldn't mention Claude.
- **Healthcare: nothing may break silently.** Hence the testing rules (§2, 2 Oct).
- **Ready, not built** (8 Oct). Don't build for a market or feature before a customer needs it. But don't hardcode anything that would make it a rewrite later: another country must be config, not code.
- **The local dev database is disposable** (8 Oct). Rahul is the only developer and nothing is in production, so a builder may drop and rebuild the local `sj` database to match the code, and needn't keep other branches working. Copy back anything that can't be recreated from the repo (today: HJE Full HQ, see §4).

## 2. Decision log (newest first)

Format: **decision**, then the reason, then where it's recorded in more detail.

### 25 Oct 2026: the open items after S28 answered (Rahul, asked by the conductor after S28 was verified)

All to be built in a new small session **S29**, before plan C5 (brief not yet written), except item 7.

1. **A later join can bring one more pre-fill.** If a join since the form's last pre-fill brings a trust that wasn't read yet, the next open reads that trust once, for sets nobody has changed yet (one `prefill.read` per newly joined trust). **Why:** with strict "once per form" (S28 answers 3), staff opening the form before the record joins left the patient's own later open with nothing from other trusts. This refines S28's answer 3; it doesn't reverse it. ([pending.md](pending.md), "From S28's verification 1")
2. **The POA Summary's "Patient reported BMI" uses only answers that are shown.** A hidden height, weight or BMI never feeds it. **Why:** a hidden answer must not feed a clinical number, as with disclosures and outputs.
3. **The opening card's three buttons each get their own audit action** (Check each answer, Everything is still correct, Start fresh), for both `behalf.prefill` and `link.prefill`. **Why:** the log alone should show that answers were cleared, and who cleared them.
4. **Network errors get one plain message on every page** (`apps/web/src/lib/api.ts`, e.g. "We couldn't reach the server. Check your connection and try again."), and the error banner is hidden while the "answers were cleared" card shows. **Why:** "Failed to fetch" is the browser's raw text.
5. **Archived episodes hide the open-task count** (list rows with Show archived, and the Tasks title); the tasks stay visible as a record. **Why:** the tasks can't be changed once archived, so the count reads as work outstanding.
6. **An episode archived from Cancelled shows the cancelled message on an old link's form too**, not "This questionnaire is closed", so the patient sees one reason wherever they arrive.
7. **On catch-up, a reminder and Not responding may still come on the same run: kept as designed** (only the latest reminder, once). **Why:** it still nudges the patient, and it's rare (it needs a suspension or missed runs). No change.
8. **"Same as…" warns before replacing a question's existing link** (a key from the library or a published HQ): the window says, for example, "This question is already linked to '…'. Using this replaces that link." before Use this. **Why:** otherwise pre-fill quietly changes what the question matches.

### 25 Oct 2026: S28, the follow-ups after S27 (builder, automatic run; tested 25 Oct, test report 1 all passed, 18 of 18; verified 25 Oct, [verification 1](sessions/S28-s27-followups/verify.md))

- **Pre-fill runs once per form** (`episode_forms.prefilled_at`, **migration 00028**): the first open locks the form row, fills, and marks it; later opens (and a second open at the same moment, which waits) never read earlier HQs again. Forms that already had patient answers were marked by the migration. ([build notes](sessions/S28-s27-followups/build-notes.md))
- **The staff-side four-field join** (`joinOnBehalf`): on Fill in for the patient, only while the record has no account and the form is neither sent nor pre-filled yet; `FindAccountForRecord`, then `JoinAccount` of this record only; an `account.join` row by the staff member in the record's trust's log (no hospital, as the patient's own join).
- **Another trust's audit row stores no user id at all:** a staff request's row under another trust holds `audit_log.actor_org_id` (the staff member's trust) instead of `user_id`. Its log, CSV and Who filter (`?user=trust:<id>`) say "Staff at <trust>", with no email. Rows from before S28 stay as written (append-only).
- **Patient reported BMI** reads a BMI question or the platform library's `height`/`weight` questions (the demo HQs use those), from the clinician's answers once the set is validated, else the patient's; the same on both POA Summary tabs.
- **The POA Summary keeps a set started fresh** even with nothing else to show, for its line (`startedFreshText`, shared with Validate).

### 25 Oct 2026: S28's questions answered (Rahul, in chat)

- **The staff-side four-field join happens when staff open Fill in for the patient**, just before the form is pre-filled, so only when staff actually fill in.
- **It joins only that record, and only to an account that already exists:** no new account, no sweep of the person's other records, and no contacts copied either way, so Trust A staff never see a mobile or email only Trust B holds. One `account.join` row in Trust A's log names the staff member.
- **The whole pre-fill runs once per form**, on its first open, from this trust and other trusts alike; the form records it, and later opens never read earlier HQs again. *Narrows S25's "while a set has no answers" re-reads.* ([answers.md](sessions/S28-s27-followups/answers.md))

### 24 Oct 2026: S27's open items answered (Rahul, asked by the conductor after S27 was verified)

All to be built in a new small session **S28**, before plan C5 ([brief](sessions/S28-s27-followups/brief.md)).

- **On the patient's behalf, a record joins an account on the four fields.** When staff fill in on the patient's behalf, a record whose **date of birth, mobile or email, first name and surname** all match an account joins it (S15's join rule), with an `account.join` audit row naming the staff member; other trusts' answers then fill in. **Why (Rahul):** patients are filled in for because of low literacy, so they won't open a link; a mistype finds no account (fails safe); staff are trusted and audited, as with guessing a date of birth (24 Oct). **Risk accepted:** someone who knows a person's details could read their other trusts' history; on the IG/DPO review list (C10). *Supersedes S27's "On the patient's behalf, other trusts' answers need this trust's record joined to the account" below.*
- **Another trust's audit log names only the trust, not the staff member.** A cross-trust `prefill.read` caused by Trust A staff shows in Trust B's log as staff at Trust A, with no name or email; Trust A's own log still names the person. Changes S27's on-behalf actor.
- **Other trusts are read only on a form's first open** (one `prefill.read` per form per trust), not on every open (S27 verification 1 note). A Trust B HQ sent after that first open doesn't fill the form.
- **The POA Summary says when a Question Set was started fresh**, with Validate's wording.
- **An archived episode's read-only View drops "A member of the team will review them."**
- **A cancelled questionnaire the patient had already sent** shows on their home as "The hospital has cancelled this. Please contact them if you have questions." (not "…nothing to fill in").
- **NOT RESPONDING is hidden once an episode is cancelled**, as well as once archived.
- **The POA Summary's "Patient reported BMI" falls back to the patient's own answers** until About you is validated.
- **S28 also takes S27's small fixes:** a `scroll-margin-top` so the sticky staff top bar doesn't hide a jumped-to question; `choose` reporting a failed reload after Start fresh; a clean-up for `restoreSpot` (`jump.ts`). ([pending.md](pending.md))

### 24 Oct 2026: S27, the follow-ups after plan C4 (builder, automatic run; test report 1: 19 of 21, F2 fixed and F1's plan step corrected; tested 24 Oct, test report 2 all passed; verification 1 needed fixes (a stale card after Start fresh, the card's wording); tested again 24 Oct, test report 3 all passed; verification 2 needed one fix (an e2e journey that assumed Cara never appears in Trust B's log); tested again 24 Oct, test report 4 all passed; **verified 24 Oct, verification 3**, a round Rahul added)

- **Another trust's answers are copied at the first open, for the patient and for staff on their behalf** (answers 3), from every active trust where the person's account has a sent HQ (`ListPrefillTrusts`), each read by its own `org_id` and audited under it (`prefill.read`, now without "(they agreed)"; on behalf its actor is the staff member). **Migration 00027** drops `patient_prefill_consents` and every stored `crossTrustPrefill`; the setting, the consent screen and `POST …/prefill-consent` are gone. Answers already copied stay. ([build notes](sessions/S27-c4-followups/build-notes.md))
- **Start fresh** (answers 1–2) is the card's third button **only when the set holds another trust's answer**: Go empties the set's patient row (`prefill_choice` `start_fresh`, the revision moves on so a stale tab gets the 409), keeps the `prefill` map as the record, and refuses it once the set changed (revision past 1) or another button was pressed. *Builder's defaults:* it asks first, inline ("This can't be undone"); after it nothing in the set counts as pre-filled (`prefilledAnswers`), and the clinician sees one line saying the patient started fresh.
- **A sent, then archived questionnaire stays Sent with its View** (the same thank-you page as a sent one); only the patient's GET is allowed, every write is still refused. Archived unsent stays "Closed by the hospital". **NOT RESPONDING is hidden once archived** (`notResponding`, `status.ts`).
- **After Start fresh, a card still open elsewhere can't go on:** its other buttons get a 409 ("…cleared with “Start fresh” somewhere else… Please reload the page."), so an emptied set is never sent as checked (verification 1, finding 1). **Start fresh is offered from the answers the card shows**, and a card whose answers all came from one other hospital names it without "including" (findings 2–3).
- **The Cancelled defaults:** "…has cancelled your pre-operative assessment. Please contact the hospital if you have any questions."; the email's fixed subject too: "<hospital>: your pre-operative assessment was cancelled" (test report 1, F2).
- ~~**On the patient's behalf, other trusts' answers need this trust's record joined to the account** (the patient's date of birth on one of its links, S15): Go never matches records across trusts by typed details alone (test report 1, F1: the plan step was corrected, not the code; for Rahul in pending.md).~~ *Superseded 24 Oct by the four-field join on the patient's behalf (above, S28).*
- **The publish refusal names text formats in words** (`keys.textFormatNames`: phone number, NHS number, UK postcode); a test keeps the patterns equal to `fields.ts`'s TEXT_FORMATS.
- **Clickable POA Summary answers** (answers 5): disclosures, capture titles, pre-filled lines and the clinician's changes link to `…/sets/:cid?q=<question>`; Validate opens the question's page, centres it and marks it `data-jumped` for 3 s, with **← POA Summary** (`navigate(-1)`). The tab is in the POA Summary's URL (`?tab=patient`). **Gotcha:** React Router's `ScrollRestoration` runs before the summary's data loads, so Back landed at the top; the link clicked and its place on the screen go in session storage and are put back once drawn (`jump.ts`). A highlight on a SurveyJS question must be a data attribute: React rewrites its `className`.
- **Gotcha: run the whole e2e suite on the shared dev database, not only a fresh one** (verification 2). `test_audit.py`'s "another trust sees nothing" assumed Cara never appears in Trust B's log, which S27's on-behalf `prefill.read` rows rightly break. It passed on the builder's fresh `sj_s27` and failed on the conductor's `sj`. It now checks only its own episode and record. The pipeline's e2e runs on QA's persistent database too.

### 24 Oct 2026: after plan C4 — pre-fill across trusts without asking, and small follow-ups (Rahul, in chat with the conductor)

- **Answers from another trust are always copied, without asking anyone first.** The trusts' `crossTrustPrefill` setting is removed (from the gate and from the trust's settings), and the patient isn't asked "Use the answers you gave …?". Instead, a pre-filled section from another trust shows an **informative message** ("We've copied answers from your questionnaire at Hospital B1…") with a way to **start fresh** (clear the copied answers). Matching is unchanged: a platform library key with choices mapped by option text, else exact wording with every safeguard. **Why:** it's the patient's own data, and to the patient it's one episode whichever trust holds it; asking trusts to opt in is a poor sales story. *Supersedes the gate in 22 Oct (S26 answers Q2–Q3) and 23 Oct (S26) below.* To be built in a small session before C5 (not yet written); it decides what happens to S26's `patient_prefill_consents` table and route. **Open (C10):** confirm with an information-governance lead / DPO before go-live, since each trust is a separate data controller under UK GDPR.
- **Settled with it:** a Yes covering every HQ of a trust, and copied answers becoming the receiving trust's own once sent, are both fine ("it doesn't matter to the patient"). ([pending.md](pending.md))
- **An archived episode keeps a read-only View of the patient's sent questionnaire**, and **hides the NOT RESPONDING badge**. (pending.md, S22)
- **The default "Cancelled" message gets neutral wording** (it's sent from any status). (pending.md, S21)
- **The editor's "NHS number" and "UK postcode" text formats stay hardcoded for now: a recorded exception to A-20** (`packages/clinical/src/fields.ts` TEXT_FORMATS). Move them to the per-country config when a second country comes.
- **The publish lock's refusal names a text format in words** ("phone number", …), not its raw pattern.
- **Two accounts for one person:** no merge tool yet; revisit with C10. **Guessing a date of birth:** left as is (staff are trusted, and every try is audited).
- **POA Summary, new requirements:** (a) **a print-designed PDF** made for paper, not a print of the web page (today's Print shows tabs and web elements) — in plan C9; (b) **the patient's and the validated answers, and the disclosure links, are clickable** and open the clinician at that question in the HQ — in the small session before C5.
- **Testers test downloads and printing themselves** (Rahul): a downloaded file lands in `~/Downloads` and is read and checked; a print layout is rendered to PDF and inspected. Neither is a hand check any more. Suspending and re-activating a trust locally is allowed in a test. (`.claude/skills/manual-test/SKILL.md`)

### 23 Oct 2026: the tender market research, and the backlog lives in Jira (Rahul)

- **The tender market research is in [docs/market/tender-research.md](market/tender-research.md)** (with [the appendix](market/tender-research-appendix.md)): 310 UK, Irish and European tenders, 625 features buyers ask for ranked by demand, 178 competitors, pricing, the first-sale gate (§10.6) and why stand-alone pre-op tenders are mostly a UK thing (§11). Live web only; every claim links its source.
- **The product backlog lives in Jira project PB "Preopsis Backlog"** (`preopsis.atlassian.net`, Jira Free: no issue cap). 18 epics (the 13 suite modules plus the compliance gate, company and bids, delivery, core product, hosting), 270 feature stories (the 625 features grouped, every id kept as a label), 539 sub-tasks split by who does the work (components Build, Integrate, Clinical, Assurance, Commercial, Service, Partner). Priority comes from demand (P1 High, P2 Medium, P3 Low, gate Highest). **Rahul records decisions in each story's Decision field**; Proposal is Claude's suggestion. **Why:** Rahul finds markdown in VS Code hard to read; Linear Free caps at 250 issues; Jira Free has no cap. **No sync back to the repo** (Rahul, 23 Oct): Jira is the record for backlog decisions. When a session builds a feature, mark its PB story Done. Loader and id→key map: the session's scratchpad (`jira_load.py`, `jira-map.json`); the API token is `JIRA_API_KEY` in `~/.zshrc`, user `lamba.rahul89@gmail.com`.
- **Rahul's preference:** everything a research or analysis finds goes into the shared document, never only into chat.
- **Gotcha: the GitHub repo is public.** `gh repo view` reports `rlamba89/poised` as PUBLIC, although this file and CLAUDE.md say private. Don't push `docs/market/` (competitive research) until Rahul decides the repo's visibility.

### 23 Oct 2026: S26, the clinician's pre-fill markers and answers from other trusts (builder, automatic run; test report 1: 20 of 20, two exploring notes fixed (audit label "read to pre-fill", a bad orgId's message); tested 23 Oct, test report 2 all passed; verification 1 needed fixes (cross-trust key matches, trust B's form id) and Rahul chose the library-key rule; tested again 23 Oct, test report 3 all passed; **verified 23 Oct, verification 2**; plan C4 Done)

- *Gate superseded 24 Oct (entry above): no trust setting, no question; always copied with a message.* ~~**Another trust's answers are used only with both trusts' `crossTrustPrefill` on and the patient's Yes**, asked once per account and source trust before the first Question Set ("Use the answers you gave Hospital B1 in October 2026?"; `patient_prefill_consents`, migration 00026; `POST /api/p/episodes/{eid}/prefill-consent`, audited). Never on the patient's behalf. A suspended trust's answers aren't offered. ([answers.md](sessions/S26-prefill-clinician-cross-trust/answers.md), [build notes](sessions/S26-prefill-clinician-cross-trust/build-notes.md))~~
- **They're read one trust per query** (`ListPrefillSources` with that trust's `org_id`), only after both checks. Deciding whether to ask reads no answers, only that trust's settings and its newest sent HQ's hospital and date (`ListPrefillOffers`). **Each read writes a `prefill.read` audit row under the source trust**, at the source episode's hospital, naming its episode and record. The form's own trust's row never names them. *Builder's choice:* the episode id stands for "the source form ID" (answers Q5); `audit_log` has no form column. Its label says the answers were **read** to pre-fill, not "used" (test report 1, found 4): the form's own trust may have covered every question.
- **Another trust's answers match by key only for a platform library key** (`library_questions` with no trust, the same in every trust, children of a library group included), **otherwise by exact wording with every wording safeguard; either way every choice is mapped by its option's text, and a text that doesn't match means no fill** (Rahul, 23 Oct, after S26's verification 1; [answers.md](sessions/S26-prefill-clinician-cross-trust/answers.md) "After verification 1"). **Why:** S23's lock freezes keyed options within one trust only, so a trust's own `q_smokes` may mean something else at another trust. The clinician's marker says "matched by link" for a library key and "matched by same wording" otherwise. ~~Another trust's answers match by wording only, never by key~~ *(the builder's first fix, superseded the same day by this rule).* **Trust A keeps no trust B id:** another trust's pre-fill entry has no `fromForm`.
- **The form's own trust's earlier forms decide first:** another trust's are tried only for questions none of them asked (`prefill.Source.OtherTrust`). *Builder's reading of "Yes fills only what's still empty, after same-trust pre-fill"*, "if in doubt, don't fill".
- **While a consent question is open, nothing is pre-filled for the patient** (the question comes before the sets), and the next open fills from every source at once, only into sets with no answers. *Builder's default:* no merge into existing rows, so a set staff or the patient already started gets nothing from another trust.
- **The pre-fill map stores the questionnaire's name, and for another trust's answer `otherTrust` and the hospital**, so the clinician's marker never reads another trust's records.
- **The clinician's markers (PRE-04):** Validate and the POA Summary mark each pre-filled answer **shown**: "Pre-filled from Demo pre-op HQ, 3 Mar 2026 (matched by link / by same wording)", "at Hospital B1 (another trust)", "Changed by patient: pre-filled “Yes”". A line per set gives the count and the card's button ("The patient chose “Everything is still correct” on …, so these pre-filled answers weren't checked one by one", or "Cara Clinician chose … with the patient"). The rules are in `packages/clinical/src/prefill.ts`.
- **A pre-filled row starts at revision 1**, so a stale tab's "no row yet" (0) gets the 409 (S25 verify). **"Started" is one generated column**, `episode_answers.started`. **A repeating-list row that would lose all its answers refuses the whole answer** (S25 verify).

### 23 Oct 2026: S25, pre-fill and the patient's confirmation screens (builder, automatic run; test report 1: 18 of 19, T-03 and two findings fixed; tested 23 Oct, test report 2 all passed; **verified 23 Oct, verification 1**, minor leftovers in pending.md)

- **`internal/prefill` (Go) fills a new HQ form's patient Question Sets from the patient's earlier submitted HQs in the same trust**, by key, else by exact wording (S23's `keys.SameWording`), when the form is first opened, by the patient or by staff for them. It's stored as the patient's answers (revision 0) with `episode_answers.prefill` `{key: {fromForm, answeredAt, value, comment?, matchedBy}}` (migration 00025). ([build notes](sessions/S25-prefill/build-notes.md))
- **The newest earlier form that asked a question decides; if its answer is empty or doesn't fit, nothing is filled** (no older answer). The clinician's validated copy is used whole per Question Set. A choice answer must map to the new question's options (by value for a key, by option text for wording), all or nothing; rows are copied keeping only the new columns. *Builder's defaults, "if in doubt, don't fill".*
- **The opening card's button is stored beside the map** (`prefill_choice`, `_at`, `_by`; the first press kept), via `POST …/prefill/{cid}` (patient and on-behalf, audited). **A pre-filled set doesn't count as complete until its card is answered**, so Send waits for it. *Builder's default (safety).*
- **"Started" is now a patient row with revision > 0, or a pre-filled set's card answered** (after test report 1, found 1: confirming is acting on it). An untouched pre-filled HQ still gets reminders and shows To do on the home.
- **After "Everything is still correct" the page scrolls the question itself to the top and keeps it there while SurveyJS settles its phone layout** (`focusQuestion`): SurveyJS's `question.focus()` left the question's text off a phone's screen (test report 1, F1). *Gotcha for any future "jump to a question".*
- **A set with nothing to fill gets no row**, so a later open may still fill it while the patient hasn't saved anything there. *Builder's default.*
- **Seed: a "Demo follow-up HQ" for Trust A** shares the Demo pre-op HQ's questions (one by wording only), for seeing pre-fill. The local `sj` was migrated and seeded (not reset).
- **Hidden answers:** SurveyJS's default `clearInvisibleValues` (`onComplete`) is kept. A pre-filled answer its condition hides is kept until the set is finished with it hidden, and isn't counted on the card.

### 22 Oct 2026: S24, the question library and "Same as…" (builder, automatic run; test report 1 all passed, two exploring items fixed; tested 22 Oct, test report 2 all passed; verification 1 needed fixes, 23 Oct; tested again 23 Oct, test report 3 all passed; verification 2 needed one fix, 23 Oct; tested again 23 Oct, test report 4 all passed; **verified 23 Oct, verification 3**, minor leftovers in pending.md)

- **A medication question's type (prescribed, non-prescribed, recreational) is part of its frozen answer** and of the wording match (`keys.formatOf`; no type = prescribed). **Why:** S23's verification 3; pre-fill must never copy prescribed medicines into a recreational-drugs list. ([build notes](sessions/S24-question-library/build-notes.md))
- **The question library is `library_questions`** (migration 00024): org_id NULL = the platform's starter set, seeded from `cmd/seed/library.json` with the codes (every deployment, not only the demo data); each trust's own, saved and removed by whole-trust super clinicians and admins (`/api/o/{oid}/library-questions`; listing too: nobody else authors the trust's questionnaires). A library question keeps its key and option values; its conditions and `linkedFrom` are dropped on save; a question that differs from the trust's published one with that key can't be saved (409). Not audited (no patient data).
- **"yes/no + details" in the starter library is a group** of a Yes/No question and a details box shown on Yes (`anaesthetic_problems_group`, `family_anaesthetic_problems_group`), both keeping their keys. *Builder's default:* a library item is one element, and the editor already builds this pattern.
- **`keys.Compatible` and `keys.Candidates` (Go only) decide "Same as…"**: the same kind of answer; choices whatever their options (adopting copies them); other kinds need the same unit, format, medication type, scale, rows and labels already (*builder's default*: only the key is copied for them, so a difference would be refused at Publish). Repeating lists are never offered. The route is `POST /api/o/{oid}/h/{hid}/questionnaires/{qid}/question-keys/search` (the brief said `/api/o/{oid}/question-keys/search`), as S23's GET, to know the HQ for "already in this HQ".
- **Adopting a key** (`adoptKey` in `packages/clinical/src/library.ts`) copies the chosen question's options without their disclosures, keeps the question's own text, renames conditions that tested it, and records `linkedFrom` (information only).
- **The seeded Demo pre-op HQ is built from library questions** (S25 needs shared keys). The seed never changes a published demo HQ, so **the local `sj` database was reset on 22 Oct (S24)** with HJE Full HQ put back; QA keeps the old Demo HQ until it's reset (pending.md).
- **PNL-06 changed in requirements.md:** library inserts keep their IDs.
- **After S24's verification 1 (Rahul, 23 Oct): "Same as…" copies Always ask fresh** when the chosen question (library or published) has it on; the author may switch it off afterwards. **Why:** a linked copy of `pregnant` or `weight` would otherwise pre-fill an old answer ("if in doubt, don't fill"). ([answers.md](sessions/S24-question-library/answers.md))
- **The logic-problem check (and so Publish) refuses a condition that compares a choice question with an answer it doesn't have** (`unknownAnswers`, `packages/clinical/src/logic.ts`). "Same as…" maps old options to new ones with the same text first, and warns before Use this what it would lose. *Builder's fix for verification 1, finding 1, which found the earlier note "the logic-problem check shows it" untrue.* ~~Adopting leaves conditions on old option values for the logic-problem check to flag.~~ *Superseded 23 Oct by this entry.*
- **After S24's verification 2 (23 Oct): a Question Set condition that tests an answer an earlier question doesn't have is a problem** (`chapterConditionProblems`), shown in the header and refused at Publish, so a whole set can't be hidden from patients unnoticed; Same as… counts other sets' conditions in its warning and renames test cases' answers. *Builder's fix, verification 2 finding 1.*
- **A repeating list's row question never goes in the library on its own** (test report 1, found 1): the editor offers Save to library on the repeating group only, and Go refuses a key that's a published row question or already inside a library item. **Why:** the same key alone would mean a single answer, not the list's rows. *Builder's fix, in scope.*

### 22 Oct 2026: the incumbent is never named

- **The incumbent product's name is removed from the whole repo** (Rahul, 22 Oct): code, comments, docs, old session notes, tools and file names. Write "the incumbent" instead. **Reason:** Rahul asked that not one mention remain.
  - **Renamed:** `internal/<name>` → `internal/hqimport`, `cmd/import-<name>` → `cmd/import-hq`, `db/seed/<name>/` → `db/seed/codes/`, `tools/export-<name>-hq.js` → `tools/export-hq.js` (it now downloads `hq-export-<id>.json`), and the UI notes → `docs/incumbent-ui-notes.md`.
  - **Guard:** `TestIncumbentNameAppearsNowhere` (`apps/api/internal/repocheck`, in `make test`) walks every file and path in the repo, in any letter case, and fails on the name. The name is base64-encoded in the test, so the test doesn't contain it. It replaces the old org-name grep in `/update-wiki` and `/verify-session`.
  - **Git history** was rewritten with `git filter-repo` on 22 Oct, as on 2 Oct, so no commit, file or commit message has the name. Anyone with an older clone must clone again.

### 22 Oct 2026: S23, question keys in the editor (builder, automatic run; test report 1 all passed; verification 1 needed fixes; test report 3: R-05 failed on old local data, fixed; test report 4 all passed; verification 2 needed fixes (date and text formats); tested 22 Oct, test report 5 all passed; **verified 22 Oct**, verification 3)

- **`internal/keys`** (Go) reads a questionnaire's keys from its JSON: `Walk` (every element but groups, at any depth, with its kind of answer, text, options, columns, clinician-only and `alwaysFresh`), `Duplicates`, `LockProblems`, and the PRE-05 matching (`Normalize`, `SameWording`, `RepeatedTexts`, `Matches`) that S25's pre-fill can reuse. ([build notes](sessions/S23-question-keys/build-notes.md))
- *Superseded after verification 1 (the whole answer is frozen, below): the lock compared options only.* ~~**Publish refuses key problems with 422** `{error, problems: [{set, message}]}`: a key used twice in the HQ (naming both sets), or a keyed question whose options differ (added, removed or reworded) from the newest published or retired version of **each** of the trust's questionnaires holding the key. Publishing runs under the trust's advisory lock, so two publishes can't race past the check. Option order, translations and the question's own text may change. *Builder's default for "which published version" and the order.*~~
- **The editor's route is `GET /api/o/{oid}/h/{hid}/questionnaires/{qid}/question-keys`** (the brief proposed `/api/o/{oid}/question-keys?questionnaire=`): it reuses the questionnaire's hospital scoping, and lets a hospital-only super clinician, who views the editor, see the label. Super clinician and above; not audited. It's fetched when the editor opens **and after every save**, so new or reworded questions get the right label. *Builder's default.*
- **The editor's placeholder texts ("Text question", "Select One question"…) never match by wording**: nobody chose them, and an untouched new question would otherwise "pre-fill" from every HQ with one. *Builder's default, found by the e2e run.*
- *Superseded after verification 1: grids, ratings and Yes/No switches are locked too (below).* ~~**The lock covers every question with options**, clinician-only ones and repeating-group children included; grids, ratings and Yes/No switches have no options here.~~ **"Always ask fresh" is offered on a repeating group** but not inside one, nor on clinician-only content or in a clinician set. *Builder's default.*
- **After verification 1 (Rahul, 22 Oct): the lock freezes every kind of answer**, not only options: the kind of answer, unit, options (values and wording), a Yes/No switch's labels, a grid's rows and columns, a rating's scale, a repeating list's columns. Changing any needs a new question with a new key. ([answers.md](sessions/S23-question-keys/answers.md), "After verification 1")
- **Signatures, file uploads and the BMI block never pre-fill** (no label, no "Always ask fresh"); wording matches also compare grid rows/columns, rating scales and list columns. *Builder's default for the BMI block (its weight changes, and it can't be marked "Always ask fresh").*
- **The key checks live only in Go:** the Publish window shows Go's problems from `GET …/question-keys` (`problems`), fetched as it opens and after a refusal, instead of a TypeScript copy of the checks. *Builder's choice after verification 1.*
- **The route lists the names of all the trust's questionnaires to any super clinician in the trust**, a hospital-scoped one included (other hospitals' own HQs too): no patient data, and pre-fill is trust-wide. *Builder's default, accepted by the verifier.*
- **After test report 3 (Rahul, 22 Oct): keep the strict rule.** A draft's keyed question is compared with **every** published question holding that key in the trust; conflicting old data is cleaned up, not accommodated. When the published copies already disagree, Publish gives **one** problem per question saying so. The local `sj` database's three "S02 test HQ" questionnaires (8–9 Oct, before the lock) were removed for this; `cmd/seed/keys_test.go` keeps the seeded Demo HQ and the e2e questionnaires agreeing on shared keys. ([answers.md](sessions/S23-question-keys/answers.md), "After test report 3")
- **The editor's default names ("Group", "Section", "Text question"…) never match by wording.** *Builder's default.*
- **After verification 2 (Rahul, 22 Oct): a date's format and a text question's input format are part of the frozen answer and of the wording match.** A date's format is day, month and year / month and year / year only; a text question's input format is any text, email address or its pattern (phone, the national ID, postcode, custom). One line or several lines of text (Text vs a long-answer box) is the same answer. **A number's range and decimal places are limits, not the kind of answer, and stay unlocked.** A repeating group reports only a question added or removed; its questions are checked as questions of their own (their text may change). ([answers.md](sessions/S23-question-keys/answers.md), "After verification 2")
- **The local `sj` database was reset to migrated + seeded on 22 Oct (S23)**, with HJE Full HQ (a draft) put back as it was; the e2e journeys now accept any published version of the seeded Demo pre-op HQ.
- **Verification 3 (22 Oct): Verified, with one gap carried into S24:** a medication question's type (prescribed, non-prescribed, recreational) isn't yet part of the frozen answer. It's S24's first build item, so it's closed before S25 copies answers by key. *Verifier's call: no answer is copied before S25, and the run had no further S23 round.* ([verify.md](sessions/S23-question-keys/verify.md))
- **Gotcha:** the integration fixtures `basic.json` and `workflow.json` give `q_smokes` two meanings (a switch vs Yes/No options). A test that publishes one where the other is published in the same trust is now refused, rightly; pick the fixture that matches the trust's.

### 22 Oct 2026: plan C4's questions answered (Rahul, asked by the `/run-sessions S23 S26` conductor)

- **Pre-fill matches by key, then by exact wording** (Rahul, changing the plan's key-only rule). **Why:** a patient should never re-answer something they already answered. Key-only matching would miss questions worded the same but written separately, and almost every question from another trust (they share keys only through the platform library). The rule:
  1. the same key is always a match (library inserts, "Same as…", new versions);
  2. otherwise the same question text (ignoring case, spaces and punctuation), the same kind of answer and, for choices, every option **text** the same; a choice answer is mapped by option text to the new question's own value;
  3. safeguards: a text used more than once in either questionnaire (e.g. "Please give details") never matches by wording; a repeating list matches only if its text and column texts match; **if in doubt, don't fill**. Authors can stop any question pre-filling with "Always ask fresh".
  
  ([S23 answers](sessions/S23-question-keys/answers.md), [S25 answers](sessions/S25-prefill/answers.md))
- **A published keyed question's options are frozen completely: values and wording, no adding, no removing** (Rahul chose this over the recommended "add and reword, never remove"). Go's publish compares each option's value and text with the published one, across every HQ in the trust holding that key. ([S23 answers](sessions/S23-question-keys/answers.md), Q2)
- **The publish lock freezes a keyed question's whole answer shape, for every kind of question** (Rahul, after S23's verification 1): the kind of answer, the unit, a grid's rows and columns, a rating's scale, a repeating panel's columns and the options (values and wording). **Why:** an old answer must always fit the question it pre-fills. ([S23 answers](sessions/S23-question-keys/answers.md), "After verification 1")
- **The editor's label (PRE-05) counts both kinds of match and says how** ("by link" or "by same wording"); `prefill` stores how each answer matched (`{key: {fromForm, answeredAt, value, matchedBy}}`), and the clinician sees it (PRE-04). ([S23](sessions/S23-question-keys/answers.md) Q5, [S25](sessions/S25-prefill/answers.md) Q4, [S26](sessions/S26-prefill-clinician-cross-trust/answers.md) Q1)
- **Everything else as the briefs recommended:** keys found by walking the trust's chapter JSON (no table); keys unique across an HQ (editor and publish); "Always ask fresh" on patient questions; the 11-question starter library; trust library kept by whole-trust super clinicians and admins; "Same as…" copies the key and options and keeps the title; `keys.Compatible` by kind of answer, Go only; pre-fill on a Question Set's first open from every submitted same-trust HQ, validated copy first, newest first; the opening card and markers, recording the button pressed; cross-trust only with both trusts' setting on and the patient's Yes, asked once per trust (*superseded 24 Oct: always copied, with a message*); never another trust's answers on the patient's behalf; an audit row under the source trust too. (Each session's `answers.md`)

### 21 Oct 2026: plan C4 split into sessions S23–S26 (Claude, for Rahul to confirm)

- **Four sessions, in this order:**
  1. S23, question keys in the editor;
  2. S24, the question library and "Same as…";
  3. S25, pre-fill from the same trust, with the patient's screens;
  4. S26, the clinician's markers and pre-fill from another trust.

  **Why:** keys must be trustworthy (unique in an HQ, option values locked) before anything copies answers by them. Pre-fill needs shared keys to show, which the library brings. Other trusts come last, because they're the only part that reads another trust's patient data. ([sessions/README.md](sessions/README.md))
- **C4's journeys are J16 and J16b**, because J9 is C3's lifecycle and C5–C9's plans already name J10–J15.
- **`store.ForOrg` isn't used:** it was dropped with A-19 on 8 Oct. S26's cross-trust read queries one trust at a time by `org_id`, only after both trusts' settings and the patient's agreement pass.
- **`crossTrustPrefill` already exists** (S10, trust-only, default off), so S26 adds only the patient's agreement.
- *Answered 22 Oct (entry above): the matching rule and the option lock differ from what this line proposed.* ~~**Each brief's "Ask Rahul first" items are open.** They're asked when the session starts, or up front by `/run-sessions`. The ones that shape the design: the option lock allows adding but not removing values (S23); "Same as…" copies the chosen question's options (S24); pre-fill runs once per Question Set on first load and stores the copied value (S25); the patient agrees once per other trust, and staff filling in on the patient's behalf get same-trust answers only (S26).~~

### 21 Oct 2026: S22, archive, auto-archive and closing a patient account (builder, automatic run; test report 1: 21 of 23, both failures fixed; test report 2: the plan's T-18 corrected, a Settings decimal fixed; tested 21 Oct, test report 3 all passed; verified 21 Oct, [verify.md](sessions/S22-archive/verify.md); plan C3 done)

- **Archive and un-archive are a super clinician's** (answers 1), from any status; un-archive needs a reason and goes back to **exactly** `archived_from`. Routes `POST …/archive` {reason?, revision} and `…/unarchive` {reason, revision}, audited, through S20's transition table and status history. ([build notes](sessions/S22-archive/build-notes.md))
- **An archived episode is read-only** (answers 2): every other write is 409, **notes included** (a cancelled episode still takes notes). It's in no worklist (`in_worklist`), and left out of the episode lists unless `?archived=true` ("Show archived") or `?status=archived`; search and the patient's page find it. The patient's form is closed (403 `code: "archived"`); their home shows "Closed by the hospital".
- **Archiving keeps the patient's invite link but stops the reminders; un-archiving doesn't restart them.** *Builder's default:* archive stays exactly undoable; reminders long overdue would arrive at once.
- **`episodes.last_activity_at` is kept by database triggers** on the General notes, status history, tasks, messages and answers (answers 3), not in each handler, so a later write path can't forget it. *Builder's default.*
- **`episode.ArchiveDue`** (pure, unit-tested): discharged → due on discharge date + N at the hospital (N from the later of discharge and the last un-archive, so it isn't re-archived the next night); before admission **or cancelled** → M months after the last activity; admitted never. The rule is recorded as "Auto-archive: 30 days after discharge" / "Auto-archive: 6 months without activity". *Builder's default for cancelled and the un-archive clock.*
- **The auto-archive job** (answers 4): kind `auto_archive`, one per hospital, at 02:00 in its zone, rescheduling itself. **Every `RunDue` first adds the job for any hospital without one**, so new hospitals and fresh databases need nothing else. Nothing is archived while the trust or hospital is suspended. **The testers' "run the jobs as of" also runs every hospital's auto-archive** as of that time (capture mode only), so test cases don't depend on date order. *Builder's default.*
- **A job run is one moment** (test report 1, F-2): what it writes (notes, the auto-archive's status event) is dated with the run's time, `last_activity_at` follows a row's own time, and the hospitals' auto-archives run before the follow-ups in the same run, so a reminder due in that run can't make an idle episode look active. On a real clock the run's time is now; on /qa/jobs it's the "as of" time. Messages sent still count as activity (answers 3). *Builder's fix, as the conductor directed.*
- **Number settings aren't clamped in the browser** (test report 1, F-1): a value out of range stays as typed and Save shows the API's refusal, rather than quietly saving the nearest allowed value.
- **Closing an account (EPI-09, answers 5):** `POST /api/p/account/close` unlinks the trust's records (they and their episodes stay, each episode noted "The patient closed their account on YYYY-MM-DD."), revokes their invite links and finishes their reminders, revokes unused sign-in links to the account's contacts **unless another account shares the contact**, deletes the sessions, verify links and the account, and writes an audit row per trust. A later invite makes a new account. *Builder's default for the invite links, reminders and shared contacts.*

### 21 Oct 2026: S21, tasks, worklists and status messages (builder, automatic run; tested 21 Oct, test report 1 all passed; verified 21 Oct, [verify.md](sessions/S21-tasks-worklists/verify.md))

- **`tasks`** (TSK-01): episode_id, title (one line, 1–200 characters), **team** (`nurse | anaesthetist | admin`, answers 1), `created_by` or `created_by_rule` (C6), `done_by` + `done_at`, `reopened_by` + `reopened_at`. Routes `GET/POST …/episodes/{eid}/tasks`, `PATCH …/tasks/{tid}` `{done, revision}`, audited (`task.list`, `task.create`, `task.update`). Anyone at the hospital adds, completes or re-opens any task. ([build notes](sessions/S21-tasks-worklists/build-notes.md))
- **A task change is an episode change:** it sends and bumps the episode's revision (stale → 409 `stale`), writes a General notes line, and is refused on a cancelled episode (S20: read-only except notes). *Builder's default.*
- **The six worklists' rules are one SQL function, `in_worklist`** (migration 00022), used by `GET …/episodes?worklist=<key>` and by **`GET …/worklists`** (the tabs' labels, help lines and counts; also `/api/o/{oid}/worklists` for All hospitals), so a count is always its list's length. Counts carry no names or IDs and aren't audited; the lists are. *Builder's default for the route.*
- **The rules:** Ready for review (the status); Not responding (S16's flag while still HQ not complete); Anaesthetic review needed (an open anaesthetist task); **Procedure within N days: from today to today + N inclusive at the hospital**, not Ready for admission / Admitted / Discharged / Cancelled; On hold; Open tasks. **A cancelled episode is in none.** N is `worklists.soonDays` (per hospital, default 14, answers 3); "All hospitals" labels the tab "Procedure soon", since hospitals may differ. *Builder's default for the edges.*
- **Status messages (EPI-08, answers 4):** four switches (`statusMessages.readyForAdmissionSms|Email`, `…cancelledSms|Email`, per hospital, **off by default**) beside four templates in a new Settings group, "Messages when the status changes". A status template may leave out `{{link}}` (`settings.Def.LinkOptional`); an invite's or reminder's still can't. **`{{link}}` is the patient's sign-in page (`/signin`), not a token link.** Sent only on a clinician's **pick of Ready for admission** and on **Cancel** (not on an undo that lands on Ready for admission), after the move is committed; recorded in `episode_messages` (kind `status`, with the status); a failure is noted in General notes and shown on the episode as "Message not delivered", never refused. *Builder's default for the link and the moves.*

### 21 Oct 2026: S20, every status, cancel and re-open (builder, automatic run; tested 21 Oct, test report 2 all passed; verified 21 Oct)

- **The transition table is Go's** (`internal/episode`: `allowed[(from, to)] → Rule{move, least role, reason, date}`, `Check`, `Options`, `UndoTarget`), unit-tested on every pair of statuses × every move × every role. The patient's submit and Complete HQ review go through it too. The web app shows only the moves Go sends (`moves`, `undo` in `GET …/episodes/{eid}`). ([build notes](sessions/S20-statuses/build-notes.md))
- **"Any other of these five" includes Ready for POA as a target** (the accepted table read literally), so On hold → Ready for POA can be picked by hand. *Builder's default.*
- **Undo** (answer 2) is its own route, `POST …/undo` {reason, revision}. It steps back the newest pick, Admit or Discharge not yet undone, and clears an admission or discharge date with it; the workflow's moves, re-open, cancel and undo itself stop it. *Builder's default for the route and the stopping points.* **Settled 21 Oct (Rahul, asked by the conductor): keep as built.** Undo stops at completing the review, an on-behalf submit, re-open, cancel and an earlier undo. **Reason:** undoing the review would be a re-open by any clinician, which answer 1 gives only to a super clinician; re-open stays the way back ([verify.md](sessions/S20-statuses/verify.md)).
- **The status history is a new table, `episode_status_events`** (move, from, to, reason, user, rule, undone_at, ordered by an identity `seq`); General notes still get a readable line. Creating an episode records its first status; episodes made before 00021 have no history.
- **The episode revision** counts the changes staff make and the workflow's (submit, review), not the jobs' bookkeeping or a resent invite. Every staff write from a page must send it (400 without it); a stale one gets **409 with `code: "stale"`**, which the page shows with Reload, keeping what was typed. This closes the two pending.md §1a items (ASA grades overwriting each other; the 409 compared the wrong status). The details `PATCH …/episodes/{eid}` no longer takes `status` (400 if sent). Complete HQ review sends none: its guard is the status.
- **Admission and discharge dates** are `date` columns (`admitted_at`, `discharged_at`, as the brief names them), required with their move, not in the future at the hospital, discharge not before admission. *Builder's default.*
- **A cancelled episode is read-only except for notes**; cancel revokes its invite links (ending the sessions opened from them), finishes its jobs, and the follow-up queries skip cancelled episodes in case a runner had already claimed the job. The patient's form answers 403 `code: "cancelled"`; the home's state is `cancelled`. Details are optional for the three fixed cancel reasons, required for Other. *Builder's default for the details.*
- **The status forms are inline (no modals)**, as S19's procedure forms, with `noValidate` and the date's `validity.badInput` check.
- **Reasons are one line, and Enter sends the form** (test report 1, F-1): a reason is a sentence, kept on one line in the history and General notes, and Enter sends every other form in the app; a textarea with Ctrl/Cmd+Enter would be the only such key rule. At most 1,000 characters in the UI, as the API.

### 20 Oct 2026: S19, procedures and episode dates (builder, automatic run; tested 21 Oct, test report 4 all passed; verified 21 Oct)

- **`procedures`** is the trust's list (PRC-01): org_id, `hospital_id` (NULL = every hospital, S19 answers 2), name (unique in the trust in any case, retired ones included), code, specialty, `default_questionnaire_id`, `info_content` and `proms_plan` (answers 3: empty JSON lists until C5/C9 define their items; returned by the API, not yet editable), retired (answers 5), updated_by and times. Routes: `GET/POST /api/o/{oid}/procedures`, `PATCH /api/o/{oid}/procedures/{prid}`; POST and PATCH write audit rows (`procedure.create`, `procedure.update`). ([build notes](sessions/S19-procedures/build-notes.md))
- **Who manages it: whole-trust super clinicians and admins** (answers 1). A super clinician of one hospital only (Sue Super) can't, even for that hospital's own procedures, as for the trust's questionnaires. Everyone in the trust reads the list (hospital-only members see the trust's and their hospitals' own). *Builder's default for the hospital-only case.*
- **The default HQ names a questionnaire, not a version** (the plan said `default_version_id`): a new episode gets its published version at the time, so a new version doesn't strand it, as with S10's `defaultHq`. **Go resolves the HQ a procedure picks** at a hospital (`GET …/procedures?hospital=`: its own default if published and used there, else the hospital's effective `defaultHq`, else none); the New episode page applies it, and also opens with the hospital's default HQ (`published-hqs` now marks it `isDefault`). That wires up S10's `defaultHq` for the first time. *Builder's default.*
- **Specialty is optional**, like the code. Carried-over procedures have none, and requiring it would block retiring them. *Builder's default.*
- **The episode's free-text procedure is carried across, not re-seeded** (answers 4a): migration 00020 makes each trust's distinct texts (trimmed, any case once) its procedures and links the episodes. Unlike S18's reshape, this is a cheap and exact data migration, and it keeps any episode made since 00019 (locally and on QA). *Builder's default, within the conductor's note.*
- **The procedure and both dates can be changed until the HQ review is complete** (409 after), and each change writes a General notes entry ("Procedure set to …", "POA appointment date set to 2026-10-28"; ISO dates in the note text, since notes are stored text). The episode keeps a retired procedure; a new choice must be offered at its hospital. *Builder's default for the notes.*
- **The seed adds one published "Demo pre-op HQ" per trust** (made up, the e2e journeys' HQ; created once, never reset) and six procedures (four in Trust A, one of them for Hospital A1 only and one with no default HQ; two in Trust B), put back as seeded on every run. Without a seeded HQ the procedures couldn't have a default HQ. *Builder's default.*
- **Forms that Enter sends are `noValidate`** (S19 test report 2): a real `<form>` with a `required` field otherwise shows the browser's own "Please fill in this field." bubble, in the browser's language, instead of the app's message. **And a browser date input typed only in part has the value ""** (test report 1): check `validity.badInput` before sending, or the date is dropped silently (`features/episodes/dates.ts`).
- **Scroll to a message after the list reloads, instantly** (test report 3): a smooth scroll started before a reload is cut short by the page changing under it.
- **Gotcha:** "Changes are audited like the other admin changes" (the brief): the Staff and Settings changes are in fact not in the audit log (only patient data and the log's own reads are). The procedure changes are, with no patient or episode IDs.

### 20 Oct 2026: S18, episode forms (builder, automatic run; tested 20 Oct, test report 1 all passed; verified 20 Oct)

- **`episode_forms`** holds an episode's questionnaires: id, episode_id, version_id, kind (`hq|assessment|proms`), status, due_at, closes_at, submitted_at, plus submitted_by and created_at. Every episode gets its HQ form in the same statement that creates it, and a unique index allows one HQ per episode. ([build notes](sessions/S18-episode-forms/build-notes.md))
- **A form's status is `open` or `submitted`** for now, and a CHECK keeps it in step with `submitted_at`. C7 and C9 add other values if they need them. *Builder's default.*
- **`patient_submitted_by` (PAT-09) moved to the form as `submitted_by`**, beside `submitted_at`, though the brief named only the submit time. *Builder's default.*
- **The answers are keyed by `(episode_form_id, chapter_id, actor)`**, with the S02a revisions as they were.
- **The episode's JSON is unchanged:** `versionId`, `patientSubmittedAt` and `patientSubmittedByName` now come from the HQ form under their old names. One field is added: `hqFormId`.
- **"Reshape and re-seed" (S18 answers 2) is migration 00019 itself.** It deletes every episode, with its answers, notes, messages, follow-up jobs, invite links and the sessions opened from them, and keeps patients, accounts, sign-in links and questionnaires. The seed has no episodes, so there's nothing to re-seed. **QA needs no hand reset:** the pipeline's migrate step does it on the next push (this replaces the §5 item). A migration test proves it on a database in the old shape. *Builder's default; simpler and safer than editing migration 00004, which would have needed every database dropped by hand.*

### 20 Oct 2026: answers to the S18–S22 briefs (Rahul, asked by the `/run-sessions` conductor)

The full answers are in each session's `answers.md`. Every answer not listed here took the brief's recommendation. Rahul chose differently on these:
- **S18: `episode_forms` gets the plan's full column set now:** id, episode_id, version_id, kind (`hq|assessment|proms`), due_at, closes_at, status, submitted_at. The brief had recommended only the columns used today. ([S18 answers](sessions/S18-episode-forms/answers.md))
- **S18: reshape the tables and re-seed, with no data migration.** The local database is disposable (§1). **QA on Neon needs the same reset when this is pushed**, and that's Rahul's step at push time. (§5)
- **S18: today's URLs stay.** They resolve the episode's one HQ form on the server (recommended).
- **S19: the INF-04 information content and PRM-01 PROMs plan columns are added to procedures now**, not in C5/C9. ([S19 answers](sessions/S19-procedures/answers.md))
- **S20: any clinician may undo any manual status move, one step at a time and as often as needed, with a reason.** Each undo goes back to the status before the latest move. The brief had recommended a super clinician, limited to admitted→ready_for_admission and discharged→admitted. Cancel stays final. ([S20 answers](sessions/S20-statuses/answers.md))
- **S22: archive and un-archive both need a super clinician** (un-archive with a reason). The brief had recommended that any clinician could archive. ([S22 answers](sessions/S22-archive/answers.md))

### 20 Oct 2026: plan C3 split into sessions S18–S22 (Rahul)

- **Five sessions, in this order:**
  1. S18, episode forms;
  2. S19, procedures;
  3. S20, statuses, cancel and re-open;
  4. S21, tasks, worklists and status messages;
  5. S22, archive, auto-archive and closing an account.

  Each ends with something a tester can try. The forms reshape stays its own session, not mixed into a features diff. ([sessions/README.md](sessions/README.md))
- **`episode_forms` now, in C3, not deferred to C7.** C4–C6 add more code that reads answers, so reshaping first means less to change later. ([S18 brief](sessions/S18-episode-forms/brief.md))
- **EPI-08 (status messages to the patient, a Should) is in C3**, in S21, though the plan allowed cutting it. ([S21 brief](sessions/S21-tasks-worklists/brief.md))
- **EPI-09 (a patient closes their account, a Must missing from C3's plan text) goes in S22**, with archive. ([S22 brief](sessions/S22-archive/brief.md))
- **C3's end-to-end journeys are J9 (the full lifecycle) and J9b (cancel)**, because `test_j8_reminders` already uses J8.
- ~~**Each brief's "Ask Rahul first" items are still open.**~~ *Answered 20 Oct* (the entry above, and each session's `answers.md`).

### 20 Oct 2026: a sales brochure for Dutch hospitals (Rahul, 19–20 Oct)

- ~~**The name is written PreOpsis** (capital O), and the wordmark shows **Op** in teal (Rahul, 20 Oct).~~ *Superseded 20 Oct (Rahul): it's **Preopsis**, one word with a capital P only, as in the app, `docs/brand/` and the rename entry below. **The brochures and the seller sheet still say PreOpsis and need changing back.***

- **The Netherlands is a target market.** Rahul asked for a brochure a salesperson can use to sense-check Preopsis with Dutch hospitals. It covers pre-operative screening and post-op recovery, and is written around the hospital's concerns.
- **Where it is: `sales/nl/`.** It holds:
  - the Dutch brochure (primary, 4 A4 pages) and an English copy;
  - an internal 2-page seller sheet;
  - To rebuild: `e2e/.venv/bin/python sales/nl/build.py`. It renders the PDFs with Playwright's Chromium, writes a PNG of every page to `sales/nl/preview/`, and fails if text overflows a page.
  - The copy for both languages is in `sales/nl/src/copy_text.py`, and one template is `src/brochure.template.html`.
  - The brochure uses the brand in [brand/README.md](brand/README.md).
  - All illustrations are HTML/SVG. Rahul offered AI image generation, but the only image tool connected was a Figma account in someone else's name with paid credits, so none was used.
- **How the brochure presents features (agreed 19 Oct):**
  - **the full offer in the present tense, with no "roadmap" or status tags;**
  - EHR integration worded as *set up per hospital* ("ingericht per ziekenhuis");
  - one footer line: "Functionaliteit en planning in overleg" (functionality and timing by arrangement).
  - **Reason:** Rahul wanted it to read as a complete product. **The limit agreed with it:** no claims of certification (NEN 7510, ISO 27001, CE/MDR), no customer names and no statistics presented as our own results. Making those claims would be false and a risk in a tender.
  - **The true status of each feature is in the seller sheet** ("What exists today": Live / Planned / Not in plan), so the seller never promises a date or a certificate.
- **⚠ The brochure offers more than the plans.** These features have no plan:
  - HiX, Epic and Nexus integration via FHIR/HL7, which contradicts **A-10** (EHR integration out of scope, saas-requirements §0);
  - BSN as the identifier;
  - hosting in the EU;
  - Dutch and other languages for patients;
  - DigiD, if hospitals ask for it.
  - **Rahul to decide** whether to change A-10 and add plans for these, or to word the brochure more cautiously. Until then, the seller sheet marks them "Not in plan".
- **What desk research found about the Dutch market (20 Oct; sources in the seller sheet):**
  - **EHR shares:** ChipSoft HiX has about 71% of hospitals, Epic 14% and Nexus 11% (M&I/Partners, 1 Jan 2024).
  - **HiX already runs questionnaire-based POS triage** with a provisional ASA. At Diakonessenhuis, 55% of screenings are digital.
  - **Dräger E-POS** is in use at WZA and Noordwest.
  - **PROMs:** the LROI's set includes EQ-5D-5L and HOOS-PS/KOOS-PS.
  - **Medical device risk:** under EU MDR Rule 11, suggested ASA and auto-triage are likely **Class IIa**, which needs a notified body. That matters for plan C6 in the Netherlands.
  - **The pitch is "alongside your EHR"**, not a replacement for it.

### 20 Oct 2026: the product is renamed Preopsis, with a logo and a teal brand (Rahul)

- **Name: Preopsis** (was Poised). **Reason:** Rahul didn't like "Poised". "Preopsis" is "pre-op" plus *-opsis* ("view", as in Synopsis, the incumbent UK pre-op product's name), so it reads as a pre-op product to clinicians and to a patient getting a text.
  - **Dropped after checks:** Optima (Optima Health, a UK occupational health provider), Fitfor (several UK FITFOR companies; "Fit 4 Surgery" is a University of Birmingham prehab app in NHS trials), Circa (a US clinical-documentation AI), Paratus (Paratus Health, YC: AI patient intake), Fettle (SH:24's UK health service).
  - **Checks for Preopsis (20 Oct):** no web use, no UK company, and `.com`, `.co.uk`, `.health` and `.io` looked unregistered (by DNS only). **Still to do (Rahul):** register the domains.
  - **UKIPO search (20 Oct, trademarks.ipo.gov.uk; covers UK marks and international marks designating the UK):**
    - **No PREOPSIS mark.** The "Similar" search found only unrelated words (PROPESS, SPIROSEP, PHORESIS; PROMESIS in 9/42 has expired).
    - **The word OPSIS is registered**, and "Preopsis" contains it whole. None of those marks covers healthcare software:
      - **Opsis AB / Opsis Holding AB** (Sweden): class 9 since 1988, and class 42 ("scientific and technological services… industrial analysis"). Air and emissions monitoring.
      - **Halliburton**: class 9 (2022), oil-well pressure gauges.
      - **Icare Finland**: class 10 (2022), eye-imaging devices for retinal disease and glaucoma, for medical use. This is the nearest to healthcare, but it covers hardware, not software or services.
      - **OPSIS THERAPEUTICS**: classes 5 and 10 (medicines).
    - **SYNOPSIS is registered only in classes 3, 5 and 10** (cosmetics, pharma, medical devices); there's no SYNOPSIS mark in 9 or 42, so the worry about VitalHub's name is smaller than thought.
    - **Verdict:** clear enough to use, and to file PREOPSIS in classes 9, 42 and 44. **Before filing**, ask an attorney whether the OPSIS owners (Opsis AB and Icare above all) could oppose. They're in different fields, which makes confusion unlikely.
- **Renamed: everything a person reads.** The web app (tab title, headers, patient pages, admin and settings text), the API's messages, the patients' sign-in and contact emails and texts, the email From name, Cognito's invitation and reset emails, and **the UK SMS sender ID: `PREOPSIS`** (`infra/countries.py`).
  - **The sender ID changes on the next deploy.** CDK makes a new `AWS::SMSVOICE::SenderId` and deletes POISED. Like POISED, it's unregistered (pending.md).
  - Texts are 2 characters longer ("Preopsis:" for "Poised:").
- **Kept as `poised` on purpose** (as the 2 Oct rename kept `sj`): the GitHub repo `rlamba89/poised` and the local folder, the Go module path, the npm packages `@poised/*`, the patient cookie `poised_patient` (renaming it signs every patient out), the browser storage keys, and **every AWS name** (`poised-<env>-…` stacks, functions, topics and configuration sets, the Cognito pool and its domain prefix, the pipeline and `poised-github`, the CDK bootstrap qualifier `poised`, the cost tag `poised-env`). Renaming those would replace live QA resources, such as the staff pool with its users, and would need the GitHub connection approved again. The seed's `@poised.example` test emails stay too.
- **Brand:** the logo is an eye with a tick (seen and checked before the operation), and the wordmark shows **op** in teal. The primary colour is **teal `#0f766e`** (Mantine's `brand`, shade 7, about 5.5:1 against white), deliberately not NHS blue. The patient bar is now teal. The files and rules are in [brand/README.md](brand/README.md). In the app the logo is `<Logo />` (`apps/web/src/features/brand/Logo.tsx`), and the favicon is `apps/web/public/icon.svg`.

### 15 Oct 2026: S17 tested; QA's staff-side manual tests run locally (Rahul, S17 answers 10)

- **From now on, the staff-side cases of a QA session's manual test run locally** on `make dev` (the same code, the dev login, captured messages). **Reason:** the tester and verifier agents won't type a password into Cognito's hosted sign-in page on a non-local site, so S17's test report 1 on QA was blocked for every staff case.
  - Agents still check on QA whatever needs no staff sign-in: read-only AWS checks, the Lambdas' logs, the patient's pages.
  - What only QA can show (e.g. a real SES bounce) becomes a hand check for Rahul, backed by the builder's own run on QA and the automated tests.
- **S17 tested 15 Oct:** test report 2 (local) all passed. Report 1 (QA) was blocked at sign-in, but its AWS checks passed. The hand checks are in [S17's build notes](sessions/S17-aws-messaging-jobs/build-notes.md).

### 14 Oct 2026: S17, the jobs Lambda and real SES/SMS on AWS (builder, automatic run; deployed to QA by hand; tested 15 Oct)

- **QA keeps capturing every message, and really sends only to named test recipients** (builder's choice in automatic mode, within answers 4–5). This is `messaging.TestRecipients`, enabled by `qa_test_recipients=True` on `uk-qa`. The list is the SSM SecureString `/sj/qa/messaging-test-recipients` (Rahul's own email and mobile; never in the repo). SES's mailbox simulator (`@simulator.amazonses.com`) is always allowed.
  - **Reason:** switching QA to `send` for a test run would break the journeys and testers, who read the captured messages, and in the sandbox the made-up mobiles would fail. With test recipients, nobody but Rahul can receive anything from QA, even after the account leaves the sandbox.
  - **CDK refuses at synth:** `capture` outside QA; `send` on QA; `qa_test_recipients` outside QA.
  - `make test-recipients` / `-verify` / `-status` / `-off` manage the list and the sandbox verifications, and print no contact.
- **QA's jobs run every 15 minutes; Training and Production every 5** (Rahul, answers 7). The setting is `jobs_every_minutes` in `countries.py`.
  - **Reason:** each run wakes QA's Neon database, which sleeps after 5 idle minutes. A 5-minute schedule would keep it awake about 180 compute-hours a month, against the free plan's 100.
  - The EventBridge Scheduler schedule has no retries; the next run comes soon.
  - `/qa/jobs` says "a schedule on AWS runs the jobs every 15 minutes" (`JOBS_SCHEDULE_MINUTES`; `JOBS_TIMER=false` on AWS).
- **The UK sender ID `POISED` is made by CDK and kept** (answers 9).
  - The account had no origination identity, so AWS refused every text, even the sandbox's verification code, with `NO_ORIGINATION_IDENTITIES_FOUND`.
  - The sender ID shows a **$0.00 monthly fee** and works **unregistered**: the verification code and a real invite text reached Rahul from POISED.
  - Registering it (the UK form) is paperwork for the demo account (pending.md).
- **Undelivered messages (OPS-05):**
  - **Recording:** every text and email about an episode is recorded in `episode_messages` with the sender's ID. A send that's refused, or that AWS later reports undelivered, is marked failed with a reason in words.
    - SES events arrive through its configuration set: bounce, reject, rendering failure.
    - SMS events arrive through its configuration set: the ten `TEXT_*` failure types.
    - Both publish to the topic `poised-<env>-deliveries`, which runs the jobs Lambda.
  - **What staff see:** an "Invite not delivered" badge and a "Not delivered" box on the episode, covering everything since the latest invite or Resend, plus an Automatic General note.
  - **The alarm:** it counts `DELIVERY_FAILED` log lines from all three functions. A second alarm covers jobs Lambda errors. Both email `/sj/alert-email` through `poised-<env>-alarms`.
  - **Proven on QA:** an episode for `bounce@simulator.amazonses.com` showed the failure within about a second, and the alarm fired.
- **SMS goes only to the deployment's country, enforced twice:** a protect configuration on the SMS configuration set allows only the entry's phone country and blocks the other 244 codes (AWS's default is ALLOW; the list is in `infra/sms_countries.py`, fetched from AWS 14 Oct), as well as PAT-11's check in Go.
- **A failed sign-in-link send now answers 202, like "no account"** (pending.md's "For S17" item, S15 verification 1 finding 4). Before, it was a 502 only for contacts with an account. The timing difference for a slow send remains (pending.md).
- **Gotchas:**
  - **CloudFormation's `AWS::SMSVOICE::SenderId` takes capitals only** (`^[A-Z0-9_-]+$`), so "Poised" failed early validation. `countries.py` checks for capitals.
  - **AWS's SMS error messages contain the phone number** (and SES's the address). The real senders log and return only AWS's error code; `test-recipients.sh` hides AWS's error text.
  - **The functions read the test recipients at cold start.** `make test-recipients` / `-off` therefore change the three functions' description, which makes them start afresh.
  - **With no origination identity, the Lambda's `SendTextMessage` failed as `AccessDeniedException`** (the policy allows the account's sms-voice resources). It went away once POISED existed.
- **Details:** [S17 build notes](sessions/S17-aws-messaging-jobs/build-notes.md); `infra/README.md` ("Texts and emails on QA").

### 14 Oct 2026: S12, staff sign-in with Cognito and MFA (builder, automatic run; deployed to QA by hand; tested 14 Oct, test report 2 all passed; verified 14 Oct, [verify.md](sessions/S12-cognito-signin/verify.md); plan C1 done)

- **The Cognito feature plan is Essentials** (S12 answers item 1: the builder checks the docs and picks the cheapest with managed login + TOTP). **Reason (AWS docs, 14 Oct 2026):** managed login exists only in Essentials and Plus (Lite has only the classic hosted UI); TOTP MFA is in every plan; Lite and Essentials are both **free to 10,000 monthly active users** (a permanent free tier), then $0.0055 vs $0.015 per user; Plus has no free tier. So Essentials costs £0 until a deployment has more than 10,000 active staff. Clear-cut, so not asked.
- **`AuthStack` (`poised-<env>-auth`) comes after WebStack:** the app client's callback and sign-out addresses are the web address, which is Amplify's own until there's a domain. So it writes the pool's IDs to SSM (`<prefix>/cognito-user-pool-id`, `-client-id`, `-domain`), which the Lambdas read at cold start like the web address, and it attaches staff-api's `AdminCreateUser` permission (this pool only) itself: ApiStack never refers to it, so there's no cycle.
- **The pool:** email is the username (case-insensitive, and **not changeable**: the API trusts the verified email); **TOTP required** (no SMS or email codes); no self sign-up; reset by email; temporary password 7 days (the invite's deadline, S09); passwords ≥ 12 with all four kinds. **Cognito's own email** (`COGNITO_DEFAULT`), so invitations reach any address while SES is in the sandbox (about 50 a day; SES before a trust: pending.md). The invitation names the app's address. QA's pool is deleted with its stack; a real stage's is retained with deletion protection. The managed login's domain prefix is `poised-<env>-<8 hex of sha256(account)>` (prefixes are unique across accounts in a region).
- **The sign-in:** `/login` → **Sign in** → the managed login (code flow with **PKCE**, a public client with no secret; verifier and state in the tab's `sessionStorage`) → `/auth/callback` posts `{code, verifier}` to **`POST /api/auth/callback`** → Go exchanges the code at the token endpoint, **verifies the ID token** (RS256 against the pool's JWKS, cached per Lambda instance and refetched at most once a minute for an unknown key; issuer, our client as audience, `token_use=id`, expiry, `email_verified`), finds the user **by `cognito_sub`, else by the verified email (then stores the sub)**, and sets **our own session cookie** (the same `startStaffSession` as the dev login: the suspended-trust and expired-invite refusals, S09/S10). **No account is ever created at sign-in** (only invites make accounts): someone Cognito knows but Poised doesn't gets 403. The browser never holds a token. **Sign out** also signs out of the managed login (its `/logout`, back to `/login`), or the next Sign in would skip the password.
- **The real `cognito.AWS` client:** `AdminCreateUser` (email marked verified, emailed invitation); an email the pool already has is fine (they sign in with it); **Resend** (`RESEND`) creates the user if the pool lacks it (e.g. a re-created pool) and does nothing if they already set a password.
- **No dev login on AWS, enforced twice:** `countries.py` has no `dev_login` field any more (an entry can't ask for it; a CDK test checks no stack sets `DEV_LOGIN`), and **Go refuses `DEV_LOGIN` with `CONFIG_STRICT`** (supersedes S05's "strict doesn't refuse DEV_LOGIN", which was only for QA until S12). Under strict the pool is required.
- **The testers' tools need a signed-in platform admin where the dev login is off** (S12 answers item 3): `/api/qa/messages`, `DELETE /api/qa/rate-limits`, `/api/qa/client`, `/api/qa/jobs` (and run, clock): 401 signed out, 403 for anyone else. **Locally (dev login on) they stay open**, because anyone can sign in as the superadmin there anyway. The pages say "Sign in as a platform admin to use this page." with a link.
- **Amplify sends security headers on every path:** `X-Frame-Options: DENY`, CSP `frame-ancestors 'none'` (only that directive, so nothing of the app breaks), HSTS one year, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer` (S14's suggestion).
- **QA test users** (S12 answers item 2): **`cmd/qa-users`** (QA only: refuses any other `STAGE`) gives each **seeded** staff account (by the seed's fixed ids) a pool user with a generated password and an enrolled authenticator (it signs in with `ADMIN_USER_PASSWORD_AUTH`, which the app client allows **on QA only**, and completes `MFA_SETUP`), and stores `{name, email, password, totpSecret}` in the SecureString **`/sj/qa/test-users`**. Idempotent (keeps stored secrets, puts the password back; re-makes a user whose secret it doesn't know). **The pipeline runs it after each seed** (its role may use the Cognito admin actions only on the pool tagged `poised-env=<env>`); `make qa-users` by hand. Testers: `make qa-code WHO=…` prints the current code, `make qa-password WHO=…` copies the password to the clipboard (never printed).
- **The journeys on QA sign in through the managed login** (Playwright + `pyotp`), once per user per run, reusing the session cookie (Cognito takes each code once); the tools are used as Sam Superadmin. `make e2e-users` (or the pipeline's E2E step) fetches the test users to the gitignored `e2e/.qa-users.json`. **`test_staff.py` and `test_j6_onboarding.py` are `dev_login`-only** (they sign in as people they've just invited, whose temporary password is in an email; and Cognito's email allows about 50 a day); **`test_cognito_signin.py` is `cognito_only`**.
- **`make aws-platform-admin ENV=… EMAIL=…`** runs `cmd/platform-admin` against an AWS environment (its database URL and pool from SSM): the new superadmin gets Cognito's invitation email.
- **Gotcha: the variables are `STAFF_POOL_ID`, `STAFF_POOL_CLIENT_ID`, `STAFF_POOL_DOMAIN`, not `COGNITO_*`:** Rahul's `.env` holds another project's `COGNITO_*` values, which `make` exports, and the API (rightly) refused a partial pool.
- **Gotcha: the Go AWS SDK can't refresh an SSO sign-in that the AWS CLI still can** ("refresh cached SSO token failed … InvalidGrantException"), so the scripts that run Go against AWS with a named profile hand it the CLI's credentials (`aws configure export-credentials --format env`, into the process only).
- **Gotcha (test fix): the audit journey read the list before it reloaded** after a filter; on QA the patient's rows can be as many as Cara's, so the count check passed early. It now waits for the patient's rows to go first.
- **Shared workstations (S12 test report 1, findings 1 and 2; the conductor's direction):**
  - **Pages are never stored** (`Cache-Control: no-store` on every path from Amplify), while the hashed files under `/assets/` are kept for good (`public, max-age=31536000, immutable`; they were revalidated on every load before). **Staff Sign out empties the page** before leaving for the managed login's sign-out, and **a page the browser restores from its back/forward cache reloads** (`pageshow` with `persisted`), so Back after a sign-out never shows the previous person's patient list.
  - **A sign-in Poised refuses after the managed login let the person in** (suspended trust, expired invite, no Poised account, a bad state) **also signs out of the managed login** (its `/logout`, back to `/login`, the message carried in the tab's `sessionStorage`), so the next person at the computer gets the password page instead of being signed straight back in as the refused person.
- **Open (test report 2, exploring 1; older than S12):** another open tab keeps showing patient names after a sign-out in a different tab until a click. In pending.md with the fix (a 401 goes to `/login`, and the sign-out tells the other tabs).
- **Gotcha: Amplify applies changed custom headers only from the branch's next deployment** (a CloudFormation update alone changes nothing you can see). The pipeline deploys the web app after the stacks, so it's automatic; by hand, run `make deploy-web` after `make deploy-stacks`. Two patterns (`'**'` and `'assets/**'`) are merged, the more specific one's `Cache-Control` winning (checked on QA).
- **Details:** [S12 build notes](sessions/S12-cognito-signin/build-notes.md).

### 14 Oct 2026: S07, the pipeline (builder, automatic run; test report 1: 17 of 17 passed; verified 14 Oct)

- **The pipeline follows `master`,** the GitHub repo's default branch and the local branch (`PIPELINE.branch` in `infra/countries.py`). **Reason:** plan F4 said `main`, but renaming would change the GitHub default and every clone for no gain. Amplify's branch `main` is unrelated (it's only where the web zips go).
- **The pipeline's home (account, region, repo, branch) is `PIPELINE` in `infra/countries.py`,** so no stack names a region (A-20). The environments' stages live inside the pipeline stack, so hand deploys name `pipeline/<env>/*` (`make deploy-stacks` does); the stack names are unchanged.
- **The GitHub connection `poised-github` is created by CDK** (Pending) and approved once in the console (S07 answers item 4); the click-path is in `infra/README.md`. Everything else is `make aws-setup` + `make deploy-pipeline`, then a push.
- **Synth on arm64 Amazon Linux 2023; the journeys on Ubuntu x86** (CodeBuild `standard:8.0`). **Reason:** the Lambdas are arm64 and the AL2023 arm image has Go 1.26, Node 22 and Python 3.12; Playwright's WebKit supports only Ubuntu and Debian.
- **The integration tests' Postgres in the pipeline comes from ECR Public** (`public.ecr.aws/docker/library/postgres:<compose's major>`), because Docker Hub limits anonymous pulls from shared build hosts. `make test-integration DB_UP=` skips docker compose.
- **The journeys' results** (JUnit report, traces of failed journeys) are copied to an S3 bucket after every E2E run, passed or failed, and kept 30 days. Not a CodeBuild report group: CodeBuild creates those outside CloudFormation, so they'd outlive `cdk destroy`.
- **The seed has a run-twice integration test** (`cmd/seed`), since the pipeline seeds QA after every deploy.
- **The "add a country" proof (14 Oct): a throwaway `us-qa` in us-east-1 took about 10 minutes** from the new entry to J1–J5 passing, then was destroyed; nothing is left in us-east-1. **The journeys' test data is the UK's** (a UK mobile, UK dates), so they need to read the country before a second country's pipeline stage (pending.md).
- **Pushes are Rahul's (S07 answers item 5):** in the automatic run, Claude Code's permission system refused the builder's `git push`, so S07 was committed locally before verification and Rahul pushed each commit himself (S07's code, then the visible change, the failing test and its revert). He approved the connection only after S07's code was on GitHub, so the self-updating pipeline never ran from code without its own definition.
- **Gotcha (S07 test report 1): a failed Synth log ends with `no matching base directory path found for infra/cdk.out`,** a side effect (the artifact upload after a failed build). The real failure is the `Command did not exit successfully …` line about 20 lines above it (`infra/README.md`). **The pipeline's CodeBuild projects are named `Pipeline…` by CDK, not `poised…`** (pending.md).
- **The pipeline is proven (14 Oct):** pushes go green through QA e2e in 11 to 15 minutes (43 journeys); a failing unit test stops the run at Synth in under 3 minutes, before any deploy.
- **Verification 1 (14 Oct): verified.** It confirmed in AWS that runs R1, R2 and R4 and the tester's run were green, that R3 ran no deploy action, and that no secret is in any build log or CodeBuild variable. **The follow-ups are in pending.md:** the pipeline reaches only its own account (decide before an environment elsewhere); migrations run after the new Lambdas; C10 must keep the demo seed and the journeys to QA; and the CDK bootstrap's `AdministratorAccess` in a shared account (CDK's default) needs Poised's own account, or a scoped execution policy, before go-live.

### 14 Oct 2026: S06, the Lambda adapter, the CDK app and QA on AWS (builder, automatic run; test report 1: 19 of 19 passed on QA; tested 14 Oct)

- **QA stays open to anyone with its address until S12 (Rahul, 14 Oct, S06 answers item 8).** Test report 1 found that, with no sign-in, anyone can: use the dev login (any account, the superadmin too); read the captured messages with live links; reset the sign-in limits; and run `/qa/jobs`. The web app also sends no security headers. **Rahul:** leave it and keep the URL private, because the data is made up and **S12 closes it**: Cognito replaces the dev login, **the QA tool pages (captured messages, `/qa/jobs`, the rate-limit reset, `/api/qa/client`) need a signed-in platform admin**, and **Amplify adds security headers** (CSP `frame-ancestors`/`X-Frame-Options`, HSTS, `X-Content-Type-Options`) (S12 answers item 3). After S12 nothing on QA works without credentials.

- **Poised has its own CDK bootstrap:** qualifier `poised`, stack `poised-cdk-toolkit` (`make aws-setup`; `infra/cdk.json`, `infra/stage.py`). **Reason:** the account's `CDKToolkit` belongs to another project; re-bootstrapping it could change it, and depending on it would break when that project changes or leaves.
- **Where the Lambdas' values come from** (`internal/config` `Resolve`, at cold start): the **database URL** from the SSM SecureString `/sj/<stage>/database-url` (`DATABASE_URL_PARAM`); the **token secret** from Secrets Manager (`TOKEN_SECRET_ARN`; generated by CDK, deleted with the stack); the **web address** from the SSM String `/sj/<stage>/app-url` (`APP_URL_PARAM`), **written by WebStack**. **Reason for the last:** with no domain the address is Amplify's own, known only once the Amplify app exists, and the app's rewrite needs the HTTP API's address, so passing it as an environment variable would be a cycle. A value set both ways is refused; on Lambda `CONFIG_STRICT=true` is required.
- **The budget alarm counts only the environment's resources,** by the tag `poised-env=<env>` on every resource; $6 a month (about £5; AWS Budgets bills in USD); emails at 80% actual and 100% forecast. **The address is in SSM (`/sj/alert-email`, put there by `make aws-setup` from `ALERT_EMAIL`)**, so it's not in the repo. **Reason:** the account hosts another project, whose costs would otherwise trip it. Billing learns a new tag only after a day's costs, so `make aws-setup` activates the tag on a later run.
- **F4's first check passed: cookies go through Amplify's reverse proxy both ways** (Set-Cookie out, Cookie in; staff and patient), so QA needs no custom domain for them. **QA is at https://main.d33kh0gi8vk83v.amplifyapp.com** (until there's a domain; a re-created Amplify app gets a new address).
- **The patient's address behind the proxies:** `FORWARDED_HOPS` (3 on AWS, measured on QA: `X-Forwarded-For` is "<browser>, <CloudFront edge>, <Amplify's outbound address>"; `GET /api/qa/client` shows it) tells `internal/serve` which `X-Forwarded-For` address is the browser's; addresses the browser sent itself are ignored. Calling the `execute-api` address directly can still dodge the rate limit (pending.md). **`GET /api/qa/client`** (capture mode only, like the other QA routes) shows the address the API counts and the header it got.
- **The session cookies are `Secure` on https** (staff and patient, and their sign-outs), as the patient's already was.
- **The "reload once" after a deploy has no time window:** `Root` clears the flag once a page renders (pending.md, S04 verification 2). **The SPA rewrite leaves files and `/assets/…` alone**, so a file gone after a deploy is a real 404. **`GET /api/config` gives up after 5 seconds.**
- **`countries.py` accepts only `language-REGION` locales** (e.g. `en-CA`), so a tag the browser can't use never reaches it.
- **Lambda pool:** `MaxConns=2`, `MaxConnIdleTime` 1 minute.

### 14 Oct 2026: the AWS phase starts, in a temporary account (asked up front by the conductor)

- **QA goes in AWS account `026090536258` (SSO profile `AdministratorAccess-026090536258`, eu-west-2) for now.**
  - **Reason (Rahul):** it's for his own testing until the demo; before sales it moves to another account.
  - The account also hosts another project's staging stacks; Poised's stacks must never touch them.
- **The CDK must need no manual AWS work**, so the stacks can be deleted here and re-created in a new account at any time. This widens S06:
  - an idempotent `make aws-setup` (CDK bootstrap; the Neon URL into SSM from an env var);
  - CDK generates the token secret in **Secrets Manager** (about £0.30/month per secret);
  - the £5/month budget alarm is in CDK.
  - Only the GitHub connection approval, the Neon sign-up and SES/SMS production access stay manual (CloudFormation can't do them).
- **Permissions for the AWS-phase run (Rahul, 14 Oct):** deploy and destroy Poised's own QA stacks; push to `rlamba89/poised`; S07's throwaway second-region proof. **No real messages** without a separate yes.
- **SES and SMS are in the sandbox in this account.** S17 tests in the sandbox to Rahul's own verified email and phone; production access is requested later in the demo account.
- **No domain yet:** QA uses Amplify's default address.
- **S12:** a QA-only command creates test staff users with a known password and TOTP secret.
- **Details:** the `answers.md` files of S06, S07, S12 and S17.

### 13 Oct 2026: S16, scheduled jobs, reminders and "not responding" (builder, automatic run; test report 1: 20 passed, 1 failed (X-1); tested 13 Oct, test report 2 all passed; verified 13 Oct)

- **Rahul (13 Oct, S16 answers 3–5):** the jobs' "Automatic" episode notes are enough (no audit rows), as built; answers staff enter for the patient count as "started", as built; **a returning patient can still reach a locked episode through an S15 sign-in link** (its own 5 tries, sent only to their own contact): kept as a fair way back in. The staff banner's "every link … is locked" is to be reworded to "the invite links are locked" later (pending.md); no extra round.

- **S14's lock rule extended to reminders (S16 test report 1, X-1; the conductor's direction):** **wrong dates of birth are counted per episode, not per link.** All of an episode's live invite links (the invite and each reminder's) share the five tries, and **the fifth locks them all together** (`episodes.dob_failed_attempts`; `locked_at` set on every live link, so every existing lock check, sessions included, still applies). A right date on any of them starts the count again. **Only staff's Resend clears a lock** (new link, older ones revoked, the count reset). **A locked episode gets no reminders:** a reminder day that passes while locked is counted but not sent (a reminder's new link would undo the lock), with a General notes entry ("Day 5 reminder not sent: the patient's link is locked…"), and isn't sent later; after a Resend the later reminder days carry on from the first invite as usual. "Not responding" still comes on day 7. **The staff banner shows the episode's lock** (any live link locked: `GetEpisodeInvite`). Sign-in links (S15) keep their own per-link count. Migration 00017. **Reason:** S14 promised five tries, then staff; per-link counts gave five per link (15 with the default reminders), and a reminder after a lock handed out a working link with no staff action.

- **Rahul's answers (12 Oct, S16 answers):** reminders go **at any time** (no 09:00–19:00 window), and **starting the HQ stops them** (any answer saved), not only submitting. Both over the recommendation.
- **Jobs are one function, `jobs.Runner.RunDue(ctx, now)`** (`internal/jobs`), with no server memory or disk (Lambda-ready for S17). **One `followup` job per episode** (`jobs` table), rescheduled to its next due time. Due rows are **claimed one at a time with `FOR UPDATE SKIP LOCKED`, which moves the row's `run_at` on by a 10-minute lease**, so nothing stays locked while a job runs. **Reason:** an earlier version held the row lock for the whole job and deadlocked when several runners shared a small connection pool (found by the four-runners test).
- **The decision is a pure function, `jobs.Due(followup, schedule, now)`:** reminders on the hospital's `reminders.days` (2, 5) while the patient hasn't started; **"not responding" once on `reminders.notRespondingDay` (7) if the HQ isn't sent, started or not** (EPI-02: "no submit"); nothing after submit or without an invite; suspended trust or hospital → nothing, look again hourly; **missed runs send only the latest reminder, once**. **Day n is the invite's local time n calendar days later in the hospital's zone.** Days count from the **first invite that reached the patient** (`episodes.invited_at`); a Resend doesn't restart them.
- **No double sending:** a reminder is counted on the episode (`CountReminder`, compare-and-set) **before** it's sent; a failed send is noted and not retried.
- **"Started" includes answers staff enter on the patient's behalf** (PAT-09).
- **A reminder sends a new link and the patient's earlier links keep working** (a Resend still revokes them). **Reason:** the patient may tap the original invite after the reminder, and a reminder mustn't end an open session; only a token's hash is stored, so the old link can't be resent.
- **The sending moved to `internal/invite`** (`Sender.Send`, kinds First/Again/Reminder), shared by the API and the jobs; reminders use S10's reminder templates. **The invite itself is still sent in the Create request** (not moved to a job; pending.md).
- **The local timer:** `cmd/api` (`make dev`) runs the jobs every minute (`Runner.Every`/`Tick`); **`JOBS_TIMER=false`** turns it off (`internal/config`). staff-api and patient-api never run it.
- **Testers' controls, `MESSAGING=capture` only (no sign-in, like the captured messages):** `POST /api/qa/jobs/run {asOf}`, `GET /api/qa/jobs`, `PUT /api/qa/jobs/clock {offsetMinutes}` (the timer's clock ahead by 0–90 days, in `qa_jobs_clock`; **`make seed` sets it back**), and the page **`/qa/jobs`**.
- **The episode shows a red "Not responding" badge** until the HQ is sent; General notes written by the jobs show **"Automatic"** (`episode_events.automatic`). **The jobs write no audit rows** (not a person's view or change; pending.md).
- **Details:** [S16 build notes](sessions/S16-jobs-reminders/build-notes.md).

### 13 Oct 2026: S15, patient accounts across trusts, sign-in links and the home (builder, automatic run; test report 1: 23 passed, 1 blocked; test report 2: 27 passed; tested 13 Oct, test report 3 all passed; verification 1's fix retested, test report 4 all passed; for re-verification)

- **Rahul confirmed (13 Oct, S15 answers 3–5):** matching by date of birth + contact; sign-in for returning patients only; a patient's contact change goes to every trust's record and the old contact is told; verification links last 24 hours.
- **Accepted risk (Rahul, 13 Oct, S15 answers 10, after verification 1's finding 1):** someone who shares a contact **and** a date of birth with an account holder but hasn't opened their own invite yet (e.g. a twin whose record isn't joined) can use a sign-in link on the shared phone or inbox to sign in to the **other** person's account: the name step is asked only when two or more accounts match. **Sign-in is not changed.** **Reason (Rahul):** "the twin use case is a very rare scenario; let's not mess up other things because of a 0.00001% chance." It needs the same date of birth and access to the shared phone or inbox (which already receives the other person's invite). In pending.md.
- **"← Your questionnaires" saves the answer just typed before it leaves** (S15 verification 1, finding 2), like Sign out and the on-behalf Episode link; if the save is refused it stays, with "Not saved".
- **The surname counts too** (**Rahul, 13 Oct, S15 answers 8**, after test report 2 found two people with the same first name, birthday and phone but different surnames merged): a record joins an account only when the **date of birth, the mobile or email, the first name and the surname** match, each ignoring case and spaces. A changed or mistyped surname keeps a separate account (fails safe). When several accounts share a sign-in link's contact and the date of birth, the link asks for the **first name and surname** (409 `code: "name"`, not a try; a wrong name is a try). Migration 00015 unjoined records merged under another surname. **Supersedes the first-name-only rule below.**
- **Wording (S15 answers 9):** the home greets by a capitalised first name; a sign-in link to a contact changed since it was sent is "expired", not a wrong date.
- ~~**Twins: the first name counts too**~~ *Superseded 13 Oct by the surname rule above (S15 answers 8).* (**Rahul, 13 Oct, S15 answers 6**, after test report 1 found twins on a parent's phone merged into one account). A record joins an account only when the **date of birth, the mobile or email, and the first name** (ignoring case and spaces: the SQL function `name_key`) match, at the invite, at sign-in and in the sweep. Another spelling stays unjoined and uses its own invite links (fails safe). When twins share the contact a sign-in link asks for the first name too (409 `code: "first_name"`, not a try). **Each home card says whose questionnaire it is.** Migration 00014 unjoined records already merged under another first name.
- **Rahul, 13 Oct (S15 answers 7), all five built:** adding a contact where there was none tells the existing contacts (as a change does); each contact box takes only its own kind; a record that joins later takes the account's contacts it lacks (never replaces one); an `account.join` audit row whenever a record joins an account; emails must be plain ASCII, at most 254 characters with a local part of at most 64 (staff-typed ones too).
- **Testers reset the rate limits themselves:** `DELETE /api/qa/rate-limits` and **Reset sign-in limits** on `/qa/messages`, in capture mode only (builder, after test report 1: one machine is one IP address, and the tester can't run `make seed`).

- **An account is one person: `patient_accounts` holds their sign-in mobile and email, the date of birth they proved, and (since round 1) their first name's key** (builder, automatic mode; confirmed by Rahul). A trust's record joins the account with **the same date of birth and the same mobile or email** (PAT-07 plus the date of birth), so a couple sharing an email or a parent and child sharing a phone stay two accounts. **Reason:** joining by contact alone would show one person another's questionnaires.
- **Records join at a right date of birth:** an invite's joins its record (making the account from it if none matches), and **also the person's other unjoined records, in any trust, with that date of birth and a contact that just reached them**; a sign-in link's joins the records with its contact and that date. **Reason:** otherwise a second trust's questionnaire wouldn't show on the home until its own invite was opened. Each join is in its trust's audit log (`account.join`).
- **A patient session belongs to the account** (`patient_sessions.account_id`), so it opens every joined record's episodes in every trust; it still ends with the link it was opened with (Sign out, Resend, lock) or 30 minutes idle. A link's expiry governs only opening it.
- **Sign-in links (PAT-06):** `POST /api/p/signin {contact}` (one box: mobile or email), then `/s/<token>` → Continue → date of birth → `/home`. **The link belongs to the contact; the date of birth picks the account.** Works once, within 15 minutes; five wrong dates lock it. **The answer is the same whether an account exists or not**; only accounts get a link (a first-time patient uses their invite).
- **Rate limits (S15 answers 2):** 3 messages per contact per 15 minutes and 10 per IP address per hour, counted in the database (`rate_limit_hits`, Lambda-safe), for sign-in links and contact verification alike; refused requests don't count. **`make seed` clears them.** A mobile outside the deployment's country is refused before counting (PAT-11).
- **The home (`GET /api/p/home`)**: every episode of the account, every trust, newest first, one query, with the hospital's name, questionnaire, procedure and state (to do, in progress, sent); suspended trusts and hospitals left out. **The patient's contacts are shown in part** (`•••123`, `s•••@example.org`), keeping S14's "no identifiers they didn't type".
- **Patients change their own mobile or email (PAT-10, S15 answers 1):** a link (`/v/<token>`, 24 hours) goes to the new contact; confirming it (no session needed) changes **the account and every trust's record**, and **the old contact is told**. Refused if another account born the same day uses that contact. **Reason:** S13 kept silent links from editing records and named PAT-10 as the deliberate way; reminders must reach the new contact.
- **`audited` can write several rows per request**, one per trust and hospital touched (`auditEntry.also`): the home (`home.view`, a row per hospital), joins and contact changes (a row per trust).
- **Behind AWS, the rate limit's client IP must be the patient's** (S06; pending.md).
- **Details:** [S15 build notes](sessions/S15-patient-home/build-notes.md).

### 13 Oct 2026: S14, patient invites and magic-link sign-in (builder, automatic run; test reports 1–3 all passed; verification 2's fix retested in test report 4: passed, with X-1 skipped by Rahul; verified 13 Oct, verification 3)

- **The patient's link shows nothing by itself** (S14 answers 1). The invite link is `/i/<token>`: **Continue** (a POST, so a link scanner can't use anything) returns only the hospital's name; the **date of birth** then starts a session. **After the check, the HQ has the patient's first name, full name, age (worked out in Go on the hospital's day) and sex** (**confirmed by Rahul 13 Oct**, answers 2: the check has passed, the name reassures them, age and sex drive the questions). Never the date of birth, NHS number, hospital number, mobile, email or record id.
- **Only a link token's SHA-256 is stored** (`login_links`); `episodes.patient_token` and `/p/:token` are gone. **Five wrong dates of birth lock a link; the right date resets the count.** **A resend makes a new link and revokes the episode's older ones** (which also clears a lock: PAT-05's "ask for a new one").
- **Invite links have no expiry date until C3** (**Rahul 13 Oct**, answers 3): they work while the episode exists; when C3 adds statuses, closing an episode ends its links.
- **Patient sessions are held in the database** (**Rahul 13 Oct**, answers 4, after test report 1 found a copied cookie still worked after Sign out): a `patient_sessions` row per right date of birth, tied to the link it was opened with. **Sign out, a resend (link revoked) or a lock (link locked) ends it at once**, and so do 30 minutes without a request (`last_seen_at`). One query per patient request checks and touches the row (Lambda-safe: no server memory). **The cookie (`poised_patient`, HttpOnly, SameSite=Strict, Path=/api/p) names only the session, signed; never the patient's record.**
- **An invite's link replaces the old one only once a message has gone** (S14 verification 1): the messages are built, the new link stored, the messages sent; then, with the episode's row locked (`FOR UPDATE`), the links older than the new one are revoked, or, if nothing went, the new link is. A failed Resend leaves the patient's working link and session alone, "Last sent" stays truthful, and two Resends at once leave only the newer link working.
- **`MESSAGING` must be set explicitly with `CONFIG_STRICT=true`** (S14 verification 1): `capture` serves the unauthenticated `GET /api/qa/messages`, so an AWS stack opts in to it, as to `DEV_LOGIN`; only local dev defaults to it.
- **Leaving the patient's form never drops the answer just typed** (S14 verification 2): **Sign out** and the on-behalf form's **← Episode** link first send any change waiting for its 600 ms autosave (and text answers count as typed: `textUpdateMode = "onTyping"`); if that save fails, Sign out's card shows "Not saved" and the on-behalf form stays with the save's message. Leaving the set any other way (Back) sends the waiting save instead of cancelling it.
- **Skipped for now: the patient's browser Back or a reload within ~0.6 s of typing loses that last input** (Rahul, 13 Oct, S14 answers 5, after test report 4's X-1: an edge case). Logged in pending.md with the fix (send the waiting save on `pagehide` with a `keepalive` fetch). S14 is verified and committed as is.
- **A save or Send refused because the session ended says what was lost** (S14 verification 1, "nothing may break silently"): the signed-out card names the section whose latest change wasn't saved, or says the answers weren't sent, and how to get back in.
- **Every `/api/p` response is `Cache-Control: no-store`**, and **the web app's page has `<meta name="referrer" content="no-referrer">`** so the link's code never leaves in a Referer (answers 4). F4's hosting should send a `Referrer-Policy: no-referrer` header too.
- **Messaging:** `Deps.SMS` and `Deps.Email` (`internal/messaging`): `Capture` stores in `sent_messages` (**`MESSAGING=capture`, the default and only mode until S17**), `Recorder` is the test fake. `GET /api/qa/messages?to=` and the **captured-messages page `/qa/messages`** (no sign-in) read them. Creating an episode sends the invite (SMS and/or email, the hospital's S10 templates); what went, and what didn't, is a General notes entry; a failed send never fails the Create. **The default invitation SMS is PAT-03's wording.** **PAT-11:** a text goes only to a number with the deployment's country calling code.
- **Staff can fill in the HQ on the patient's behalf** (PAT-09, `…/episodes/{eid}/patient-form`): the answers are the patient's, with `updated_by` = the staff member ("entered by", sticky per Question Set), and `episodes.patient_submitted_by`.
- **Details:** [S14 build notes](sessions/S14-patient-invites/build-notes.md).

### 13 Oct 2026: S13, patient records (builder, automatic run; tested 13 Oct, test report 3 all passed; verified 13 Oct, verification 1)

- **A patient record is the trust's, identified by `national_id` + `national_id_type`** (A-20, S13 answers 1), unique per trust when set. **The scheme is a per-deployment setting, `NATIONAL_ID_TYPE`** (`internal/config`; local default `nhs`), validated by `internal/nationalid` (only the NHS number's Modulus 11 check exists). `GET /api/config` returns `nationalId: {type, label}`, so screens say "NHS number" without a literal. **Reason:** A-20 (another country's identifier is a new validator plus config).
- **The hospital number is on the episode** (`episodes.hospital_number`, PAT-01b), each hospital's own label; the migration copied it from the patient.
- **Mobiles are stored in E.164**, read with `github.com/nyaruka/phonenumbers` (**kept, Rahul 13 Oct**, S13 answers 4: it keeps a new country config-only, where hand-written parsing would end up UK-only) (a number without a country code is the deployment's `PHONE_COUNTRY`). Only the number's length for its country is checked, so Ofcom's made-up 07700 900xxx numbers pass; "mobiles in the deployment's country only" (PAT-11) is checked when sending (S14). Emails are stored in lower case.
- **The New episode page (`…/episodes/new`) replaced the drawer.** It searches **this hospital's patients only** (at least one episode here; S13 answers 3): `GET …/patients?q=&dob=` by NHS number (exact), hospital number here (exact) or surname prefix, needing 3 characters (or 1 with a date of birth), at most 20 results (name, date of birth, NHS number, hospital number; never contacts). `POST …/patients` is gone: a patient is created with their first episode (`POST …/episodes` with `patientId` of this hospital's patient, or typed `patient` details).
- **Matching typed details** (`patient.Decide`; decisions 11 Oct and S13 answers 2): the NHS number with the same date of birth and surname → **linked silently**; with another date of birth or surname → **409 "mismatch"**, no stored values, nothing written. Without a known NHS number: the same first name, surname and date of birth **and** the same email or mobile → linked silently; the same name and date of birth otherwise → **409 "possible duplicate"**, which lists only **this hospital's** matches ("Use this record") and never one known only elsewhere, and **Create anyway** (`createAnyway`) makes a new record.
- **A silent link never edits the trust's record** (builder; **confirmed by Rahul 13 Oct**, S13 answers 5): what staff typed isn't written over its first name or contacts, and the pages show the stored record. **Reason:** a link mustn't silently change someone else's data; updating contacts is a separate, deliberate action (patients: S15, PAT-10).
- **After a link, the episode and patient pages show the trust's record** (builder, following the S08 answer that a hospital's staff see the trust's patient record): nothing is shown **before** the episode exists (Create returns only the episode id), and other hospitals' episodes and hospital numbers are never shown.
- **The Episode card's hospital number follows the chosen patient** (S13 test report 1): choosing a patient fills in their newest number here (from the search or from their patient page), and **Change patient** empties it, so a new patient's episode never carries the previous one's number.
- **A patient page, `…/patients/:pid`** (`GET …/patients/{pid}`, audited `patient.view`): the trust's record and the patient's **episodes at this hospital only**; 404 for a patient with no episode here.
- **Details:** [S13 build notes](sessions/S13-patient-records/build-notes.md).

### 13 Oct 2026: S11, the audit log (builder, automatic run; tested 13 Oct; verification 1's fixes retested, test report 3 all passed; verified 13 Oct, verification 2)

- **One `audit_log` row per request that reads or changes patient data**, list screens included (Rahul, 12 Oct, S11 answers 1): when, trust, hospital (NULL on a trust route), actor (`staff` or `patient`), user, action (e.g. `episode.view`), the **route pattern** (never the path: the patient's path holds their link's token), the HTTP status, and the IDs of the patients and episodes touched. **No patient values**; names are read when the log is read. List screens record every patient and episode they showed, so a patient filter also finds the lists that showed them.
- **Written by Go middleware, `audited(action, handler)`** (`internal/httpapi/audit.go`), inside `requireHospital`/`requireOrg`. Handlers note records with `auditOf(r).touched(patient, episode)`; `loadEpisode` and `loadByToken` do it for every per-episode and patient-link route. **A new patient-data route must be wrapped in `audited` in `router.go`**, with an integration case in `TestAuditRowsWritten`.
- **No row, no data:** the response is held until its row is written; if the insert fails the caller gets a 500 and nothing else (a change already saved stays saved). **Reason:** STF-06 says every view is logged; a view that can't be logged mustn't happen.
- **Only requests that passed the sign-in and membership checks are logged**, each with its status, so a probe for another hospital's episode shows as "Refused (404)". Refusals before that, and unknown patient links, write nothing (no trust the caller belongs to). **The patient's own link is logged** with the patient as actor.
- **Append-only** (NFR-06): a trigger refuses UPDATE, DELETE and TRUNCATE; foreign keys don't cascade. **Nothing is deleted for now** (S11 answers 2); removing rows past the trust's retention period will need a deliberate change to that trigger.
- **`GET /api/o/{oid}/audit`**, for whole-trust admins only (as Staff and Settings): filters `user` (an id, or `patient`), `from`/`to` (exact times; the web app sends the browser's midnights, so no zone is in the code), `patient` (name in either order, hospital number or record id, the trust's own patients only). The screen shows the newest 500; **`?format=csv`** exports every match, times in the browser's zone (`tz`, else the deployment's), with formula-looking cells defused. **Reading and exporting the log are logged too** (AUD-01: exports are recorded).
- **The Audit screen gets at most ten patients (named) and ten episode ids per row, plus their counts**; the CSV has them all (S11 verification 1). **Reason:** a list screen's row touches every patient in the trust, and naming them all made the page 4 MB at 337 patients, past Lambda's 6 MB response limit at about 600.
- **A patient-filtered view or export of the log is recorded against the patients it matched** (S11 verification 1); unfiltered views name nobody. **Reason:** looking up a patient's history reads their data (STF-06).
- **A refused export is logged as an export** ("Exported the audit log", with its 400), not as a view (S11 test report 2): the action comes from what was asked for, before the filters are checked.
- **In `make dev`, most views are logged twice**, because React's StrictMode runs each page's first load twice in development; production loads once.
- **Details:** [S11 build notes](sessions/S11-audit-log/build-notes.md).

### 13 Oct 2026: S10, the superadmin console, trusts, hospitals and settings (builder, automatic run; tested 13 Oct, test report 2 all passed; verification 1's fixes retested, test report 3 all passed; for re-verification)

- **Settings are one Go package, `internal/settings`:** `Defs` lists every ORG-04 setting (label, group, level, kind, default, limits); `Effective(platform, org, hospital)` resolves platform default → trust → hospital, taking a hospital's value only for settings a hospital may override. Values are stored as flat JSON objects in `orgs.settings` / `hospitals.settings`; a missing key is "not set here". **Every ORG-04 setting exists now** (Rahul, 12 Oct, S10 answers 2), though most are read only by later sessions.
  - **Defaults:** retention 8 years, cross-trust pre-fill off; assessments, files and observations on, PROMs and the three licensed instruments off; reminders on days 2 and 5, "not responding" on day 7; archive 30 days after discharge or after 6 months without activity; default texts for "what happens next" and four patient message templates (invitation and reminder, by text and email). Templates may use only `{{firstName}}`, `{{hospital}}`, `{{link}}`, and must include `{{link}}`.
  - **The web app has no list of settings:** the Settings screen draws one field per *kind* from the definitions Go sends.
- **The opt-ins (feature switches and licensed instruments) are set only by the platform admin** (PRM-08: they can be sold as add-ons). Trust admins see them on Settings; the API answers 403 if they try.
- **Only whole-trust admins see and change the trust's Settings** (`GET/PATCH /api/o/{oid}/settings`), as for Staff (S09).
- **The trust's contact (name, email, phone) is for Poised only** (`orgs.contact_*`). The patients' phone and email are the `contact.*` settings: the trust's default, which a hospital overrides. **The logo is the `branding.logoUrl` setting** (an http(s) address now; an upload in C8).
- **Suspension is enforced** (ORG-03): suspended trusts and hospitals drop out of every membership query, so a signed-in member's next request is 404 and `/api/me` leaves them out. **Sign-in is refused** (403) only when *all* of someone's memberships are suspended (S12's Cognito sign-in must check `SignInState` too). **The patient's link is closed** (403) while its trust or hospital is suspended. A suspended hospital's episodes leave "All hospitals". Nothing is deleted.
- **`GET /api/me` lists a trust that has no hospitals yet** (`hospitals: []`), and the start page explains that the platform admin adds them (S08 verification follow-up).
- **A platform admin can always sign in**, even when every trust they're staff of is suspended, so they can reach the console to re-activate it (S10 verification 1). The suspended trusts still stay out of their `/api/me` and every trust and hospital route. S12's Cognito sign-in must use the same check (`signInAllowed`).
- **A trust whose hospitals are all suspended is treated like a suspended trust for its staff:** it's left out of `/api/me`, and someone with no other usable access can't sign in. Only a trust with **no hospitals yet** is listed with `hospitals: []` (S10 verification 1).
- **The console's routes** (`/api/admin/…`, platform admins only, re-read on every request; others 403): list/create trusts (with the first admin invited by S09's invite, now `inviteTo` in `staff.go`), view a trust, add hospitals, `PATCH` a trust or hospital (status, opt-ins). Two small routes beyond the brief: `POST /api/admin/orgs/{oid}/admins` (re-invite an expired first admin, or recover a trust without a working admin) and `GET /api/admin/opt-ins` (the New trust form's checkboxes).
- **Codes** are 2–12 letters, digits or hyphens, kept in capitals, unique whatever their case; **hospital names** are unique within a trust; a hospital's **time zone** is chosen from the browser's IANA list, with the deployment's as the default (A-20: no zone in the code).
- **`cmd/platform-admin add <email> [name]`** / `make platform-admin EMAIL=…` adds a superadmin (A-21): the account if new (Cognito too), and the `platform_admins` row. Its test uses the apitest database harness, a stated exception to "integration tests are API → database only", because the command has no API.
- **The platform admin edits a trust's and a hospital's details at any time** (Rahul, 13 Oct, S10 answers 3): name, code, address, contact, logo address and time zone, through the same `PATCH` routes as status and opt-ins; only the fields sent change. The trust's own admins can't.
- **Opt-ins are the platform admin's only** (Rahul, 13 Oct, S10 answers 4: "opt-ins are what we sell; we set them from the agreed pricing and features"). Confirms the builder's choice.
- **`make seed` puts the demo trusts and hospitals back to how they were seeded** each run: active, their names, codes, contacts and addresses, the deployment's time zone and Poised's default settings, so every test round starts clean.
- **Switching hospital from the header keeps Staff and Settings wherever the user manages the target trust**, not only within the same trust (`switchPath`, S10 test report 1).
- **A change's message is shown where the change was made** (beside Save, or by the table changed), not at the top of a long page (S10 test report 1, F2).
- **Details:** [S10 build notes](sessions/S10-superadmin-console/build-notes.md).

### 13 Oct 2026: S09, staff invites, roles and removing access (builder, automatic run; verified 13 Oct)

- **Only a trust's whole-trust admins manage its staff** (`GET/POST/PATCH/DELETE /api/o/{oid}/staff…`). An admin of one hospital gets 403, like a clinician or super clinician; non-members get 404. **Reason:** Rahul's S08 rule that only whole-trust staff change trust-level things, and these are trust routes. The web app shows **Staff** only to them (`canManageStaff` in `apps/web/src/lib/auth.tsx`).
- ~~**One membership per person per trust.**~~ *Superseded 13 Oct by Rahul's answer below.*
- **A person can hold several memberships in one trust** (Rahul, 13 Oct, S09 answers; STF-02 "a role in each"): the whole trust and/or some of its hospitals, each with its own role, e.g. clinician at A1 and A2 but not A3.
  - **Routes:** `PATCH/DELETE /api/o/{oid}/staff/{uid}/access/{scope}` change or remove **one** membership, where `{scope}` is `trust` or a hospital id (PATCH body `{hospitalId|null, role}`, both required: it can move the membership and/or change its role; moving onto a scope they already hold is 409). `DELETE /api/o/{oid}/staff/{uid}` removes all of their memberships in that trust (the account and invite rows stay). The person-level PATCH was dropped.
  - **Inviting** someone already in the trust to a scope they don't hold **adds** that membership; to a scope they hold is 409 ("already has that access") unless their account is still waiting for its first sign-in, when it resends with that role.
  - **`GET …/staff`** returns one row per person with `access: [{hospitalId, hospitalName, role}]`, the whole trust first. The Staff screen shows each access on its own line with **Change** and **Remove**, and per person **Add access**, **Resend invite** and **Remove from trust**.
- **The membership is added when the invite is made. The 7-day expiry is the deadline for a new account's first sign-in**, matching Cognito's temporary password (AdminCreateUser, 7 days, RESEND when expired).
  - An account is "waiting" while it has an invite with no `accepted_at`. Its first sign-in accepts them all, unless every one has expired: then sign-in is **403 "Your invitation has expired…"** (the dev login now; S12's Cognito sign-in must call the same `checkInvites`).
  - Inviting an account that has signed in or was seeded only adds the membership (STF-02); its invite row is accepted at once and Cognito isn't called.
  - Inviting a waiting account again (from this trust or another) adds that access here (or takes the new role where they already hold it), keeps their other access, opens a new 7-day invite and calls `ResendInvite`: the Staff screen's **Resend invite**.
- **The last whole-trust admin can't be removed, demoted or narrowed to one hospital** (409). **Reason:** otherwise a trust could lock itself out until the superadmin (S10) steps in.
- **Emails are unique whatever their case** (migration 00007: `users_email_key` is a unique index on `lower(email)`); new accounts are stored in lower case. **Reason:** STF-02 (never a second account).
- **The invite takes an optional name;** without one, the email is the account's name until S12 knows better.
- **`httpapi.Deps` gained `Cognito` (`internal/cognito.Client`: `CreateUser`, `ResendInvite`) and `Clock` (`internal/clock`).** `cognito.Local` (the local servers) only logs; `cognito.Recorder` and `clock.Fake` are the test fakes, on every `apitest` client (`c.Cognito`, `c.Clock`). `Cognito` is required (a missing one panics at start-up); a nil `Clock` is the system clock. The session token's expiry still uses the system clock.
- **The Cognito call is the last step inside the invite's transaction,** so its failure rolls everything back (500).
- **Testing two signed-in people in one browser: `http://tester.localhost:3000`** reaches the same dev server and keeps its own cookie (found by the S09 tester). `http://[::1]:3000` works too, but Claude in Chrome refuses to open it; `127.0.0.1` doesn't answer (Vite listens on the IPv6 loopback only).
- **Dialogs that collect input start afresh each time they open** (S09 test report 1, F-1: the Invite dialog kept the last access and role, so the next person could get "Whole trust · Admin" unnoticed). The Staff page remounts its dialog's form with a `key` bumped on every open.
- **Rahul confirmed (13 Oct)** the other S09 choices: only whole-trust admins manage staff; access starts at the invite, with 7 days for the first sign-in; the last whole-trust admin is protected.
- **Details:** [S09 build notes](sessions/S09-staff-invites/build-notes.md).

### 12 Oct 2026: S08, trusts, hospitals and the role ladder (builder, automatic run; for verification)

- **Every staff route is under `/api/o/{oid}/h/{hid}/…`**, plus one trust route, `GET /api/o/{oid}/episodes` ("All hospitals", STF-03). The web app mirrors it: `/o/:orgId/h/:hospitalId/…`, and `/` is the hospital picker.
  - **Reason:** keeping the hospital in every URL meant the existing per-hospital queries and screens changed only by their prefix, and the middleware (`requireHospital`) can check the trust, the hospital and the role in one query. The plan's `/api/o/{oid}/…` shape is kept; C2's `/api/h/{hid}/…` routes become `/api/o/{oid}/h/{hid}/…`.
  - **Non-members get 404, not 403**, including a hospital-scoped user at another hospital of their trust, and someone whose membership was just removed. C1's "removed membership → 403" is a 404 here, matching NFR-04 (trusts and hospitals can't be probed).
- **Content belongs to the trust** (`questionnaires.org_id`, `option_lists.org_id`; `hospital_id` NULL = every hospital of the trust, as the brief says). Everything created so far, and everything created now, is the trust's. A row with a `hospital_id` is shown only at that hospital (tested), but nothing creates one yet. Draft names and option-list names are unique per trust.
  - ~~Open for Rahul: whether a *hospital-scoped* super clinician may edit the trust's shared questionnaires. As built, yes.~~ *Superseded 12 Oct by Rahul's answer below.*
- **Only whole-trust super clinicians and admins change the trust's questionnaires and option lists** (Rahul, 12 Oct, S08 answers 5; the recommendation had been "yes, as built"). A super clinician or admin of one hospital only can see them but not create, edit, publish, version or delete them; they can change only content that is their hospital's own (none is created yet). Deleting a draft takes its creator (with that right) or a whole-trust admin. The API checks it (`canChange` in `internal/httpapi/middleware.go`, from the caller's whole-trust role) and the web app mirrors it (`canChange` in `apps/web/src/lib/auth.tsx`). The seed adds **Sue Super**, a super clinician of Hospital A1 only, to show it.
- **Outputs are normalised when read** (`normalizeOutput` in `packages/clinical/src/properties.ts`, S08 round 3): a saved output may lack `codes` or its note's `category` (the e2e fixtures, older content), so every reader gets `codes: []` and `category: ""`. A note without a category is written `Note: …` in test cases. **Reason:** two test rounds found crashes from the same missing field in different places; one normaliser at the source fixes all readers.
- **Switching hospital from the header keeps the section** (Episodes stays Episodes, Questionnaires stays Questionnaires), after S08 test report 1. Elsewhere it goes to the role's start page.
- **Patient records belong to the trust** (`patients.org_id`; `hospital_id` dropped), as answered on 12 Oct. Each hospital lists all the trust's patient records when creating an episode, but sees only its own episodes. `hospital_number` stays on the patient until S13 moves it to the episode (PAT-01b).
- **Everyone in a hospital is at least a clinician**, so the episode routes have no role check, and their "wrong role" integration case became "a clinician at the trust's other hospital gets 404". Authoring, publishing and new versions need super clinician; deleting someone else's draft needs admin.
- **`hospitals.timezone` is nullable: NULL means the deployment's `DEFAULT_TIME_ZONE`** (A-20). `GET /api/me` returns each hospital's effective zone. Screens still show times in the browser's zone; reminders (S16) are the first to use it.
  - **Reason:** a migration can't know the deployment's zone, and the seed and migrations can't hold a zone literal (the A-20 guard test).
- **Migration 00006 carries existing data over:** each old hospital becomes the only hospital of a trust of its own; roles map as agreed (author/reviewer/publisher → super clinician, hospital_admin → admin, viewer/clinician → clinician, keeping the highest). **`make seed` then adopts** the trust holding Hospital A (renamed Hospital A1) as Trust A, and Hospital B's as Trust B, so HJE Full HQ and other local data stay. On a fresh database the trusts get fixed ids.
- **Seed users:** Ada Admin (Trust A admin), Alex Author (Trust A super clinician), Cara Clinician (A1 only), **Val Clinician** (A2 only; was Val Viewer), Bea Author (Trust B super clinician), **Max Multi** (clinician at A2 and Trust B's admin: the two-trust user), **Sam Superadmin** (`platform_admins`, no trust). Every seed run resets the demo users' memberships.
- **`GET /api/me` is `{user, platformAdmin, trusts: [{id, name, code, role, hospitals: [{id, name, timeZone, role}]}]}`.** A trust's `role` is the trust-wide one (null for hospital-only members); a hospital's is the higher of trust-wide and hospital roles. The role ladder is `internal/role` in Go and `atLeast` in `apps/web/src/lib/auth.tsx`.
- **The fake-querier route tests were deleted** (`httpapi_test.go`, `versions_test.go`, `validation_test.go`, `optionlists_test.go`, most of `episodes_test.go`): every route moved, and each already had API → database tests (testing rule 4). The input checks stay as unit tests.
- **Suspended trusts and hospitals** (`status`) are stored but not enforced; S10 does that.
- **Details:** [S08 build notes](sessions/S08-trusts-permissions/build-notes.md).

### 12 Oct 2026: S05, per-country config and the router split (builder, automatic run)

- **`internal/config` is the only home for per-country values** (A-20). It reads `APP_URL`, `DEFAULT_TIME_ZONE`, `DEFAULT_LOCALE` and `PHONE_COUNTRY` from the environment.
  - **Locally,** a missing one falls back to the UK.
  - **With `CONFIG_STRICT=true`** (AWS, from S06), a missing one stops start-up, and one error names them all. A bad value (an unknown zone, `en_GB`, `gb`) stops start-up in both modes.
  - **A guard test** (`internal/config/literals_test.go`) fails on a locale, IANA zone or AWS region literal in `apps/api`, `apps/web/src` or `packages/clinical/src` (tests and comments aside).
  - **Reason:** A-20 rule 2, enforced by a test rather than by review.
- **The time-zone database is built into the Go binary** (`time/tzdata`). **Reason:** a zone mustn't depend on what the host (Lambda's base image) has installed.
- **The dev login, `/api/dev/users` included, exists only when `DEV_LOGIN=true`.** `make dev` and `make staff-api` set it; nothing else does. **Reason:** F3 Step 1. The user list is part of the stub and lists everyone.
- **`GET /api/config` returns only `{locale, timeZone}`.** It needs no sign-in, and both routers serve it. **Reason:** the sign-in page and the patient's link need it first, and nothing secret or internal (web address, phone country) needs to reach the browser yet.
- **The web app loads `/api/config` once, before the first render** (`apps/web/src/main.tsx`, `src/lib/config.ts`), and every date goes through `src/lib/format.ts`.
  - If the fetch fails, it renders anyway with the browser's locale and logs a console error. The pages show their own "can't reach the server" errors.
  - **Times are still shown in the browser's time zone;** only the locale comes from config. S08's hospital time zones can decide which zone staff screens use.
- **`httpapi.Deps` is `{Pool, Config}`.** The token secret comes from the config, so there's one source for every setting.
- **Routers:** `NewStaffRouter` and `NewPatientRouter` run one each (`cmd/staff-api`, `cmd/patient-api`, through `internal/serve`). `NewRouter` mounts both for `make dev` (`cmd/api`) and the tests.
  - A route a router doesn't serve gets Go's plain-text 404. **Reason:** a JSON catch-all would also turn every wrong-method 405 into a 404.
- **`make patient-api` / `make staff-api` (`ADDR`, default `:8081`)** run one part alone beside `make dev`. Point a second Vite at it with `API_URL=http://localhost:8081 npm run dev -w apps/web -- --port 3001`.
- **Tests:** `apitest.NewWith(t, Options{Router, Env})` starts a test server with another router or environment, through the real `config.Load`.
- ~~**`CONFIG_STRICT=true` doesn't refuse `DEV_LOGIN=true`** (S05 verification). **Reason:** the QA environment on AWS runs strict *and* needs the dev login (F4); the guard against the dev login in Training or Production is F4's CDK synth check.~~ *Superseded 14 Oct by S12: QA signs staff in with Cognito, so strict now refuses DEV_LOGIN.*
- **Accepted after testing (report 1):** `DEFAULT_LOCALE` is checked only for shape, so a well-formed but unknown tag (`zz-ZZ`) starts, and the browser quietly uses its own locale. **Reason:** each country's value is set once, in `infra/countries.py` (S06), and Go has no list of locales the browser knows without adding a library. `/api/config` has no `Cache-Control` header; it's fetched once per full page load, which is cheap.

### 12 Oct 2026: answers for the local phase, asked up front by the conductor

Rahul answered every "Ask Rahul first" question for S05–S16 at the start of the first `/run-sessions` run. Each session's full answers are in its `answers.md` (`docs/sessions/<id>-*/answers.md`).
- **Git:** the 12 Oct planning-doc changes (skills, plans, briefs, S17) were **left uncommitted** at the start of the run. Each verifier commits its own session's files. Where a file holds both (e.g. `docs/plans/README.md`), the planning changes go into that session's docs commit.
- **S05: dates use the locale from `GET /api/config`,** not a hardcoded `en-GB`. **Reason:** it removes a UK assumption (A-20).
- **S08:**
  - **a time zone per hospital** (A-20). **Reason:** Canada, the US and Australia span several zones.
  - **Roles map** as author/reviewer/publisher → super clinician, hospital_admin → admin, viewer and clinician → clinician, with a reseed.
  - **The `dev` branch (`557be2c`) is ignored entirely.** Builders don't read or reuse it.
  - **A hospital-scoped user sees the trust's patient record** (name, date of birth, contacts), but only their own hospital's episodes.
- **S10:**
  - **a logo URL field now,** replaced by an upload in C8;
  - **all of ORG-04's settings are built now** (trust-only and hospital-overridable), even ones nothing reads until later sessions. This widens the brief at Rahul's choice; the recommendation was only what's used now.
- **S11: one audit row per request** that reads or changes patient data, list screens included. **Nothing is deleted for now.**
- **S13:**
  - **`national_id` + `national_id_type`** (A-20);
  - **patient search** as in C2's "New episode page" design (confirmed);
  - **duplicate matching uses email and mobile too.** On Create:
    - the same first name, surname, date of birth **and** the same email or mobile → link silently to the trust's record, as with a matching NHS number;
    - the same name and date of birth with a different or missing contact → a warning plus "Create anyway". Nothing about the other record is shown.
  - **Reason (Rahul):** in the incumbent, email is what has helped here, and the NHS number is often empty. Name + date of birth alone was rejected, because two people in a large trust can share both.
- **S14: before the date-of-birth check, the patient API returns only the hospital's name.** After the check, it returns the HQ without the date of birth or any identifier the patient didn't type. This settles the 8 Oct open item ([pending.md](pending.md) §3).
- **S15:**
  - **patients can update their own email or mobile now (PAT-10),** verified before use. This widens the brief at Rahul's choice.
  - **Sign-in links are rate-limited** to 3 per contact per 15 minutes and 10 per IP address per hour.
- **S16:**
  - **reminders are sent at any time,** with no 09:00–19:00 window;
  - **starting the HQ stops reminders,** not only submitting it.
  - Both are Rahul's choice over the recommendation.

### 12 Oct 2026: local first, and automatic session runs (session ae16fe2f)

- **Local only until the first demo or customer.** No AWS for now. **Reason:** nothing needs AWS until someone outside Rahul's laptop must use the app, and building locally lets automatic runs go without AWS stops.
  - **This reverses the 2 Oct order** (F3–F4 before C1). The cost is that the AWS hosting setup and its near-£0 idle cost are proven later.
  - **The local phase:** S04, S05, S08–S11, S13–S16.
  - **The AWS phase**, later and with Rahul there: S06 (it now also takes the Lambda half of F3), S07, S12, and a new **S17** (jobs Lambda, EventBridge schedule, real SES/SMS).
  - **Details:** [sessions/README.md](sessions/README.md).
- **Stand-ins for the AWS pieces:**
  - **email and SMS:** capture mode, with a captured-messages page;
  - **staff sign-in:** the dev login instead of Cognito;
  - **files:** MinIO instead of S3 (C8);
  - **scheduled jobs:** a timer in the local server instead of EventBridge.
- **Write the code as if it already runs on Lambda,** so the AWS phase is wiring, not a rewrite:
  - jobs are one function, `jobs.RunDue(ctx, now)`;
  - no state in the server's memory;
  - no reliance on local disk;
  - files go through presigned URLs.
- **Docker is unchanged:** only Postgres runs in it (later MinIO too). The API, web and jobs run on the Mac with `make dev`. Rahul asked about dockerising everything. It was kept as is (the 1 Oct decision) for fast feedback, and because Docker is unreliable on his Mac (§4).
- **Automatic runs: a conductor session (`/run-sessions`).**
  - **What it does:** for each session in the run order, it runs:
    1. a builder agent;
    2. a new tester agent for every test round, with the reports going back to the same builder;
    3. a verifier agent;
    4. a local commit.
  - **What it handles:**
    - it asks every brief's questions up front;
    - it restarts `make dev` before each test round, because `make dev` doesn't reload Go;
    - it sends phone notifications;
    - it resumes from `docs/sessions/run-state.md`.
  - **Rahul's standing permission (12 Oct):** the verifier commits locally to **`master`** after a Verified verdict, without asking. **Nothing is ever pushed.**
  - **Limits:** at most **4 test reports** and **2 verifications** per session, then it stops and asks.
  - **Testers mark cases their tools can't do** (blocking a network request, the real print dialog) as **hand checks for Rahul**, not failures.
  - **Checked 12 Oct:** an agent started by a session can use Claude in Chrome. It needs `createIfEmpty: true` to get its own tab group.
  - **Rejected:** a scripted workflow, and a shell loop over `claude -p`. Neither can pause to ask Rahul a question and carry on.

### 12 Oct 2026: S04, the frontend on React + Vite

- **The UI is renamed from the old name to "Poised" in S04** (Rahul, 12 Oct). This covers the browser tab title, the hospital header, the login page title (the old name becomes "Poised") and the patient page header. Code comments that cite the incumbent as layout evidence stay.
  - **Reason:** every layout is rewritten in S04 anyway, and it's the only visible change S04 allows.
  - **Details:** [S04 brief](sessions/S04-react-vite/brief.md).
- **Node 22.22 or newer is required** (was 20.9; S04 builder). React Router 8 needs it.
  - **Reason:** React Router 8 is the current major version, and Node 20 has been end-of-life since April 2026. F4's build images and the `evaluate` Lambda should use Node 22 or newer.
- **Each page's code loads when the page is first opened** (React Router's route `lazy`, in `apps/web/src/routes.tsx`; S04 builder).
  - **Reason:** with plain imports, any first page load downloaded every screen (2.3 MB of JavaScript), the editor included on the patient's phone. Next.js had split the code per page by itself. Now the first load is 0.53 MB of JavaScript.
- **Pages are keyed by their URL IDs, and the hospital layout by the hospital** (S04 builder).
  - **Reason:** Next.js started a page afresh when the ID in its URL changed; React Router reuses the component. Keying keeps the old behaviour, so one episode's or hospital's state never lingers on the next.
- **A page whose code can't be downloaded reloads the app once, then shows "Something went wrong" with Reload** (S04, after verification 1). The root route's error screen is `apps/web/src/pages/RouteError.tsx`; a `sessionStorage` timestamp allows at most one automatic reload in 10 seconds.
  - **Reason:** with per-page loading, anyone with the app open across a deploy gets a missing file on the next page they open (and a patient on a patchy signal can too). Next.js recovered with a full page load; React Router's default is a raw error screen with no way out.
- **Details:** [S04 build notes](sessions/S04-react-vite/build-notes.md).

### 12 Oct 2026: S03, the end-to-end journeys (builder's choices, for verification)

- **The patient's journey (J3) runs in WebKit with Playwright's iPhone 13 profile,** not in Chromium at phone size.
  - **Reason:** it's the closer match to a real iPhone (Safari's engine), and F1 already installs WebKit.
  - The `phone` fixture keeps its trace through pytest-playwright's private `_artifacts_recorder`; check it when upgrading pytest-playwright.
- **Each journey sets up its own data through the API** (`e2e/support.py`), as the seeded users, with unique names. J4 sends the patient's answers through the patient-link API; J5 also validates and completes the review through the API. Nothing is deleted.
  - **Reason:** journeys must not depend on each other or their order (testing rule 5), and the UI is used only for the journey being tested.
- **No `data-testid` was added.** Everything is found by role or label, apart from three ids that already existed, listed in `e2e/README.md`.
- **`make e2e ARGS=…` passes options to pytest** (e.g. `ARGS=--headed`), and **`make e2e-setup PYTHON=…`** names the Python 3.12+ to use (default `python3.12`).
- **Details:** [S03 build notes](sessions/S03-e2e-journeys/build-notes.md), [e2e/README.md](../e2e/README.md).

### 11 Oct 2026: finding the patient when creating an episode

- **Matching a patient to an existing record is silent, and shows nothing from other hospitals** (Rahul, 11 Oct).
  - **What staff see:** they type the patient's details from the referral, and the server links the episode to the trust's existing record when the identifiers agree.
  - **What staff never see:**
    - which other hospital has seen the patient;
    - the patient's episodes, answers or notes there;
    - any stored value they didn't type.
  - **A mismatch** (same NHS number, different date of birth or name) is reported without showing the stored values.
  - **Other trusts' records are never searched.**
  - **Reason:** privacy, and no sharing of information Hospital A doesn't need. Claude's first proposal showed "last seen at Hospital B" and earlier episodes, and Rahul rejected it.
- **The New episode page is rebuilt in C2,** as a dedicated page. It replaces the drawer and its Existing/New patient tabs (Rahul, 11 Oct).
  - **Reason:** silent matching needs C2's trust-wide records keyed by NHS number.
  - **Staff can search patients, but only their own hospital's** (Rahul, 11 Oct). C2's plan had a trust-wide search, and it's now hospital-scoped.
  - **The search must work with millions of patients:** today's dropdown doesn't.
  - **The design** is in [plans/c2-patients-signin.md](plans/c2-patients-signin.md): server-side search as you type, at most 20 results, and a precise term required. Its details (what counts as "this hospital's patient", the search terms) are Claude's proposal, for C2's build session to confirm with Rahul.


- **The patient's autosave gets the same revision check as the clinician's** (`PUT /api/p/{token}/answers/{cid}`).
  - **Reason:** a patient with the link open on a phone and a laptop loses answers in the same way when the older one saves last, and just as silently. It's the same pattern on the same table.
  - **Details:** [S02a brief](sessions/S02a-answer-revisions/brief.md).
- **S02a also fixes the episode PATCH status write-back.**
  - **The fix:** the UPDATE succeeds only while the status is still the one loaded; otherwise the PATCH gets a 409.
  - **Its own commit,** test-first.
  - **Reason:** it's small and the same idea. It was found by S02's verification (pending.md §1a, and S02's verify.md item 2).
- **S03's J4 gains a two-tab step:** a stale tab's save gets the "changed somewhere else" message.
  - **Reason:** the integration tests prove the 409, but only an end-to-end test can guard the browser's message and Reload.
- **Any 409 on an answers save stops autosaving and offers Reload,** not only a stale revision. That includes "the review is already complete" and the patient's "already sent" (S02a builder; it passed manual testing on 9 Oct).
  - **Reason:** in every case the page is out of date, and Reload shows the truth. The refused change stays on screen, locked, until Reload, so nothing typed vanishes before the user reads why.
- **The patient page doesn't save a section the server already has** (S02a builder, 9 Oct; corrected after S02a's verification the same day). Leaving a section used to save it every time.
  - **"Already has" means what the server accepted, not the page's own copy.** The first version compared with the page's copy, so a failed save was never sent again (verify.md finding 1).
  - **The section list, "Complete" and "Send my answers" go by what the server has,** so a section whose save failed can't look complete or be sent.
  - **And a section with a change that hasn't saved isn't complete either** (corrected again after verification 2, 11 Oct). A section that was already complete, then changed while offline, used to stay Complete, and the HQ could be sent without the change. `isSetReadyToSend` in `packages/clinical` decides it. After a failed save, the section's label says "Not saved".
  - **Reason:** with the revision check, a tab left open on another device would otherwise get the conflict message for doing nothing. Before S02a, it would have overwritten newer answers with stale ones. The clinician page already worked this way.
  - **Details:** [S02a build notes](sessions/S02a-answer-revisions/build-notes.md).
- **Race tests make the race happen in a fixed order with `meanwhile`** (`apps/api/internal/httpapi/cast_integration_test.go`).
  - **How:** the conflicting change holds a row lock until `pg_stat_activity` shows the request waiting, and only then commits.
  - **Reason:** two concurrent HTTP requests would pass or fail by timing. This way it goes through the real API and database, with no fakes.
  - **Use it for any future "between the read and the write" bug.**

### 8 Oct 2026: S02, what the workflow integration tests prove

- **Lost updates on answers are prevented by a revision check, not a lock** (Rahul, 8 Oct). Each answers row gets a revision. A save must send the revision it started from; if someone saved in between, it's a 409 and nothing is written. This is the same pattern as Question Set saves.
  - **Reason:** S02's manual test found that a stale tab's Validate silently overwrote a newer answer (X-1). A per-person lock wouldn't stop one person's two tabs. A lock needs expiry, a "still here" signal and a take-over button, and still needs this check after a lock expires.
  - **Built in session S02a** ([brief](sessions/S02a-answer-revisions/brief.md)), after S02 and before S03.

- **The integration tests prove what Go enforces, not the two workflow rules the browser applies** (Rahul, 8 Oct). Two of F1's "must also cover" items are browser rules:
  - **"Complete review is refused until every set is validated."** The server refuses only when the episode isn't Ready for review. The browser enables the button once every *visible* set is validated, and visibility depends on Question Set conditions, which Go can't evaluate (the key rule in CLAUDE.md).
  - **"The first write copies the patient's answers."** The browser pre-fills the clinician view from the patient's row and sends the full data. The server stores a separate clinician row and leaves the patient's row as it was.
  - **Reason:** S02 makes no behaviour changes beyond the two its brief asked for (the episode PATCH `null` fix and the recover middleware), and the server can't know which sets are visible without evaluating expressions. S03's J4 covers both rules end-to-end. The gap is in [pending.md](pending.md) §1a, for C6 (the `evaluate` Lambda can work out the visible sets).

### 8 Oct 2026: S01, the integration test harness (session a8c9de3a)

All of these are detailed in [sessions/S01-integration-harness/build-notes.md](sessions/S01-integration-harness/build-notes.md) and the F1 Status section.

- **Integration tests: one Postgres database per test, copied from a template.** `apitest.Main` migrates `sj_test_template` with goose (as a library); `apitest.New(t)` copies it (about 0.3 s) and serves the real router over `httptest`.
  - **Reason:** every test starts clean and tests run in parallel, with no clean-up code in the tests.
- **A Postgres advisory lock is held while a test package runs, and leftover `sj_test_…` test databases are dropped at the start.**
  - **Reason:** `go test ./...` runs packages in parallel, and two packages would otherwise drop each other's template. Killed runs would otherwise leave databases filling the disk.
- **`make test-integration` uses `-count=1`, and `make lint` vets with `-tags integration`.**
  - **Reason:** Go's test cache doesn't know the database changed, and untagged vet would skip the integration test files.
- **How the five cases apply where one can't exist** (agreed by Rahul, 8 Oct):
  - for a read route, the "wrong role" case asserts that a viewer **can** read;
  - for codes and categories (reference data shared by every hospital), the "another hospital" case asserts a user in another hospital gets the same results;
  - every refusal also asserts that nothing was written.
- **A bug found by the integration tests is fixed with a unit test of a small pure function, and committed before the route tests.**
  - **Reason:** each commit must pass on its own, and the route-test commit already contains the case that exposed the bug.
  - S01 found two bugs: a 500 on `?page=99999999999` in the questionnaire list, and a crash on a `null` field in a chapter PATCH.

### 7–8 Oct 2026: countries, superadmin and the platform features (session ae16fe2f)

- **Rahul's six platform features are already planned in C1 and C2.** No new plan was needed. The six:
  - superadmin;
  - hospital onboarding;
  - staff and patient invites;
  - permissions;
  - staff in several hospitals;
  - patients in several hospitals.
- **Superadmin: only Rahul, created by hand, with no screen** (A-21).
  - Reason: there will be one or very few, and no screen means nothing to build or attack.
  - A small command creates the Cognito user and the `platform_admins` row. It's run once per environment and country ([plans/c1-trusts-staff.md](plans/c1-trusts-staff.md)).
- **Residency rules usually cover where data is processed and accessed, not only where it's stored.** Rahul expected storage only. Research on 8 Oct covered Canada, the US, the EU, Ireland, Australia and New Zealand ([report](../reports/Health%20data%20residency%20storage%20vs%20processing.md); notes in `research_notes/`). It found:
  - **Storage-only rules exist only in Texas, Florida and France (HDS).**
  - **The EU's GDPR counts remote access from the UK as a transfer.** It's allowed only because of the UK adequacy decision, renewed until Dec 2031.
  - **Some rules cover access, use or processing:** Nova Scotia, Quebec and Alberta; Germany, Italy and Spain; Australian state rules.
  - **Hospital tenders and contracts often add support location,** e.g. an HSE tender's "supported from within the EEA" and Texas Medicaid contracts.
  - **London is lawful with the right contract** under US federal law (HIPAA), and in Ontario, New Zealand and Irish law.
  - **It's research, not legal advice.** A local lawyer confirms the rules before the first contract in a new country.
- **One deployment per country, each at its own web address. Only the UK runs now, and nothing is hardcoded for the UK** (A-20, which supersedes A-19).
  - **How we got there:**
    - 7 Oct: Claude proposed a copy per country. Rahul rejected it as too much to maintain alone.
    - Rahul then proposed code in London with each country's database in that country (the same idea as A-19). The research showed that only helps in Texas, Florida and France.
    - 8 Oct: Rahul agreed to build only the UK, but to make a new country quick to spin up, at its own subdomain.
  - **Why it's little extra maintenance:** one codebase, one build and one pipeline. A fix is one push that deploys every country, and every country's alarms go to one email.
  - **Details:** [saas-requirements.md §13 "Countries"](saas-requirements.md). The groundwork is in F3 (`internal/config`) and in F4:
    - `infra/countries.py`;
    - a CDK test that synthesises a made-up second country;
    - the "add a country" runbook.
  - **Dropped from A-19:**
    - `data_region`;
    - `store.ForOrg`;
    - the control-plane / data-cell split;
    - `account_records`;
    - the jobs loop over regions.
- **The plans are built in work sessions** (8 Oct; [sessions/README.md](sessions/README.md)). F1–F4 and C1–C2 are split into S01–S16, each with a `brief.md`.
  - **Three roles, each in its own Claude session,** run by project skills:
    - **builder** (`/build-session`): builds test-first and writes `test-plan.md`;
    - **tester** (`/manual-test`): a new session for every round, which drives Chrome like a manual tester and never edits code;
    - **verifier** (`/verify-session`): reviews, runs everything, spot-checks, and offers the commits.
  - **The loop:** the builder fixes what each test report finds, until a report says everything passed.
  - **Nothing is committed until verified,** and a session starts only when the one before it is verified and committed.
  - **Reason:** Rahul wants to start each session himself, with independent manual testing in a real browser, and a check before anything is committed.
- ***Proposed, not agreed*** (to confirm when C1 and C2 start):
  - `national_id` + type instead of an `nhs_number` column; *agreed 12 Oct (S13 answers)*;
  - a time zone per hospital; *agreed 12 Oct (S08 answers)*;
  - dates formatted in the deployment's locale; *agreed 12 Oct (S05 answers)*;
  - the UK on `uk.<domain>` from day one;
  - one AWS account per environment, with a region per country;
  - support tooling that never shows patient answers, plus customer-approved break-glass access (OPS-06).

### 2 Oct 2026: SaaS productionisation (session bf5d9b3a)

All of these are in [saas-requirements.md](saas-requirements.md) §0 and [plans/](plans/README.md).

- **Name: Poised** (was sj-demo). *Superseded 20 Oct: renamed Preopsis (see 20 Oct).*
  - Go module `github.com/rlamba89/poised/apps/api`; npm packages `poised`, `@poised/web`, `@poised/clinical`.
  - **Kept on purpose:** the CSS prefix `sj-`, the local Postgres user and database `sj`, and the docker volume `sj-demo_dbdata`, pinned in `docker-compose.yml` so local data survives the folder rename.
- **Repo:** private, at `github.com/rlamba89/poised`, branch `master`. On 2 Oct the history was rewritten with `git filter-repo` to remove the incumbent's organisation name, and the GitHub repo was deleted and recreated so no old commits remain. On 22 Oct it was rewritten again to remove the product's name.
- **Fresh app, no historic data:** schemas can be reshaped freely before go-live.
- **Tenancy:** a **trust** (the tenant and data controller) has many **hospitals**. Settings resolve platform default → trust → hospital, and each setting says whether a hospital can override it.
- **Roles:** `clinician < super_clinician < admin`, plus our platform admin. **Everyone in a hospital sees all its data**; roles only limit what people can do.
- **Patients:** one login account, plus a record per trust identified by **NHS number** (optional, check-digit validated). The hospital number is only a label on the episode.
- **Patient sign-in:** a **magic link (SMS or email) + date of birth**, built in Go.
  - Reason: Cognito has no native magic link; codes are friction for older patients.
  - A **Continue** button before using the link stops email scanners from consuming it.
- **Staff sign-in:** **Cognito** (managed login + TOTP MFA). After the OAuth callback, Go sets our own session cookie.
- **Pre-fill:** any question answered in an earlier HQ is pre-filled in a new one, **with no expiry**.
  - Matching is by **question key** (the SurveyJS `name` / stable ID). *Widened 22 Oct: by key, then by exact wording (§2, 22 Oct).*
  - There's a "Same as…" link, "Always ask fresh", and locked option values.
- **ASA:** platform defaults per code, which can be changed per HQ.
  - **Rules per HQ.** Defaults: ASA 1–2 with no red flags → **auto-triage straight to Ready for admission**; ASA ≥ 3 → a "Book anaesthetic review" task + flag + anaesthetist ASA required.
  - The clinician can change the target per HQ.
- **D-1: server-side evaluation.** A Node **`evaluate` Lambda** runs `packages/clinical` on patient submit.
  - Reason: a patient's browser isn't trusted, and Go can't evaluate SurveyJS.
  - This **supersedes, for this one case, the 30 Sep rule "no Node service"**.
- **"Capture" Group/Section:** filled in only by clinicians.
- **More than one Clinical Summary per page:** the summary becomes an element. This needs SGN-04 in requirements.md updated.
- **Videos:** hosted on **YouTube or Vimeo** (no-cookie / `dnt=1` embeds), never by us. There's a built-in suggested library plus author links.
- **Assessments:** **STOP-Bang first**, then DASI, PRISMA-7, Apfel, MUST, Falls, VTE, Rockwood. **MMSE is deferred** (copyrighted).
- **PROMs:** a baseline plus follow-ups after discharge.
  - **EQ-5D-5L and the Oxford Hip/Knee Scores are built now.** Each is **opt-in per trust, with a per-hospital override**, and the licence cost may become a customer add-on. Licensors aren't contacted for now.
  - Free recovery check-ins are always available.
- **Files:** **no virus scanning**, a 25 MB limit, and no patient–clinician messaging yet.
- **Out of scope:** EPR integration.
- **Cloud: AWS eu-west-2**, serverless, **near-£0 when idle before the first customer**.
  - Other **data** regions per trust must be possible later (code stays in London). The groundwork is a `data_region` per trust, `store.ForOrg`, no cross-trust SQL joins of patient data, and no patient data in control-plane tables.
  - *Superseded 8 Oct by one deployment per country (A-20); see the 7–8 Oct entry above.*
- **Compute: 4 Lambdas from one Go codebase** (`staff-api`, `patient-api`, `jobs`) + the Node `evaluate`.
  - Reason: separate permissions and blast radius for the public patient API, without one function per route.
  - Lambda Managed Instances, durable functions and SnapStart were checked; none is needed.
- **Database: plain Postgres.** **Neon** (AWS London, free, scales to zero) until the first customer, then **RDS**.
  - RDS rejected for now: it can't scale to zero, and it needs a VPC + NAT.
  - Aurora DSQL rejected: no extensions (we use `pg_trgm`).
  - Aurora Serverless v2 rejected: about 15 s to wake.
  - Supabase rejected: the free tier pauses after 7 days.
- **Frontend: a React + Vite + React Router SPA on Amplify Hosting** (plan F2).
  - Reason: Amplify supports Next.js ≤ 15 (we're on 16.3), and the app uses no server features.
  - Vercel rejected: $20/seat, no commercial use on Hobby, a second platform, and its proxy would carry patient data.
  - A static S3 export was rejected by Rahul.
- **Infrastructure: AWS CDK in Python + CodePipeline** (CDK Pipelines). No SAM.
- **Messaging:** SES (email) and AWS End User Messaging (SMS).
- **Testing:**
  - TDD unit tests;
  - **integration tests (preferred) = API → real database only**;
  - **Playwright + Python e2e in `e2e/`**, in this repo, not a separate one.
  - Full rules in [plans/README.md](plans/README.md).
- **Plan structure:** foundation first (F1 tests → F2 React/Vite → F3 Lambda entry points → F4 AWS + pipeline), then core C1–C10, each testable before the next starts.

### 1–2 Oct 2026: the episode workflow (sessions dc7d83c0, 74e1fcfc, 57814611)

Mostly recorded in [plan-workflow.md](plan-workflow.md).

- **Build the whole workflow thinly end to end first**; platform work (auth, hospitals, accounts) later. Reason: see the core flow working first.
- **Answers per actor:** the patient's copy is frozen at submit, and the clinician's working copy is final. The POA Summary marks changed answers, and the patient's originals are kept for audit.
- **Triage is only a status** (Ready for review), set on submit.
- **The patient link `/p/<token>` needs no login, and Publish is one click.** Validation is per Question Set; statuses follow the incumbent's list. (The link is superseded by magic-link sign-in in plan C2.)
- **Go strips clinician-only content** for patients, and disclosures are computed in the browser at view time.
- **Clinician answers are locked** after the review is complete. There's no re-open yet (plan C3 adds it).
- **Clinician boxes sit in the row of the patient question above them**, with no "linked to" setting. Same as the incumbent's layout.
- **Only Postgres runs in Docker.** The API and web run with `make dev`, so there's no image rebuilding.
- **The manual test plan** ([manual-test-plan.md](manual-test-plan.md), MT-01…MT-68) was fully automated headless (63/63 passing on 1 Oct) using gitignored scripts in `spikes/mt/`. Plan F1 replaces these with committed Playwright tests.

### 1 Oct 2026: incumbent-style redesign and SurveyJS features (session 5ac0c8c2)

Mostly recorded in [plan-redesign.md](plan-redesign.md), [plan-features.md](plan-features.md) and [incumbent-ui-notes.md](incumbent-ui-notes.md).

- **Our own editor; the SurveyJS Creator is dropped.**
  - Reasons: no Creator licence is needed, and content is edited as plain JSON data. survey-core still runs the preview, runtime and outputs.
  - **Copy the incumbent's layout and behaviour, not its styling**: the Structure tree, card canvas, Settings / Disclosures / Logic panel, and incumbent terms (Question Set, Page, Disclosures).
- **Our output model stays richer than the incumbent's:** review flags + ASA on disclosures. The incumbent has no flags, and its ASA is disabled.
- **"Create the HQ" means rebuilding the HJE Full HQ content** in our tool, from screenshots ([hje-full-hq-content.md](hje-full-hq-content.md)).
  - Sets 1–4 were imported.
  - All 11 sets + disclosures need `tools/export-hq.js` run in the incumbent's authoring tool, which hasn't been done yet.
- **The logic Builder uses SurveyJS's own `ConditionsParser`**, so the Builder and the Code tab always agree. It grew from a single Is / Is not condition to All / Any / None / Not all at any depth.
- **Other choices:**
  - **Test cases** are stored in the chapter JSON (`testCases`) and stripped for patients.
  - **Question Set conditions** use `chapterVisibleIf`.
  - **Skip rules** record their `page`.
  - `calcKind` and `textFormat` are stored explicitly, not inferred.
- **Saved option lists are copied into a question, not linked.** This protects stable IDs. *Still awaiting Rahul's confirmation.*
- **Editor implementation choices** (not in other docs):
  - **Selection is kept in the URL** (`?set=&page=`).
  - **Autosave** is debounced at 600 ms, with the 409 revision guard. Undo is a list of snapshots.
  - **The Structure tree stays mounted** (hidden) while Settings is open; unmounting it collapsed the expanded sets.
  - **Mantine multi-selects are `searchable`**, otherwise the input can't be clicked.
  - **Decide behaviour by question type, not `inputType`.**
- **Clinician two-column layout:**
  - The grid sits on `.sd-page__content`, and `.sd-row:has(.sj-clinician)` goes in column 2.
  - The clinical summary panel `cs_<page>` (`cs_notes_` / `cs_comments_`) is injected in the clinician view only.

### 30 Sep 2026: first plan and demo slice (sessions 684ab54e, 28534d3d)

Recorded in [plan.md](plan.md) (§2, §6, §10, §12) and the README's "Deviations" table.

- **Stack:**
  - a monorepo (npm workspaces + a Go module + a Makefile);
  - Next.js + Mantine (*superseded 2 Oct by React + Vite*);
  - Go stdlib `net/http`, REST;
  - Postgres with sqlc + goose;
  - **SurveyJS 3.1.2, pinned exactly**;
  - one SurveyJS JSON per chapter;
  - stub login, local only.
- **No Go port of SurveyJS, and no Go evaluation of expressions.**
  - Reason: no non-JS evaluator exists, and a port would drift silently, which is a clinical safety risk.
  - Also rejected: a restricted condition subset, and **goja** (JS inside Go, untried).
- **Trust the frontend** (Rahul's choice over a recommended Node evaluate service).
  - All answer replay (preview, test cases, outputs) runs in the browser through `packages/clinical`, and Go stores and enforces.
  - Test results are tied to the chapter revision.
  - *Partly superseded 2 Oct (D-1): patient submits are evaluated server-side.*
- **Use real incumbent codes; never invent codes.** Seed CSVs are in `apps/api/db/seed/codes/` (276 codes; 21 have no category; ICD-10 `F17.1` text is broken at source).
- **Stable IDs are the SurveyJS `name` / option `value`.** They're letters only (`q_kfbwpx`, `o_tqmd`) and never change once published.
- **Content rules:** clinician-only content is stripped for patients; our own preview page; `/api/*` is proxied; Tabler icons; drafts are hard-deleted.
- **sqlc settings** (`apps/api/db/sqlc.yaml`):
  - `uuid.UUID` and `time.Time` overrides, camelCase JSON tags, pointers for nulls;
  - `emit_interface`, so handlers take `db.Querier` (the fake-querier unit tests use it; new tests use the real database);
  - search arguments joined with `||` need a `::text` cast.
- **Two test HQs** were built for manual testing: "General pre-operative assessment" and "Comprehensive pre-operative assessment (test)" (7 chapters, 76 questions, every option type).
  - They exist **only in Rahul's local database**. The generator script wasn't saved.


## 3. SurveyJS findings (v3.1.2, exact pin)

- **`clinicalOutputs`:** held in memory in an immutable `{ items }` box behind `outputsOf` / `setOutputs`; the JSON stays a plain array. Reason: `setPropertyValue` overwrites existing arrays in place, which breaks redo. (README "Deviations")
- **`useElementTitles: true`** replaces the deprecated `showTitlesInExpressions`.
- **"None of these"** is a normal option with `isExclusive: true`. The built-in None, Other, Don't know and Refuse options don't save custom properties, so outputs on them would be lost.
- **"Yes / No"** is a Select One preset, not a boolean.
- **Question CSS:** `cssClasses.mainRoot` is the question's box; `root` is only the inner input.
- **`model.validate(false)`** checks required answers without showing errors.
- **`showCompleteButton`** (v3) lets the clinician's Validate button replace Complete.
- **Warning-type validators** (`notificationType: "warning"`) don't block Next.
- **No non-JS evaluator exists**, so Go must never evaluate expressions (exception: the `evaluate` Lambda, D-1).
- **`setPropertyValue`** copies into an existing array, which breaks redo. Untyped custom properties default to `string`, so edits within 1 s merge into one undo step.
- **`question.isVisible` stays true inside a hidden panel.** Use `getAllQuestions(true)`.
- **There's no "render if"**; use visible-if, enable-if and required-if.
- **"Is not" / `notcontains` is true when the question is unanswered.** The incumbent behaves the same way.
- **A matrix has `choices`.** Date lists, Medication and Admissions were mistaken for choice questions, and their disclosures never fired (fixed). Branch on the question type.
- **Dispose the old survey model** when switching view or language. Otherwise ResizeObserver throws "scrollWidth" errors (seen with Rating and File upload).
- **Unit tests must call the custom-property registration first**, or every custom flag is reported as unknown.
- **A skip rule fires only once the page's required questions are answered.**

## 4. Environment gotchas

- **Docker keeps dying / the disk fills up (on Rahul's Mac).**
  - **Cause:** Microsoft Defender's network extension (`com.microsoft.wdav.netext.systemextension`) isn't pre-approved by the Mac's management (Jamf). It loops, copying 121 MB into `/Library/SystemExtensions/.staging` every 12–20 s (~29 GB/hour).
  - **Check:** `df -h /System/Volumes/Data`, `ls /Library/SystemExtensions/.staging | wc -l`, `mdatp health`, `systemextensionsctl list`.
  - **Real fix:** allow the extension and give Defender Full Disk Access in System Settings, or IT approves it.
  - **Temporary:** restart, or clear `.staging` with `sudo` (keep the newest copy).
  - Quit Docker before the disk fills. **Never factory-reset Docker**, because it wipes other projects' data.
  - If Docker hangs after a disk-full dialog, force-kill its backend process.
- **Postgres won't start after a full disk** ("bogus data in lock file"):
  ```sh
  docker run --rm -v sj-demo_dbdata:/data alpine rm /data/postmaster.pid
  docker compose up -d --force-recreate db
  ```
- **Long-running dev servers started by Claude** are stopped after their time limit (max 2 h). Run `make dev` in your own terminal for long sessions.
- **Headless browser checks** (`spikes/ui-*.ts`, gitignored, so they won't exist on a new machine; *superseded 12 Oct 2026 by the committed journeys in `e2e/`*):
  - They use Playwright's `chrome-headless-shell` from `~/Library/Caches/ms-playwright/`, via `CHROME_PATH`.
  - A visible run needs `HEADED=1 SLOWMO=100 HEADED_CHROME_PATH=<full Chromium>`.
  - Use an explicit browser context to open several tabs.
  - Plan F1 replaced all this with `e2e/` (S03, 12 Oct).
- **Driving the incumbent's training site from a script:**
  - It redirects `/author` URLs to `/hospitals` until a hospital is picked.
  - Clicking from VS Code needs "Allow JavaScript from Apple Events" in Chrome.
  - When capturing, screenshot only the Chrome window (`screencapture -l <CGWindowID>`): other apps are open.
- **zsh treats `===` specially**, so avoid it in `echo` separators.
- **npm:** a workspace package needs a `version` field before npm will link it.
- **Go 1.26** is required because sqlc (run as a Go tool) needs it.
- **Docker Desktop hangs:** `docker info` and CLI calls can hang, including on the `docker-ai` plugin. Force-quitting the backend fixes it, but it also stops other projects' containers, so ask first.
- **Invalid HTML in React** (e.g. a Mantine `<Badge>` (a div) inside `<Text>` (a p)) shows as a console error. Check the console after UI changes. (Under Next.js it also showed in the dev overlay's "Issues"; *Next.js removed 12 Oct, S04*.)
- **Playwright selectors:** Mantine `Select` isn't exposed as a textbox, so find it by label or placeholder. The code picker needs a short pause after selecting.
- **Expected console noise:** two 401s from `/api/me` on the login page, and 409s in the duplicate-name and two-editor tests.
- **The exact headless Chrome path used:** `CHROME_PATH=$HOME/Library/Caches/ms-playwright/chromium_headless_shell-1223/chrome-headless-shell-mac-arm64/chrome-headless-shell npx tsx spikes/ui-<name>.ts`.
- **Leftover test data:** published "Workflow test …" questionnaires stay in Hospital A, because published versions can't be deleted. Reset the local database if it gets cluttered.
- **After using another branch, master's tools break** (found 8 Oct, after the `dev` branch ran on 7 Oct).
  - **The local `sj` database keeps that branch's schema** (its migrations 5–8), so `make seed` and the app fail. Rebuild it:
    ```sh
    docker compose exec -T db psql -U sj -d postgres -c "DROP DATABASE sj WITH (FORCE)" -c "CREATE DATABASE sj OWNER sj"
    make migrate seed
    ```
  - **`apps/web/.next/dev/types/` keeps that branch's routes,** so `tsc` in `make lint` fails with "Cannot find module '…/page.js'". Delete the folder; `next dev` regenerates it. (*Only on branches older than S04; master has no Next.js since 12 Oct.*)
- **HJE Full HQ can't be recreated from the repo:** it was transcribed from screenshots. Before rebuilding the database, save it, then load it back after `make migrate seed`:
  ```sh
  docker compose exec -T db pg_dump -U sj --data-only -t questionnaires -t questionnaire_versions -t chapters sj > hje.sql
  ```
  (This works while the database holds only HJE Full HQ; otherwise filter the rows.)
- **On Rahul's Mac, the `docker` CLI isn't on the PATH of Claude's shells** (builder and tester sessions). Docker itself runs; Claude prefixes `PATH="/Applications/Docker.app/Contents/Resources/bin:$PATH"` for `make test-integration`.
- **Claude in Chrome's tab can stay "hidden" even with Chrome in front** (S02a testing, 9 and 11 Oct).
  - **Symptoms:**
    - `document.visibilityState` is `hidden`, and no animation frames run;
    - Mantine drawers, modals and dropdowns are in the page but never appear, and screen clicks on them are lost;
    - timers can be slowed to about once a minute.
  - **Check:** run `requestAnimationFrame` in the page; if it doesn't fire within a second, the tab is hidden.
  - **Workaround the testers used:**
    - click the app's own controls from script, and confirm results in the page text and the request log;
    - send a dialog's request directly when its content can't show, and say so in the report;
    - run long scripts in the background and poll their results, because the tool times out after 45 s.
- **End-to-end journeys (`e2e/`, found in S03, 12 Oct):**
  - **SurveyJS hides the real radio input** behind a drawn one, so Playwright's `check()` waits forever ("intercepts pointer events"). Tap the option's text instead: `support.choose(page, question, option)`.
  - **`make dev` adds a Next.js tools button** whose name contains "Next", so `get_by_role("button", name="Next")` matches two buttons. Use `exact=True` (until F2 removes Next.js). *Superseded 12 Oct (S04): Next.js is gone; the journeys keep `exact=True`, which is harmless.*
  - **A failed journey keeps one trace per browser plus one for the API set-up** (`trace-1.zip`…, in no fixed order), in `e2e/test-results/<journey>/`.
  - **To stop a pytest run with one clean message, raise `pytest.UsageError` in `pytest_sessionstart`.** `pytest.exit` from a fixture prints it twice, with pytest noise.
  - **On Rahul's Mac, `python3` is 3.10;** `python3.12` and `python3.13` come from Homebrew. `make e2e-setup` uses `python3.12`.
  - **For F4's pipeline:** CodeBuild needs Python 3.12+ (`PYTHON=` if it isn't `python3.12`), and the journeys sign in with the dev stub, so QA needs it until S12.
- **`docker` may not be on a Claude Code shell's PATH on Rahul's Mac** (found verifying S03, 12 Oct), so `make test-integration` stops at `db-up` with "docker: No such file or directory". Prefix the command with `PATH="/Applications/Docker.app/Contents/Resources/bin:$PATH"`.
- **`.env` wins over the shell for make (found in S06, 14 Oct):** the Makefile loads `.env` and exports it, so an `AWS_PROFILE` there beats the one in your shell. So every AWS target takes **`PROFILE=<profile>`** (e.g. `make aws-setup PROFILE=AdministratorAccess-026090536258`), which wins over both; Rahul's `.env` names another project's profile. Every AWS script prints the profile and account first and stops before doing anything if the account is wrong.
- **Make variables named `CDK…` break the CDK CLI** (S06): the Makefile exports every variable, and the CLI reads `CDK_*` environment variables as options ("Unknown option(s): --"). Hence `AWS_CDK` and `AWS_CDK_VERSION`.
- **Node 23 prints a large "end-of-life" banner from jsii** on every CDK Python run (S06). Harmless; `export JSII_SILENCE_WARNING_END_OF_LIFE_NODE_VERSION=23` hides it, or use Node 22 or 24.
- **The Claude in Chrome network log shows every `204 No Content` as 503** (found in S01 testing, 8 Oct). `fetch` and `curl` both return the real 204. Testers should ignore a 503 on a route that answers 204, such as a DELETE, publish or logout. Other error codes are reported correctly.

## 5. Open items (as of 8 Oct 2026)

- **Next work (25 Oct):** plan C4 done (S23–S26, 23 Oct; not pushed). S27, the 24 Oct follow-ups (§2: cross-trust copy without asking, archived View and badge, Cancelled wording, the format names, clickable POA Summary answers), is verified (24 Oct, verification 3; not pushed). Rahul answered S27's open items the same day (§2, "S27's open items answered"). **S28** (those answers plus S27's small fixes) verified 25 Oct ([verification 1](sessions/S28-s27-followups/verify.md); not pushed; its minor follow-ups are in pending.md, "From S28's test report 1"). Rahul answered S28's open items on 25 Oct (§2, "the open items after S28 answered"). Next: write and run **S29** (those answers), then split plan C5. **Before go-live (C10):** IG/DPO sign-off on copying answers across trusts without asking, and on the four-field join when staff fill in on the patient's behalf.
- **Next work:** S17 (plan C2's last session) verified 15 Oct ([verify.md](sessions/S17-aws-messaging-jobs/verify.md), verification 1; **plan C2 done**); its minor follow-ups are in [pending.md](pending.md) §3, and Rahul's hand checks are in its verify.md. S17 is committed, so **the push is now safe**: one push brings the pipeline up to S07's docs, S12 and S17 (what its first run should show: S17's verify.md, "Pushing after this commit"). Next: plan C3. Earlier: S12 (Cognito staff sign-in, plan C1's last session) verified 14 Oct ([verify.md](sessions/S12-cognito-signin/verify.md), verification 1; **plan C1 done**); its minor follow-ups are in [pending.md](pending.md) §3. **QA now needs credentials** (test users: `make qa-users`, infra/README.md "Signing in on QA"). S12 is committed, so **the push is now safe** (it brings the pipeline up to S12; H-3 is its first run). Earlier: S06 (F3's Lambda half and F4 Step 1: QA on AWS at https://main.d33kh0gi8vk83v.amplifyapp.com) verified 14 Oct ([verify.md](sessions/S06-aws-qa-stack/verify.md), verification 1; F3 done; minor follow-ups in [pending.md](pending.md) §3); then S07 (the pipeline). QA stays open until S12 (Rahul, 14 Oct). The local phase is done. S16 (plan C2, jobs and reminders) verified 13 Oct ([verify.md](sessions/S16-jobs-reminders/verify.md), verification 1); its minor follow-ups are in [pending.md](pending.md) §3. Next is the AWS phase, with Rahul there (S06 first; S17 finishes C2). S15 (plan C2, patient home) verified 13 Oct ([verify.md](sessions/S15-patient-home/verify.md), verification 2; answers 6–9: first name and surname in every account match; answers 10: the unjoined-twin sign-in is an accepted risk); its follow-ups are in [pending.md](pending.md) §3 (**Rahul to decide** whether staff get a tool to join one person's two accounts). **Rahul to hand-check R-6** (suspending Trust B hides its questionnaires from a patient's home) ([build notes](sessions/S15-patient-home/build-notes.md)). S14 (plan C2, patient invites) verified 13 Oct ([verify.md](sessions/S14-patient-invites/verify.md), verification 3); X-1 skipped by Rahul; its follow-ups are in [pending.md](pending.md) §3. Earlier: S13 (plan C2, patient records) verified 13 Oct ([verify.md](sessions/S13-patient-records/verify.md), verification 1); its follow-ups are in [pending.md](pending.md) §3. **Rahul to decide** the date-of-birth guessing exposure (pending.md §3, S13). S11 (plan C1, the audit log) verified 13 Oct ([verify.md](sessions/S11-audit-log/verify.md), verification 2); its follow-ups are in [pending.md](pending.md) §3. **Rahul to hand-check** a real Export CSV download and the patient journey in a real 390×844 window. S10 (plan C1, the superadmin console and settings) tested 13 Oct (test report 2 all passed; verification 1's fixes retested in report 3, all passed), awaiting re-verification; its minor follow-ups are in [pending.md](pending.md) §3 ([build notes](sessions/S10-superadmin-console/build-notes.md)). S09 (plan C1, staff invites) verified 13 Oct ([verify.md](sessions/S09-staff-invites/verify.md)); its minor follow-ups are in [pending.md](pending.md) §3. Next: S10. S08 (trusts and permissions) verified 13 Oct; its minor follow-ups are in [pending.md](pending.md) §3. Plan **F3** ([plans/f3-lambda-entrypoints.md](plans/f3-lambda-entrypoints.md)), session S05 (the local half) verified 12 Oct; its minor follow-ups are in [pending.md](pending.md) §3. F3's Lambda half is S06 (AWS phase). Plan F2 is done (S04 verified 12 Oct; the verifier checked T-19 in the production build, so no hand check is needed). **Rahul to click Print once** on a POA Summary. Plan F1 is done (S03 verified 12 Oct); its six minor e2e follow-ups are in [S03's verify.md](sessions/S03-e2e-journeys/verify.md).
- **The `dev` branch** (on `origin` too; commit `557be2c`, 7 Oct): *settled 12 Oct: ignored entirely* (§2). Delete it later if wanted.
- **Build order: decided 12 Oct.** Local first (S04, S05, S08–S11, S13–S16), then the AWS phase (S06, S07, S12, S17), as in §2.
- **No domain chosen yet.** `<domain>` in the docs is a placeholder.
- ~~**QA's Neon database needs a reset and re-seed when S18 is pushed**~~ *Not needed (S18 builder, 20 Oct): migration 00019 removes the made-up episodes itself, so the pipeline's migrate step does the reset* (§2, "S18, episode forms"). Know that the push **deletes every episode on QA**. *Verifier, 20 Oct:* the pipeline deploys the Lambdas before it migrates, and 00019 isn't backward-compatible, so expect a few minutes of 500s on QA's episode routes between the two steps (pending.md, "From the S18 verification").
- **The Dutch brochure (`sales/nl/`, 20 Oct):**
  - **Rahul to decide** A-10 versus the brochure's EHR integration offer, and the other unplanned features (§2, 20 Oct).
  - It needs a native Dutch speaker to proofread it.
  - It has no contact details (Rahul removed the placeholder line, 20 Oct). The salesperson hands them over in person.
  - The brochures and the seller sheet say "PreOpsis"; change them to "Preopsis" (Rahul, 20 Oct).
  - Check the seller sheet's status table before each meeting.
- **Research files:** `research_notes/` and `reports/` at the repo root are untracked. Either keep them (e.g. move them to `docs/research/` and fix the links in saas-requirements §0 and §13 and in this file) or delete them.
- **What the patient link returns:** *settled 12 Oct* (§2, S14). S14 builds it.
- **The proposed parts of A-20** (above, 7–8 Oct): the national ID, the per-hospital time zone and the locale were agreed 12 Oct. `uk.<domain>`, one AWS account per environment and the support tooling still need Rahul's yes (the AWS phase).
- **App branding** still shows the old name in the UI (`apps/web/src/app/layout.tsx`, `h/[hospitalId]/layout.tsx`, `PatientHQ.tsx`). Rename it to Poised. *Done in S04 (12 Oct), pending verification.*
- **S04's manual tests and verification found small items, none blocking:** they're in [pending.md](pending.md) §3 ("Found by the S04 manual tests" and "From S04's verification"). The worst: the POA Summary lists a clinician's change to a question that a later change hid. **Rahul to click Print once** on a POA Summary (no tester could open the real dialog).
- **Unanswered:** should clinician-only Question Sets render full width in `ValidateSet.tsx`, rather than in an empty two-column grid?
- **Unconfirmed:** whether saved option lists should be copied (as built) or linked.
- **Unsaved test data:**
  - **The comprehensive test HQ is gone.** On 8 Oct the local database held only HJE Full HQ. Rebuild it from `apps/api/internal/apitest/testdata` fixtures if it's needed again.
  - **HJE Full HQ exists only in local databases.** Consider saving it as a file in the repo (§4 has the commands).
- **Content:** HJE Full HQ sets 5–11 and their disclosures still wait on a real incumbent export (`tools/export-hq.js`). Low priority (functionality over content).
- **Small issues:**
  - "Years since a date" opens with an empty list and no message.
  - The Medication, Admissions and date-list disclosure fixes have no test-plan cases.
- **Stale docs:**
  - README says there's no publish workflow, and its seed-user table lacks Cara Clinician and Alex's publisher role.
  - `pending.md` §4 lists Versions and publish as not started.
