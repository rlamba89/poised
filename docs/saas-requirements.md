# SaaS productionisation — requirements and approach

Drafted 2 Oct 2026 with Rahul and revised the same day after his answers. Status: **brainstorm draft**. Items marked *Agreed* are settled. Items marked *Proposed* need Rahul's yes before a session builds them.

**Read this first if you are a new session.**
- Poised has proven the authoring tool and one thin pass of the episode workflow: publish → the patient fills in the HQ by link → a clinician validates → POA Summary.
- [plan-workflow.md](plan-workflow.md) describes what exists. [requirements.md](requirements.md) holds the authoring requirements; IDs like PNL-07 and CAL-02 refer to it.
- This document covers what's needed to turn Poised into a multi-trust SaaS product on AWS.
- References to Lifebox, from the repos under `~/dev/lifebox/`, are **product evidence only**: what hospitals use and expect today. **Lifebox is never the template for our technical architecture** (rule 8).

---

## 0. Decisions log

| # | Topic | Decision | Status |
| --- | --- | --- | --- |
| A-1 | Existing data | **Fresh app, with no historic data.** There's no data migration and no backward compatibility, so tables can be reshaped and migrations squashed before go-live. | Agreed |
| A-2 | Tenancy | **Trusts with many hospitals.** Some decisions are made at trust level and some at hospital level (section 3). | Agreed |
| A-3 | Patients | **One account, many records.** A patient has one login, and a patient record per trust links to it. Records are identified by **NHS number**, not by hospital number (section 4). | Agreed |
| A-4 | Patient sign-in | **A magic link, received by SMS or email, plus date of birth**, built in Go. No passwords, and no codes to type (section 4). | Agreed |
| A-5 | Roles | **Admin, super clinician, clinician** (plus our platform admin). **Everyone in a hospital sees all its data.** Roles only limit what people can *do* (section 2). | Agreed |
| A-6 | Videos | **Both:** authors paste a link, and the platform ships a built-in library of suggested videos. **All videos are hosted on YouTube or Vimeo**, never by us. | Agreed |
| A-7 | "Capture" Group/Section | A container that **only clinicians fill in**. | Agreed |
| A-8 | ASA | **Platform defaults that can be changed per HQ.** Low ASA → auto-triage, **by default straight to Ready for admission**. High ASA → book an anaesthetic review. Clinicians can change the target per HQ. | Agreed |
| A-9 | Pre-fill | Any question the patient answered in an earlier HQ is pre-filled in a new one, **however old the answer is** (section 6). | Agreed (design Proposed) |
| A-10 | EPR integration | **Out of scope.** | Agreed |
| A-11 | Cloud | **AWS, serverless, near-zero cost when idle.** Low scale is expected. **Nothing may cost money while it's unused before the first customer**, so there's no always-on database. | Agreed |
| A-12 | Staff sign-in | Clinicians sign in as hospital staff, with MFA, through **Amazon Cognito** (a staff user pool). | Agreed |
| A-13 | SMS / email | **AWS:** SES for email, and AWS End User Messaging (SNS) for SMS. | Agreed |
| A-14 | Virus scanning | **None.** This is an accepted risk; the mitigations are in section 9. | Agreed |
| A-15 | Files and messaging | Max **25 MB** per file. **No messaging** between patients and clinicians yet. | Agreed |
| A-16 | Infrastructure and delivery | **AWS CDK in Python**, deployed by **AWS CodePipeline**. No SAM, and Lifebox's architecture isn't the reference. | Agreed |
| A-17 | MMSE | Not now. | Agreed |
| A-18 | Licensed instruments | **Build them now** (EQ-5D-5L, Oxford Hip and Knee Scores). Licence costs are decided later and can be a paid add-on. Each is **opt-in, set when a trust or hospital is added**: a trust opts in or out, and each hospital can follow the trust or decide differently (PRM-08). Licensors aren't contacted for now. | Agreed |
| A-19 | Data regions | **London (eu-west-2) for now.** The architecture must let a trust's **data** live in another AWS region later. Code and the control plane stay in London (section 13). Not built this session. | Agreed |
| D-1 | Server-side form evaluation | A small **Node Lambda** runs `packages/clinical` to compute disclosures, ASA and triage when a patient submits (section 10). This **reverses the 30 Sep "no Node service" decision**. | **Agreed 2 Oct** |
| D-2 | Database | **Plain Postgres throughout.**<ul><li>**Neon** (AWS London) until the first customer: free tier, sleeps when idle, wakes in under a second.</li><li>**RDS for PostgreSQL** in our own AWS account from the first customer.</li></ul>Both are plain Postgres, so the move is a dump and restore (section 13). | Agreed |
| D-3 | PDF rendering | Browser print first; a headless-Chromium Lambda later. | Proposed |
| D-4 | Lambda layout | **Four functions from one Go codebase**, split by audience and job: staff API, patient API, jobs, and evaluate (Node). Not one Lambda, and not one per route (section 13). | Agreed |
| D-5 | Frontend | A **plain React SPA (React + Vite + React Router)** on **AWS Amplify Hosting**, replacing Next.js. The Mantine UI and `packages/clinical` stay (section 13). | Agreed |
| D-6 | Sessions | **Sessions in Postgres** for staff and (later) patients. The cookie holds a random id, and the table keeps only its hash. Logout and idle timeout therefore end a session at once. Staff: 8 h maximum, 30 min idle. Built 6 Oct ([plans/auth-first.md](plans/auth-first.md)). | Proposed (built; Rahul to confirm) |
| D-7 | MFA timing | **Staff MFA off until before real patient data** (C10). This departs from A-12 for now. | Proposed |

---

## 1. Simplicity rules (apply to every section)

1. **Everything a person fills in is a SurveyJS form.**
   - That covers the HQ, PROMs, assessments and clinician capture blocks.
   - They share one editor, one renderer, one versioning model and one answer store.
2. **One Go codebase and one Next.js app**, both serverless.
   - The Go code is deployed as a few Lambdas split by audience (D-4).
   - Background work (reminders, PROMs sends, auto-archive) uses a `jobs` table, drained by a scheduled Lambda.
   - There are no queues or extra services.
3. **The trust is the tenant.**
   - `org_id` is on every row and checked in Go middleware.
   - `hospital_id` narrows access within a trust.
   - There's one shared database.
4. **Question keys link answers across forms** (section 6). One idea powers four features: pre-fill, assessments, risk scores and PROMs baselines.
5. **Platform content is just data with no owner.** Built-in videos, validated scores, PROMs, the question library, default ASA rules and document templates have `org_id IS NULL`.
6. **Defaults over settings.**
   - Platform default → trust → hospital (or → HQ, for clinical rules).
   - A setting exists only where customers really differ.
7. **Automation never sets a clinical value silently.**
   - The ASA and scores are *suggested*.
   - Auto-triage is the one automatic status change. It's visible, explained ("ASA 2, no red flags") and reversible. Its rules are signed off with the HQ, and every auto-triaged episode appears on a worklist.
8. **Choose technology on its own merits, never because Lifebox uses it.** Lifebox shows what hospitals need (statuses, observation fields, assessments). It doesn't decide how we build. Every technical choice in this document has its own reason written next to it.

**Choices that came from Lifebox, checked on 2 Oct 2026:**

| Choice | Where | Verdict |
| --- | --- | --- |
| Answers stored per actor (the patient's original + the clinician's validated copy) | Built (plan-workflow) | **Keep, on merit:** audit needs the patient's own words, and the POA Summary shows both. |
| Episode status names | Built (plan-workflow) | **Keep:** this is product vocabulary hospitals already know. The keys and transitions are ours (section 5). |
| Editor, clinician view and POA screen layouts | Built (plan-redesign) | **Keep:** a product/UI decision taken on 1 Oct, not a technical one. |
| Observation fields and assessment list | Section 9 | **Keep:** the standard pre-op clinical set. Our storage design is our own. |
| SAM, video hosting, STOP-Bang first, Cognito, AWS London | Earlier drafts | **Settled 2 Oct:**<ul><li>SAM is dropped;</li><li>videos are hosted on YouTube or Vimeo;</li><li>STOP-Bang stays first;</li><li>Cognito stays for staff;</li><li>London is the default region, with other data regions possible later (A-19).</li></ul> |

---

## 2. Who uses it

**Agreed:** everyone in a hospital can see all of its data. Roles only decide what people can *do*, and each role includes everything the role below it can do.

| Actor | Scope | Can do (*Proposed*) |
| --- | --- | --- |
| **Platform admin** (us) | All trusts | Create trusts and hospitals. Invite the first trust admin. Manage platform content: videos, question and score library, default ASA rules, templates. No routine access to patient data (OPS-06). |
| **Clinician** | A trust, or one hospital | Patients and episodes (including the booking work), validating HQs, observations, assessments, nurse ASA, files, tasks. |
| **Super clinician** | A trust, or one hospital | Everything a clinician does, plus:<ul><li>author, review and publish HQs, ASA rules and PROMs plans;</li><li>anaesthetist ASA;</li><li>re-open reviews and un-archive;</li><li>override auto-triage.</li></ul>Sign-off rule: the approver must not be the editor of that version (SGN). |
| **Admin** | A trust, or one hospital | Everything a super clinician does, plus settings, staff and roles, procedures, the content library and document templates. |
| **Patient** (or a carer for them) | Their own records, in any trust | Fill in forms, watch and read information, share files. |

This replaces today's `viewer / author / reviewer / publisher / hospital_admin / clinician` roles. In code it's one ordered role (`clinician < super_clinician < admin`), so a permission check is a single comparison.

---

## 3. Trusts, hospitals and staff

### How trusts and hospitals split decisions (*Proposed*)

- **A trust** (or an independent provider group) is the customer, the tenant and the data controller.
- **A hospital** is a site within a trust. An independent single-site customer is a trust with one hospital.
- **Patient records belong to the trust** and are identified by **NHS number**. Episodes belong to a hospital. A hospital number is an optional label on the episode, never an identity (section 4).
- **Content** (HQs, procedures, information content, document templates) is owned by the trust and used by all its hospitals, or owned by one hospital and used only there.
- **Settings** resolve as platform default → trust → hospital. Each setting is marked as either "trust-only" or "a hospital can override".

### Requirements

| ID | Requirement | Priority |
| --- | --- | --- |
| ORG-01 | Only a platform admin can create a trust or a hospital. There's no public sign-up. | Must |
| ORG-02 | **Creating a trust** takes its name, short code, contact details, logo and the first trust admin's email (they get an invite). **Creating a hospital** takes the trust, name, address, patient-facing phone and email, and an optional logo. | Must |
| ORG-06 | **Data region.** When a trust is created, the platform admin picks its data region (default London). All of the trust's patient data is stored in that region. The region can't be changed later without a planned migration. Not built yet (A-19); the architecture allows it (section 13). | Could |
| ORG-03 | A trust or hospital can be suspended (no logins, data kept) and re-activated. | Must |
| ORG-04 | **Settings**, by level:<ul><li>**trust-only:** retention period, cross-trust pre-fill allowed;</li><li>**hospital can override:** feature switches and opt-ins (ORG-05), branding, contact details, default HQ, reminder schedule, auto-archive periods, the "what happens next" text and patient message templates.</li></ul> | Must |
| ORG-05 | **Feature switches and opt-ins:** assessments, PROMs, files, observations, and **each licensed instrument** (PRM-08). They're set when a trust or hospital is added, and can be changed later. The trust sets the default, and each hospital either follows it or opts in or out on its own, so some hospitals in a trust can have a feature and others not. | Must |
| STF-01 | An admin invites staff by email, chooses the **scope** (the whole trust or one hospital) and the **role**. The invite expires after 7 days. | Must |
| STF-02 | One staff account can belong to several trusts and hospitals, with a role in each. Inviting an existing email adds a membership; it doesn't create a second account. | Must |
| STF-03 | After sign-in, a staff member picks a hospital (when they have more than one) and can switch from the header. The current trust and hospital are always shown. Trust-scoped staff can also choose "All hospitals" on worklists. | Must |
| STF-04 | Removing a membership ends access at once. Past actions stay in the audit log. | Must |
| STF-05 | Staff sign in with email, password and **MFA** (an authenticator app). Later, a trust can switch on SSO with its Microsoft Entra ID / NHSmail. | Must (SSO Should) |
| STF-06 | Every view and change of patient data is written to an audit log (who, what, when, trust, hospital). Admins can export it. | Must |

### Approach (*Proposed*)

- **Data:**
  - `orgs (id, name, code, status, settings jsonb)`
  - `hospitals (+ org_id, status, settings jsonb)`
  - `users (+ cognito_sub)`
  - `memberships (user_id, org_id, hospital_id NULL = whole trust, role)`
  - `staff_invites`
  - `platform_admins`
- **Effective settings** = the platform defaults in Go, merged with `orgs.settings`, merged with `hospitals.settings`, filtered by the "hospital can override" list. This is one small Go function.
- **Routes:** `/api/o/{oid}/…`. Episode rows carry `hospital_id`, and a hospital-scoped user only sees their hospital's rows. The frontend URL becomes `/o/{oid}/h/{hid}/…`.
- **Staff auth:**
  - A Cognito **staff user pool** with email + password + required TOTP MFA. Cognito's managed login pages handle sign-in, MFA, reset and lockout, plus SAML/OIDC federation with Entra later.
  - After the OAuth code flow, Go verifies the ID token and sets **our own session cookie**, the same model as patients. No API Gateway authorizer is needed (C1).
- **Audit:** one append-only `audit_log` table, written by Go middleware.

---

## 4. Patients: identity, invites and sign-in

### Requirements

| ID | Requirement | Priority |
| --- | --- | --- |
| PAT-01 | Staff search the trust's patients by NHS number, name and date of birth, and create a record if none matches. Within a trust, a record is unique by **NHS number** (validated with its check digit), and a duplicate is refused. | Must |
| PAT-01a | **The NHS number is optional**, for patients who don't have one or don't know it: private, self-pay and overseas patients, and patients from Scotland (CHI) or Northern Ireland (H&C). Without it, staff are warned about possible duplicates with the same name and date of birth. | Must |
| PAT-01b | The **hospital number** is an optional field on the *episode* (the number at that site). It's searchable and printed on the POA document, so the hospital can file the document, but it's never used to identify the patient. | Should |
| PAT-02 | A patient record can have many episodes at any of the trust's hospitals, including more than one open at once. | Must |
| PAT-03 | Creating an episode sends an invite by **SMS and email** (whichever the record has). It can be resent. The SMS names the hospital, e.g. "St Mary's Hospital: please complete your pre-op health questionnaire: <link>". | Must |
| PAT-04 | **Every session starts with a link plus date of birth.** The patient taps the link in the SMS or email, taps **Continue** (this stops email security scanners from using up the link), and enters their **date of birth**. There are no codes to type. The link alone never shows data. | Must |
| PAT-05 | **Invite links** keep working until the episode closes, so patients can tap the same SMS again days later. **Sign-in links** (PAT-06) work once and expire after 15 minutes. Five wrong dates of birth lock the link, and the patient must ask for a new one. | Must |
| PAT-06 | A returning patient without their SMS goes to the site, enters their mobile or email, receives a sign-in link, and lands on their **home**: every open episode from every trust, newest first. | Must |
| PAT-07 | If an account with that verified mobile or email already exists, the new record joins it, so one login shows every trust. | Must |
| PAT-08 | A carer or parent can manage more than one person from one login. Each record is still verified by that person's date of birth (PAT-04). | Should |
| PAT-09 | Staff can complete the HQ **on the patient's behalf**, e.g. on the phone. The answers are marked "entered by <staff member>". | Must |
| PAT-10 | Patients can update their contact details. Name, date of birth and NHS number come from the hospital, with a "contact your hospital" note (PNL-04). | Should |
| PAT-11 | Sending sign-in links is rate-limited per number and per email. SMS goes only to UK mobile numbers, which blocks SMS-pumping fraud. | Must |

### Approach (*Proposed*)

- **Data:**
  - `patient_accounts (id, mobile, email)` is the login.
  - `patients (org_id, account_id NULL, nhs_number NULL, name, dob, sex, mobile, email)` is the trust's record.
  - `account_id` stays NULL until the first invite is used.
  - `episodes.hospital_number` is a label (PAT-01b).
- **Patient sign-in is built in Go, not Cognito.**
  - Cognito has no built-in magic link; its "passwordless" sign-in means typing a code.
  - A magic link through Cognito needs three custom-auth trigger Lambdas (there's an AWS sample), which is more moving parts than doing it ourselves.
  - What we build:
    - `login_links (token_hash, account_or_patient_id, kind invite|signin, expires_at, used_at, failed_dob_attempts)`;
    - a random 32-byte token, of which we store only the hash;
    - the session is a signed, HTTP-only cookie with a 30-minute idle timeout.
- **Staff use Cognito** (A-12). Go checks two kinds of credential: Cognito JWTs for staff, and our own signed cookie for patients.
- **SMS:** one platform sender ID, registered for UK sending in AWS End User Messaging. A protect configuration allows UK numbers only.
- **Email:** SES, with a verified domain.

---

## 5. The episode journey and lifecycle

### 5.1 End-to-end pre-assessment journey (*Proposed*)

```
 HOSPITAL                          PATIENT                            STATUS
 ────────                          ───────                            ──────
 1 Find/create patient (trust)
 2 New episode at a hospital:
   procedure (from the trust's
   list) → default HQ, info
   content, PROMs plan; dates  ──► SMS + email invite             hq_not_complete
                                   3 Link → Continue → DOB
                                     → Home: to-do (HQ, videos,
                                       leaflets, PROMs baseline)
   Reminders day 2 and day 5 ──►   4 Fill in the HQ: pre-filled
   Day 7 with no submit →            from earlier HQs ("still
   "Not responding" worklist →       true?"), autosave, resume,
   staff phone & complete it         videos in the flow
                                     Submit → "what happens next"
 5 On submit the server works
   out disclosures, suggested
   ASA and the HQ's ASA rules:
   ├ low ASA, no red flags  → AUTO-TRIAGED ─────────────────────► ready_for_admission
   │                          (target set per HQ)
   └ otherwise               → nurse validates each set ────────► ready_for_review
     high ASA → task "Book anaesthetic review" + worklist flag       → ready_for_poa
 6 POA: observations, assessments
   (pre-filled), nurse/anaesthetist
   ASA, files, tasks                                             poa_complete /
                                                                 on_hold / not_ready
 7 Fit for surgery            ──►  Pre-admission info: fasting,  ready_for_admission
                                   medicines to stop, what to bring
 8 Day of surgery                                                admitted
 9 Discharged (date)          ──►  PROMs sent on schedule        discharged → recovery
10 Auto-archive N days after discharge, or when PROMs close      archived
   Cancelled at any point                                        cancelled → archived
```

### 5.2 Statuses

| Key | Label | Set by |
| --- | --- | --- |
| `hq_not_complete` | HQ not complete | Creating the episode |
| `ready_for_review` | Ready for review | Submitting (patient or staff), when not auto-triaged |
| `ready_for_poa` | Ready for POA | Completing the review |
| `poa_complete`, `on_hold`, `not_ready` | As today | A clinician |
| `ready_for_admission` | Ready for admission | A clinician, **or auto-triage** (the default target; the HQ can choose another) |
| `admitted` | Admitted | A clinician |
| `discharged` | Discharged | A clinician, with a discharge date |
| `recovery` | Follow-up (PROMs) | Automatic on discharge, when PROMs are scheduled |
| `cancelled` | Cancelled | A clinician, with a reason (procedure cancelled, patient withdrew, moved hospital) |
| `archived` | Archived | Automatic (EPI-05), or a clinician |

### Requirements

| ID | Requirement | Priority |
| --- | --- | --- |
| EPI-01 | An episode has a **procedure** from the trust's list (PRC-01), a **planned procedure date** and a **POA appointment date** (both optional), plus admitted and discharged dates. | Must |
| EPI-02 | **Worklists** are saved filters on the clinician home:<ul><li>Ready for review</li><li>Auto-triaged (for a quick check)</li><li>Not responding (more than 7 days since invite, no submit)</li><li>Anaesthetic review needed</li><li>Procedure within 14 days and not Ready for admission</li><li>On hold</li></ul> | Must |
| EPI-03 | Every status change records who, when, from, to and why (automatic changes say which rule fired). | Must |
| EPI-04 | Archived episodes are read-only, leave every worklist, and stay searchable. **Un-archive** restores the previous status. | Must |
| EPI-05 | **Auto-archive:** a daily job archives episodes N days after discharge (or after the last PROMs form closes), and pre-op episodes with no activity for M months. The defaults are 30 days and 6 months, which a hospital can override. | Must |
| EPI-06 | **Retention:** after the trust's retention period (default 8 years after the episode ends; the trust is the controller and must confirm it), the episode's data is deleted and a tombstone is kept for audit. | Should |
| EPI-07 | A super clinician can re-open a completed review or an auto-triage, with a reason. | Must |
| EPI-08 | A status change can send the patient a message from a hospital-editable template, e.g. Ready for admission → "You're fit for surgery; here's what happens next". | Should |
| EPI-09 | A patient can close their account. The login is removed and the trust's records stay. | Must |
| PRC-01 | The trust keeps a **procedure list**: name, optional code, specialty, **default HQ**, **information content** (INF-04) and **PROMs plan** (PRM-01). Picking a procedure on a new episode fills these in, and a clinician can change them. | Must |

### Approach (*Proposed*)

- **An episode holds several forms.** Instead of `episodes.version_id`, add `episode_forms (id, episode_id, version_id, kind hq|assessment|proms, due_at, closes_at, status, submitted_at)`. `episode_answers` then hangs off `episode_form_id`. One mechanism covers the HQ, the PROMs baseline, PROMs follow-ups and assessments.
- **Jobs:** `jobs (id, run_at, kind, payload jsonb, done_at, attempts)`. An EventBridge Scheduler rule runs the Go Lambda's job entry point every 5 minutes, and it claims due rows with `FOR UPDATE SKIP LOCKED`.

---

## 6. Pre-filling from earlier HQs (*Agreed need; Proposed design*)

**Goal:** if the patient has answered a question before, in any earlier HQ, the new HQ is filled in for them, and they only confirm or change it.

### How it works

1. **Every answerable question has a question key.** It's the SurveyJS `name`, our stable ID (LCY-07): unique within an HQ and never reused.
2. **Two questions with the same key are the same question,** so their answers carry over. Keys survive:
   - a new version;
   - copying an HQ, a Question Set or a question into another HQ;
   - inserting from the **question library**: a platform or trust library of common questions (allergies, medicines, smoking, previous anaesthetic problems…).
   - **Change to PNL-06:** library inserts *keep* their keys, rather than getting new IDs.
3. **"Same as…" link:** an author can link a new question to an existing one ("This is the same question as … in *HJE Full HQ*"). The new question adopts that key, but only when the type and option values are compatible.
4. **Locked meaning:** wording can be edited, but a keyed question's **option values are locked**, so an old answer always means the same thing.
5. **"Always ask fresh":** a question setting for things that change, such as current symptoms, pregnancy or today's weight. Those questions are never pre-filled.
6. **Where answers come from:**
   - When a patient opens a new HQ, Go looks through their earlier **submitted** episodes (newest first) for each patient question's key.
   - It prefers the clinician-validated answer (the final one) over the patient's original.
   - **Answers never expire** (agreed). Every pre-filled answer shows its date, and anything that changes over time is marked "Always ask fresh" (point 5).
   - This is a plain JSON lookup, so Go still never evaluates SurveyJS.
   - Repeating panels (medicines, admissions) are copied row by row.
7. **Other trusts:** answers from another trust are offered only if the trust allows it (ORG-04) **and** the patient agrees on screen: "Use the answers you gave St Mary's in March?"

### Requirements

| ID | Requirement | Priority |
| --- | --- | --- |
| PRE-01 | Questions in a new HQ whose keys the patient answered before are pre-filled, using the rules above. | Must |
| PRE-02 | Each Question Set with pre-filled answers opens with: "We've filled in 23 answers from your questionnaire on 3 Mar 2026. Please check them." There are two buttons: **Check each answer** and **Everything is still correct**. The second skips to the questions that weren't pre-filled. | Must |
| PRE-03 | Pre-filled answers show a small "From your answers on 3 Mar 2026" marker in the patient view. | Must |
| PRE-04 | The clinician sees which answers were pre-filled, where from, and whether the patient changed them. | Must |
| PRE-05 | The editor shows on each question "Pre-fills from N other HQs" or "New question", and offers "Same as…" and "Always ask fresh". | Must |
| PRE-06 | Clinician-only questions and Profile fields are never pre-filled from answers; Profile comes from the patient record. | Must |

**Storage:** each `episode_answers` row gets `prefill jsonb` (`{key: {fromEpisode, answeredAt}}`), so PRE-03 and PRE-04 need no extra table.

---

## 7. Patient experience and what hospitals share

Goal: **a patient taps the SMS and is answering the first question within 30 seconds**, and can finish on a phone in short sittings.

| ID | Requirement | Priority |
| --- | --- | --- |
| INF-01 | A **Video** element, placed anywhere in an HQ. The author either picks from the **built-in library** (searchable, and suggested by tags such as "general anaesthetic" or "knee replacement") or pastes a URL. It shows a title, the length and captions. | Must |
| INF-02 | An optional "I have watched this video" tick, recorded as an answer, which can be made required. | Should |
| INF-03 | An **Information** element: rich text with images and links (fasting rules, what to bring, directions, parking). | Must |
| INF-04 | A patient **Information** tab per episode: leaflets (PDF), videos and contact details. Defaults come from the procedure (PRC-01), and a clinician can add items to one episode. | Should |
| INF-05 | Content library management: the platform admin manages the built-in videos (YouTube or Vimeo links, with title, length and tags) and leaflets. Trust admins add their own. | Must |
| PX-08 | **Patient home:** for each episode, the hospital and logo, the procedure, a **to-do list** (forms, required videos, files requested) with progress, Information, Files and contact details. | Must |
| PX-10 | Each Question Set shows an estimated time ("about 4 minutes"). | Should |
| PX-11 | A "what happens next" screen after submitting. | Must |
| PX-12 | Measure the experience per hospital: time to complete, drop-off per page, and % submitted within 7 days. Shown to admins. | Should |
| PX-13 | **Checklist** for the content library and templates: appointment details; fasting; medicines to stop or keep taking; what to bring; MRSA and other screening; prehab and smoking-cessation advice; consent information; the anaesthetic explained; recovery expectations; who to call if unwell before surgery. | Should |

Existing requirements still apply: PX-01 (phones from 320 px), PX-02 (WCAG 2.2 AA), PX-05 (resume), VAL-05 ("Don't know"), the LNG languages, and one question per screen (plan-features step 5). Pre-fill (section 6) is the biggest single time-saver.

**Approach (*Proposed*):**
- Video and Information are new element kinds in `packages/clinical/src/doc.ts`.
- `content_items (id, org_id NULL = platform, kind video|leaflet, title, url, captions_url, tags[], status)`.
- **Videos (agreed):** every video, built-in or the author's own, is hosted on **YouTube or Vimeo** and embedded by URL. We store only the link and metadata.
  - Captions, encoding and streaming are the video platform's job.
  - Built-in videos are uploaded once to our own YouTube (unlisted) or Vimeo account.
  - **Privacy:** embed YouTube through `youtube-nocookie.com`, and Vimeo with `dnt=1`, so patients aren't tracked. Record this in the DPIA template (OPS-03).

---

## 8. Authoring additions

| ID | Requirement | Priority |
| --- | --- | --- |
| CAP-01 | A Group or Section can be marked **Clinician capture**. Everything inside becomes clinician-only, the patient strip removes it, and the clinician view shows it as a tinted capture box. | Must |
| CAP-02 | A capture block can hold any question type, including score panels. Its answers appear in the POA Summary under the block's title. | Must |
| SUM-01 | **More than one Clinical Summary per page.** The summary becomes an element the author places, not a page toggle. Each summary has a title and lists the disclosure notes from **its own Group/Section** (or the whole page, at page level), with its own clinician comment box. | Must |
| SCR-01 | **Editor-authored risk scores** (PONV/Apfel, STOP-Bang…) embedded in an HQ: options carry scores, a Calculation totals them, and **bands** carry disclosures and can drive conditions (CAL-02, plan-features step 9). | Must |
| SCR-02 | Scores can use the patient's **age and sex** (profile), **BMI** (HQ or observations) and answers from other Question Sets **by question key**. | Must |
| SCR-04 | A **validated score library** (PNL-07): platform items with locked wording, scoring and bands, inserted with their question keys. Each shows its source and licence note. | Should |
| ASA-01 | Each disclosure can carry an **ASA contribution** (1–6). The **platform code library sets a default per code**, and an author can change it on the disclosure in an HQ. **Suggested ASA** = the highest across the episode's disclosures, shown with its reasons ("ASA 3: heart failure, I50"). | Must |
| ASA-02 | **ASA rules per HQ.** A new HQ starts with the **platform default rules**, and super clinicians can change them in the HQ's settings. Rules are part of the published version, so they're versioned, signed off and testable. The defaults:<ul><li>**Suggested ASA 1–2 and no red review flags → auto-triage.** Every Question Set is stamped "Auto-triaged (ASA 2, no red flags)", and the episode goes straight to **Ready for admission**. Super clinicians can change the target per HQ, e.g. to Ready for POA.</li><li>**Suggested ASA ≥ 3 → add task "Book anaesthetic review"**, flag the episode on worklists, and require an anaesthetist ASA before Ready for admission.</li></ul> | Must |
| ASA-03 | A rule is *when* [suggested or nurse ASA, compared with a number] (optional extra conditions: a score band, a disclosure code, the procedure) *then* one of a fixed list of actions: auto-triage, add a task, add a flag, require anaesthetist ASA, suggest a status. | Must |
| ASA-04 | **Test cases** (PRV-05) can expect a suggested ASA and the rules that fire, so changing the rules is caught before publishing. | Should |
| ASA-05 | ASA can be used in clinician-view conditions, e.g. show the "Anaesthetic plan" capture block when ASA ≥ 3. | Should |
| TSK-01 | **Episode tasks:** title, created by (a user or a rule), assigned role, done by and when. They're shown on the episode and as a worklist. | Must |
| DOC-01 | **Custom POA document layouts.** Trust or hospital admins build templates (e.g. "Full POA", "Anaesthetist summary", "GP letter") by picking and ordering blocks from a fixed list:<ul><li>header;</li><li>patient details (choose the fields);</li><li>status and ASA;</li><li>observations;</li><li>assessments and scores;</li><li>tasks;</li><li>general notes;</li><li>each Question Set summary (validated or patient answers);</li><li>corrections;</li><li>files list;</li><li>sign-off footer.</li></ul> | Must |
| DOC-02 | Download any template as a **PDF**, showing the template name, generated date and who generated it. | Must |
| DOC-03 | A platform default template is used when a trust has none. | Must |

**Approach (*Proposed*):**
- **Capture:** a `capture: true` flag on the Group/Section. When saving, the editor sets `clinicianOnly` on every child, so the Go strip doesn't change.
- **Summaries:** a `clinicalSummary` element kind, scoped to its parent container. **Update requirements.md SGN-04**, which currently blocks more than one summary per page.
- **ASA rules:** stored in the HQ version as JSON, e.g. `{rules: [{when: {asa: ">=3"}, then: ["task:Book anaesthetic review", "flag:anaesthetic_review", "require:anaesthetist_asa"]}]}`. The platform default is copied in when an HQ is created.
- **Where rules run:** when the patient submits (section 10).
- **Document templates:** `document_templates (org_id NULL = platform, hospital_id NULL, name, blocks jsonb)`, edited in a simple list editor (add, remove, reorder, per-block options), **not** a free-form designer. The renderer reuses today's POA Summary components.
- **PDF (D-3):** browser print-to-PDF first. A headless-Chromium Lambda later, when PDFs must be stored or sent.

---

## 9. Files, observations and assessments

### Files

| ID | Requirement | Priority |
| --- | --- | --- |
| FIL-01 | A clinician uploads files to an episode, either **hospital only** (ECG, bloods, letters for reference) or **shared with the patient**. The patient is notified when a file is shared. | Must |
| FIL-02 | A patient uploads files to their episode (e.g. a photo of their medicines list). These are always visible to the hospital. | Must |
| FIL-03 | Allowed types: pdf, jpg, png, heic, doc, docx. **Max 25 MB.** Checked by the server. | Must |
| FIL-05 | Files are listed on the episode and in the POA Summary (name, who, when, visibility). | Must |
| FIL-06 | A clinician can request a file, and the request appears in the patient's to-do list. | Could |

**Approach:**
- S3 with keys `{org_id}/{episode_id}/{uuid}`. The metadata is in a Postgres `files` table.
- Upload with a presigned POST, which enforces the size and type limits. Download with a short-lived presigned GET.
- **No virus scanning (A-14).** Instead: a strict type list, `Content-Disposition: attachment` on every download (files are never shown inline in our origin), and files served from S3's domain, not ours. Record this in the hazard and security logs.

### Observations

| ID | Requirement | Priority |
| --- | --- | --- |
| OBS-01 | Clinicians record: height (cm), weight (kg), BMI (calculated), blood pressure (several readings), heart rate, SpO₂, temperature, respiratory rate, blood glucose, peak flow, and notes for ECG and urinalysis. These are Lifebox's `hospital.observations` fields. | Must |
| OBS-02 | Each reading has a value, when it was taken, and who entered it. Corrections keep history. | Must |
| OBS-03 | Soft plausible-range warnings, plus hard limits. | Must |
| OBS-04 | The POA Summary shows **hospital BMI** next to patient-reported BMI. A trust setting says whether observations take precedence in scores. | Must |

**Approach:** `observations (id, episode_id, kind, value, value2 /* diastolic */, unit, taken_at, entered_by, superseded_by)`, one row per reading. It's a table rather than a form because observations are a fixed numeric series that feeds calculations.

### Assessments through the HQ

In Lifebox, the eight assessments in the screenshot are hardcoded SurveyJS JSON in Go. Patients never answer them. Only STOP-Bang is calculated automatically, from HQ answers, observations and the profile.

**Proposed:**
- An assessment is a clinician form from the score library, attached to the episode as an `episode_form`.
- Its items share **question keys** with library questions in the HQ, so it **opens pre-filled** from the HQ, observations and profile.
- Each item shows its source, missing items are highlighted, and the clinician completes and confirms.
- If the HQ embeds the same score panel (SCR-01), the assessment is already complete.

| # | Tool | Can the patient answer it in the HQ? | Notes |
| --- | --- | --- | --- |
| 1 | **STOP-Bang** | 7 of 8: S, T, O, P (HQ), BMI, age and sex. Neck > 40 cm: clinician, or ask "collar size ≥ 16 in". | First, because it's the most widely used pre-op screen (for obstructive sleep apnoea), and 7 of its 8 items come with no extra work. |
| 2 | **DASI** | 12 of 12 | Fully automatic. Gives METs (functional capacity). |
| 3 | **PRISMA-7** | 7 of 7 | Fully automatic frailty screen. |
| 4 | **Apfel PONV** | 3 of 4 (post-op opioids: clinician) | Small. |
| 5 | **MUST** | Mostly (acute-disease effect: clinician) | |
| 6 | **Falls risk** | Mostly | Confirm which tool the customer uses. |
| 7 | **VTE (DoH/NICE)** | Patient risk factors only | Pre-filled; the clinician completes it. |
| 8 | **Rockwood CFS** | No: it's a clinical judgement | Shows DASI and PRISMA-7 as hints. |
| — | **MMSE** | No: it's done in person, and it's **copyrighted (PAR Inc.)** | **Not now (A-17).** If a cognitive screen is needed later, AMTS and 4AT are common UK alternatives (check their terms). |

---

## 10. Server-side evaluation and auto-triage (D-1 — agreed 2 Oct 2026)

**Problem:**
- Auto-triage (ASA-02) happens when the **patient** submits.
- Today, disclosures are computed only in the browser, and that was safe because the browser was a trusted clinician's.
- A patient's browser isn't trusted: a tampered submit could skip review.
- Go can't run SurveyJS expressions, because there's no non-JS evaluator.

**Proposal:**
- A small **Node.js Lambda, `evaluate`**, runs the same `packages/clinical` code the browser runs.
- Input: the published form JSON, the answers and the patient context. Output: the disclosures, the suggested ASA and the rules that fire.
- Go calls it on submit and stores the result with the episode, then applies the actions: auto-triage, tasks, flags.
- The clinician view still computes live in the browser, and both run the same code.

**Why it's still simple:**
- It's one function, deployed beside the Go API, with no state and no database access.
- It runs only on submit and on "complete review", so it costs nothing when idle.
- It also settles OPS-07 ("does the server need its own check before live use?").

**What changes:** this reverses the 30 Sep 2026 decision "no Node service; all answer replay runs in the browser". That decision was made because the frontend was trusted, and a patient frontend isn't.

---

## 11. PROMs

### How PROMs are usually run

- **The NHS national PROMs programme** covers hip and knee replacement. It asks the **EQ-5D** (generic health status) plus the **Oxford Hip or Knee Score** before surgery and again **6 months after**. The outcome is the change in score.
- **Licences are needed, and here's why:**
  - The questionnaire engine is ours, but the instruments' **wording, answer options and scoring are copyrighted**. The licence pays for the right to reproduce them.
  - Rewording an instrument to avoid the licence makes it a different, unvalidated questionnaire, whose scores can't be compared with national or published results.
- **Known costs, as of Oct 2026.** These will be decided later and may become a customer add-on (A-18):

  | Instrument | Owner | Cost |
  | --- | --- | --- |
  | **EQ-5D-5L** | EuroQol | Free for non-commercial users after registration. Commercial users pay. The licence policy (Aug 2025) lists **about €7,000** for the UK English digital version. What that covers (per study, per year or per customer) must be confirmed. |
  | **Oxford Hip / Knee Score** | Oxford University Innovation | **Free for publicly funded healthcare** (NHS trusts). Commercial use pays a fee that isn't published. The user manual costs £150. |
  | **MMSE-2** (deferred) | PAR Inc. | About **$1–3 per form** on paper; digital is by quote. |

  - **Ask EuroQol and OUI:** can a trust's free licence cover use through our platform? If it can, the cost to us may be zero.
- **PROMs products generally** work like this:
  - a **baseline before surgery** plus follow-ups at time points after it (commonly 6 weeks, 6 months and 12 months);
  - each follow-up has a reminder and a response window;
  - results are shown as change from baseline.
- **Lifebox** has a recovery diary instead: weekly hospital-authored check-ins for 6 weeks after discharge.

### Requirements (*Proposed*)

| ID | Requirement | Priority |
| --- | --- | --- |
| PRM-01 | A **PROMs plan** belongs to a procedure (PRC-01). It's a list of: instrument, time point (**baseline** or **N days after discharge**) and response window (default 28 days). | Must |
| PRM-02 | **Baseline** forms are added to the patient's to-do list with the HQ, so there's one sitting and no extra invite. If a baseline answer already exists in the HQ (same question keys), it's pre-filled. | Must |
| PRM-03 | **Follow-ups:** at discharge date + N days, the patient gets an SMS and email, and the form appears on their home. There's one reminder after 7 days. When the window closes, the form is marked **Missed**. | Must |
| PRM-04 | The patient can opt out of PROMs follow-ups. | Must |
| PRM-05 | The episode shows each instrument's score per time point and the **change from baseline**. A follow-up much worse than baseline (a drop beyond a set threshold) adds a task for clinician review. | Should |
| PRM-06 | A per-procedure report: completion rate and average change, per hospital and consultant. CSV export. | Should |
| PRM-07 | Instruments are **platform library items with locked wording and scoring**, and each records its owner and licence. Trusts can also author their own **recovery check-ins** (free, no licence), like Lifebox's diary. | Must |
| PRM-08 | **Each licensed instrument is opt-in** (ORG-05). A platform admin sets it when adding a trust or hospital, or later: the trust opts in or out, and each hospital follows the trust or decides for itself. The admin can note who holds the licence and until when. A hospital that hasn't opted in can't add or send the instrument, and the opt-in can be sold as an add-on. | Must |

**What we build and ship:**
- **EQ-5D-5L** (5 questions + the visual analogue scale, scored with the UK value set), **Oxford Hip Score** and **Oxford Knee Score** (12 questions each, scored 0–48).
- Default plans: hip or knee replacement → EQ-5D-5L + the Oxford score at **baseline** and **6 months**; any procedure → a free **recovery check-in** at 7 days and 6 weeks.
- These are built now. A hospital can use each one only once it has opted in (PRM-08). The free check-ins need no opt-in.

**Approach:** PROMs instruments are questionnaires of kind `proms`, attached as `episode_forms` with `due_at` and `closes_at`. The `jobs` table sends them. The episode is `recovery` while any form is open, and auto-archive waits for the last one.

---

## 12. Production and compliance (before real patient data)

| ID | Requirement |
| --- | --- |
| OPS-01 | AWS **London (eu-west-2)**. UK data stays in the UK. Other data regions can come later (A-19). |
| OPS-02 | Separate QA, Training and Production AWS accounts or stacks (NFR-13). Non-production environments scale to zero. |
| OPS-03 | NHS DSPT, Cyber Essentials Plus, a DPIA template for customers, a data processing agreement (the trust is the controller, we are the processor), and a pen test (NFR-05). |
| OPS-04 | Clinical safety: DCB0129 hazard log (NFR-01), extended to cover:<ul><li>patient identity linking;</li><li>pre-fill (old answers accepted without being read; answers never expire);</li><li>**auto-triage straight to Ready for admission, with no clinician involved**. The "Auto-triaged" worklist is the mitigation;</li><li>ASA rules;</li><li>notifications;</li><li>no virus scanning.</li></ul>Customers do DCB0160. |
| OPS-05 | Alarms on errors, failed jobs and failed SMS/email deliveries. Undelivered invites are shown to the hospital. |
| OPS-06 | Break-glass support access for platform staff: time-limited and audited. |
| OPS-07 | Settled by D-1 (server-side evaluation on submit). |

Billing is out of scope. A monthly episode count per trust is enough for invoicing.

---

## 13. AWS architecture (*Proposed*, serverless, near-zero cost when idle)

```
Browser ─► Amplify Hosting (React app, no server compute) ─┬─ /*        → the app (SPA rewrite to index.html)
                                                           └─ /api/*    → reverse-proxy rewrite to API Gateway
API Gateway (HTTP API) ─┬─ /api/*    (session cookie) → staff-api Lambda (Go)
                        └─ /api/p/*                → patient-api Lambda (Go)
staff-api, patient-api ─► Postgres (Neon now, RDS later) · S3 files (presigned) · SES · SMS
                       └► evaluate Lambda (Node, packages/clinical): on submit and on complete review
EventBridge Scheduler, every 5 min ─► jobs Lambda (Go) ─► jobs table: reminders, PROMs, auto-archive
Cognito: staff user pool, managed login (password + TOTP MFA; Entra SSO later) → our own session cookie. Patients use our own magic links.
CodePipeline (CDK Pipelines, Python): GitHub → test + build → QA → Training → approval → Production
```

### Lambda layout (D-4): four functions, one Go codebase

| Function | Runtime | Serves | Why it's separate |
| --- | --- | --- | --- |
| `staff-api` | Go, arm64 | Every `/api/o/…` route | Staff traffic, checked against Cognito |
| `patient-api` | Go, arm64 | `/api/p/…` and sign-in links | It faces the public internet. It gets its own IAM role (no staff permissions) and its own concurrency cap, so abuse can't starve staff traffic or reach admin actions. |
| `jobs` | Go, arm64 | The EventBridge schedule | Background work, with no HTTP |
| `evaluate` | Node | Called by the two APIs (D-1) | A different language |

- **Why not one Lambda for everything:**
  - Patient and staff code would share one set of permissions and one point of failure.
  - The scheduled jobs would be mixed in with HTTP code.
- **Why not one Lambda per route:**
  - At low traffic, every extra function means more cold starts and more configuration.
  - AWS's own guidance (Compute Blog, 2024) recommends a middle path between the two extremes.
- **Newer Lambda features we checked, and don't need:**
  - **Managed Instances** runs Lambda on EC2 for steady, high-volume work.
  - **Durable functions** could wait weeks to send PROMs, but a jobs table is simpler and runs locally.
  - **SnapStart** doesn't support Go, and Go already starts quickly.
- **Limits that matter:**
  - A request or response can be at most 6 MB, so files go straight to S3 through presigned URLs.
  - API Gateway times out after 29 s by default, so PDF generation will run as a job later.
- **Same code locally and on Lambda:** each `cmd/<function>/main.go` wraps the same `net/http` handlers. It uses an adapter (e.g. `algnhsa`) on Lambda and `ListenAndServe` locally. For local development, one process serves both routers.

### Data regions (A-19): possible later, not built now

**Goal:** a trust outside the UK can have its patient data stored in its own AWS region, while the code keeps running in London.

**What lives where:**

| Where | What |
| --- | --- |
| **London, always** (the control plane) | All code (Lambdas, Amplify, `evaluate`), Cognito staff accounts, the platform admin data, trusts and hospitals (each with its `data_region`), staff memberships, platform content, and published questionnaires. Also **patient login accounts** (`patient_accounts`: mobile or email only) and an `account_records (account_id, org_id)` pointer that holds no clinical data. |
| **The trust's region** (a data cell) | Patient records, episodes, answers, observations, tasks, files (an S3 bucket in that region), the patient-data audit log, and the trust's jobs. |

**Rules to follow from now on**, so that adding a region later is configuration rather than a rewrite:
1. **One store per trust.** Go gets the database pool and S3 bucket through `store.ForOrg(orgID)`. Today it returns the one London pool and bucket for every trust. Later it looks up the trust's `data_region`.
2. **No SQL joins of patient data across trusts.** Every patient-data query is scoped to one trust. Anything that spans trusts loops over trusts in Go:
   - the patient's home lists records from each trust in `account_records`;
   - platform reports do the same.
3. **Control-plane tables never hold patient data.** They may hold only IDs that point into a data cell.
4. **The jobs worker loops over regions.** Each data cell has its own `jobs` table.
5. **Pre-fill across trusts in different regions** moves data between regions. It's allowed only with the patient's on-screen consent (section 6), or it's switched off for that trust.
6. **The region is chosen per trust, not per hospital.** A trust's patient record is shared by all its hospitals, so it can't be split. A hospital that needs a different region is set up as its own trust.

**Cost of keeping the door open today:**
- a `data_region` column;
- the `store.ForOrg` function;
- the discipline of rules 2 and 3.

**When a second region is added:**
- the CDK stack deploys a data cell (a Postgres database and an S3 bucket) in that region;
- `store.ForOrg` routes to it;
- queries from London to a far region are slower, and deploying the API Lambdas into that region too is a later option.

### Database (D-2): plain Postgres, cheap before customers

- **Why not RDS Postgres from day one:**
  - RDS can't scale to zero. It bills every hour, and a stopped instance restarts itself after 7 days.
  - A Lambda that talks to RDS has to sit inside a VPC.
  - From inside a VPC, reaching SES, SMS and Cognito needs a NAT gateway or VPC endpoints, and both are fixed monthly costs.
- **Until the first customer: Neon, in AWS London (`aws-eu-west-2`):**
  - It's plain Postgres: `pg_trgm` code search, sqlc and goose all work unchanged.
  - It sleeps after 5 minutes idle and wakes in about 0.3–0.5 s.
  - The free tier is enough.
  - We connect over TLS, with the connection string in SSM Parameter Store (the free standard tier). There's no VPC, so there's no NAT.
  - There's no real patient data before the first customer, so a third-party database is acceptable.
- **From the first customer: RDS for PostgreSQL** in our own AWS account, in eu-west-2.
  - The smallest Graviton instance, plus Multi-AZ when the contract needs it.
  - The Lambdas move into the VPC, and we add VPC endpoints or a NAT gateway; by then that's an accepted fixed cost.
  - Moving the data is a `pg_dump` and `pg_restore`.
- **Rejected:**
  - **Supabase:** the free tier pauses after 7 days idle and needs a manual restore. Pro is $25/month and always on.
  - **Aurora DSQL:** it has no extensions, so code search would need rewriting, and it's only a subset of Postgres.
  - **Aurora Serverless v2:** it takes about 15 s to wake from a pause, and it needs a VPC.

### Frontend (D-5): Amplify Hosting

- **Agreed:** AWS Amplify Hosting, with no hand-built S3 + CloudFront.
- **Found while checking the docs:**
  - Amplify Hosting officially supports Next.js **only up to version 15**, and we're on 16.3.
  - Every page is already a client (browser) component, so we use none of Next.js's server features.
- **Agreed 2 Oct: a plain React single-page app** (React + Vite + React Router; the Mantine UI and `packages/clinical` stay):
  - Amplify serves it from its CDN, with **no server compute**, so idle cost is near £0.
  - A documented **SPA rewrite** (`/<*>` → `/index.html`, 200) makes deep links such as `/h/123/episodes/456` work, with no route rework.
  - A documented **reverse-proxy rewrite** (`/api/<*>` → `https://<api-gateway>/api/<*>`, 200) keeps one domain, so cookies work and there's no CORS.
  - Moving from Next.js is about one session: swap `next/navigation` and `next/link` for React Router, and turn the layouts into route components.
- **Alternatives considered:**
  - **Next.js 15 on Amplify's server compute:** a downgrade, server compute we don't need, and the same version gap at every Next.js major release.
  - **Vercel:** runs Next.js 16 unchanged. But its free Hobby plan doesn't allow commercial use, so Pro costs **$20 per developer seat per month** from day one. It's a second platform outside AWS and CodePipeline, and its proxy would carry patient data, making Vercel another data processor for NHS assurance.
- **Status:** agreed. The move from Next.js is part of phase 0.

### Infrastructure and delivery (A-16)

- **CDK in Python**, in `infra/`. The stacks:
  - **Web:** the Amplify app (`aws_amplify.CfnApp` + a branch), with auto-build off, plus its rewrite rules.
  - **Api:** the four Lambdas, the HTTP API and the schedule.
  - **Auth:** the Cognito staff pool.
  - **Messaging:** the SES identity and the SMS configuration.
  - **Data:** the S3 file bucket and the SSM parameters.
  - **Network** is added only when RDS arrives.
- **CodePipeline** through CDK Pipelines (`pipelines.CodePipeline`):
  - **Source:** GitHub, through AWS CodeConnections. The pipeline updates itself.
  - **Build:** runs `make test` (Go + Vitest), builds the Go binaries for arm64 and the Node `evaluate` bundle. After the backend deploys, a step zips the frontend build and calls Amplify `StartDeployment`, so one pipeline controls the order and Amplify never builds on its own.
  - **Migrations:** goose runs as a post-deploy step against each stage's database.
  - **Stages:** **QA** only until the first customer. Then **Training** and **Production** are added, in separate AWS accounts, with a manual approval before Production.
- **Cost:** CodePipeline (V2) charges per action-minute and CodeBuild per build-minute, so pennies at our volume. **Idle target before the first customer: close to £0.** The only small fixed costs are a Route 53 hosted zone and CloudWatch log storage (we set a retention period).

---

## 14. Build order (one or two sessions per phase)

**The buildable plans are in [plans/README.md](plans/README.md)**. Phase 0 is split into foundation plans F1–F4 (tests first, then React + Vite, Lambda entry points, AWS and the pipeline), and phases 1–10 become C1–C10. The testing rules are there too.


| Phase | Delivers | Main IDs |
| --- | --- | --- |
| 0. AWS skeleton | Move the frontend from Next.js to React + Vite + React Router; CDK (Python) + CodePipeline; today's Go API split into the staff, patient and jobs Lambdas; Neon; the frontend on Amplify Hosting; the QA stage. Today's app is deployed with the stub login, and we **measure the idle cost**. | Section 13, D-2, D-4, D-5 |
| 1. Trusts and staff | Trusts (with `data_region`, London only for now), hospitals, settings inheritance and opt-ins, Cognito staff pool + MFA, roles, invites, hospital picker, audit log, and `store.ForOrg` (the data-region rules in section 13) | ORG, STF |
| 2. Patients | Magic-link sign-in, records by NHS number, SMS/email invites, patient home, `jobs` + reminders | PAT, PX-08, PX-11 |
| 3. Episode lifecycle | Procedures, `episode_forms`, new statuses and dates, worklists, tasks, cancel, archive and un-archive, auto-archive | EPI, PRC, TSK |
| 4. Question keys and pre-fill | Keys, question library, "Same as…", "Always ask fresh", pre-fill and confirmation screens | PRE |
| 5. Authoring additions | Capture blocks, Summary element, Video and Information elements, content library | CAP, SUM, INF |
| 6. Scores, ASA and auto-triage | Score bands, ASA contributions, ASA rules per HQ, `evaluate` Lambda, auto-triage | SCR, ASA, D-1 |
| 7. Observations and assessments | Observations; STOP-Bang, DASI, PRISMA-7 and Apfel pre-filled; then the rest | OBS, section 9 |
| 8. Files | Uploads both ways, files on the POA Summary | FIL |
| 9. Documents and PROMs | Document templates + PDF; PROMs plans, sends and results; EQ-5D-5L, Oxford Hip and Knee (switched on per trust); recovery check-ins | DOC, PRM |
| 10. Go-live | Move to RDS, the Training and Production stages, compliance, pen test | OPS |

**Why this order:**
- Phase 0 proves the hosting model and its cost before we build on it.
- Phases 1–3 are what make it a SaaS, and everything after depends on them.
- Question keys (phase 4) come before scores, assessments and PROMs baselines, because all three reuse them.
- Real patients wait for phase 10.

---

## 15. Open questions for Rahul

None open. Settled on 2 Oct 2026:
- magic-link sign-in (A-4);
- four Lambdas (D-4);
- Neon, then RDS (D-2);
- a React + Vite + React Router SPA on Amplify Hosting (D-5);
- licensed instruments as opt-ins (A-18);
- data regions possible later (A-19).
