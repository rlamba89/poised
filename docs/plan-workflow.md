# Plan: the episode workflow (publish → patient → clinician → POA Summary)

Agreed 1 Oct 2026. This makes sj-demo a Lifebox-like app, not only an authoring tool. Until now, [requirements.md](requirements.md) left episodes, patients filling in forms and clinical documents out of scope; they are now in.

The goal is one thin, working pass through the whole workflow. Auth, hospital setup, user and patient accounts come after it ([Later](#later)).

## The workflow

1. An author **publishes** an HQ.
2. A clinician **creates an episode** for a patient and picks the published HQ. They get a link for the patient.
3. The **patient fills in** the HQ through the link and submits it. The episode becomes **Ready for review** (the triage status).
4. A clinician **validates** each Question Set: they see the patient's answers next to the clinician questions, correct answers, and fill in their own questions. When every Question Set is validated, they **complete the HQ review** and the episode becomes **Ready for POA**.
5. The **POA Summary** shows two tabs:
   - **Validated summary:** built from the clinician's answers, which are the final answers. Answers the clinician changed are marked with what the patient said.
   - **Patient answers:** the same document, built from the patient's original answers. It's kept for audit.

## Decisions

- **Answers per actor, as Lifebox does.** Each Question Set in an episode has up to two answer sets:
  - the **patient's**, frozen when they submit;
  - the **clinician's**, which starts as a copy of the patient's when the clinician first opens the set.

  This mirrors Lifebox's `AnswerSubmission.actorType` (PATIENT or CLINICIAN). The POA tabs are its `POA_SUMMARY_POST_VALIDATION` and `POA_SUMMARY_PRE_VALIDATION`.
- **Validation is per Question Set.** "Validated by X on <date>" is recorded once per set, as in Lifebox (`ClinicalAnswersCompletedBy`).
- **Triage is only a status.** The episode becomes Ready for review when the patient submits. There's no separate triage screen.
- **Patients use a link, with no login.** `/p/<token>` opens their episode. Patient accounts come later.
- **One-click Publish.** Review and sign-off (SGN) come later.
- **An episode is tied to one published version.** Published versions are read-only, so answers always match the questions they were given against (OUT-03).
- **Go strips clinician-only content for patients.** The patient endpoint removes `clinicianOnly` elements, `clinicalOutputs` and `testCases` before sending the JSON. It also drops any answer key that isn't a patient question. This is a plain JSON walk, not expression evaluation, so the "Go never evaluates SurveyJS" rule still holds.
- **Disclosures are computed in the browser at view time.** The POA Summary runs `computeOutputs` over the published JSON plus one answer set. The version is frozen, so the result is always the same, and nothing extra is stored.
- **Patient details are made up.** This is still a local tool, so no real patient data goes in (NFR-03).

## Episode statuses

These are Lifebox's Full HQ states (`statemodel.go`), with readable keys.

| Key | Label | Lifebox state | Set by |
| --- | --- | --- | --- |
| `hq_not_complete` | HQ not complete | `primary` | Creating the episode |
| `ready_for_review` | Ready for review | `review` | The patient submitting (this is "triage") |
| `ready_for_poa` | Ready for POA | `triageready` | Completing the HQ review |
| `poa_complete` | POA complete | `pending` | The status dropdown |
| `on_hold` | On hold | `onhold` | The status dropdown |
| `not_ready` | Not ready for admission | `notready` | The status dropdown |
| `ready_for_admission` | Ready for admission | `ready` | The status dropdown |

`admitted`, `discharged`, `recovery` and `archive` come later. Every status change is written to the episode's events.

## Data (migration 00004)

```sql
-- Made-up patients only (NFR-03).
CREATE TABLE patients (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital_id     uuid NOT NULL REFERENCES hospitals,
    first_name      text NOT NULL,
    last_name       text NOT NULL,
    date_of_birth   date NOT NULL,
    sex             text NOT NULL CHECK (sex IN ('female', 'male', 'other', 'unknown')),
    hospital_number text NOT NULL DEFAULT '',
    phone           text NOT NULL DEFAULT '',
    email           text NOT NULL DEFAULT ''
);

CREATE TABLE episodes (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital_id          uuid NOT NULL REFERENCES hospitals,
    patient_id           uuid NOT NULL REFERENCES patients,
    version_id           uuid NOT NULL REFERENCES questionnaire_versions,  -- must be published
    status               text NOT NULL DEFAULT 'hq_not_complete' CHECK (status IN (…the keys above…)),
    procedure            text NOT NULL DEFAULT '',
    anaesthetic          text NOT NULL DEFAULT '',
    consultant           text NOT NULL DEFAULT '',
    nurse_asa            int CHECK (nurse_asa BETWEEN 1 AND 6),
    anaesthetist_asa     int CHECK (anaesthetist_asa BETWEEN 1 AND 6),
    patient_token        text NOT NULL UNIQUE,   -- random, 32 bytes, base64url
    patient_submitted_at timestamptz,
    review_completed_by  uuid REFERENCES users,
    review_completed_at  timestamptz,
    created_by           uuid NOT NULL REFERENCES users,
    created_at           timestamptz NOT NULL DEFAULT now()
);

-- One row per Question Set per actor. The patient row is frozen once the episode is submitted.
CREATE TABLE episode_answers (
    episode_id   uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    chapter_id   uuid NOT NULL REFERENCES chapters,
    actor        text NOT NULL CHECK (actor IN ('patient', 'clinician')),
    data         jsonb NOT NULL DEFAULT '{}',      -- SurveyJS survey.data
    updated_by   uuid REFERENCES users,            -- null for the patient
    updated_at   timestamptz NOT NULL DEFAULT now(),
    validated_at timestamptz,                      -- clinician rows only: "Validated by … on …"
    PRIMARY KEY (episode_id, chapter_id, actor)
);

-- "General notes" on the POA Summary: automatic events and clinicians' comments. Append-only.
CREATE TABLE episode_events (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_id uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('event', 'comment')),
    text       text NOT NULL,
    user_id    uuid REFERENCES users,              -- null for the patient
    created_at timestamptz NOT NULL DEFAULT now()
);
```

The `clinician` role is added to `memberships`. Seed data adds one clinician user and three made-up patients.

## Steps

Each step is committed on its own, with Go handler tests and Vitest tests as before.

### Step 0: unblock and commit
[pending.md §1](pending.md): start Docker, run `make migrate`, do the browser checks, then commit the current work.

### Step 1: publish and new version (LCY-01/06/07, minimal)
- **Publishing:**
  - `POST /api/h/{hid}/questionnaires/{qid}/publish` with `{versionId}` publishes the latest version.
  - It is for the `publisher` role, and only works on a draft that has at least one Question Set.
  - The version published before is retired, in the same statement.
- **New version:** `POST /api/h/{hid}/questionnaires/{qid}/versions` copies the published version's Question Sets into a new draft. The content is unchanged, so stable IDs and test cases carry over.
- Go already refuses content writes to a version that isn't a draft.
- **Publish checks:** `publishProblems()` in `packages/clinical` runs in the browser. It blocks publishing on logic problems, Question Set condition problems, failing test cases, or unsaved changes.
- **UI:**
  - a version and status badge in the editor header;
  - Publish, with a modal that lists the problems;
  - a "Create new version" button on published versions.
- Alex Author gets the `publisher` role in the seed data.

### Step 2: patients and episodes
- **Routes:**
  - `GET` / `POST /api/h/{hid}/patients`
  - `GET /api/h/{hid}/published-hqs`: the HQs an episode can be given
  - `GET` / `POST /api/h/{hid}/episodes`
  - `GET /api/h/{hid}/episodes/{eid}`
  - `PATCH /api/h/{hid}/episodes/{eid}`: status, ASA grades, procedure details
  - `POST /api/h/{hid}/episodes/{eid}/notes`
- All of these are for the new `clinician` role (Cara Clinician in the seed data). Clinicians land on Episodes; the header links Questionnaires and Episodes by role.
- **Episodes screen** (`/h/[hospitalId]/episodes`): a table of patient, HQ, status and date created, with a status filter.
- **"New episode":**
  - Pick a patient or add one.
  - Enter the procedure, anaesthetic and consultant.
  - Pick a **published** HQ.
  - Afterwards, the patient link is shown with a Copy button.
- The `episode_events` row "Episode created" is written.

### Step 3: the patient fills in the HQ
- **Routes** (no login; the token is the key):
  - `GET /api/p/{token}`: the episode, the patient's name and the stripped patient-audience chapters, with the patient's answers so far
  - `PUT /api/p/{token}/answers/{cid}`: autosave
  - `POST /api/p/{token}/submit`
- **Page `/p/[token]`:**
  - The Question Sets are listed as tiles with done or not done.
  - Each one opens in the patient view, with Sections as screens (STR-03). Profile shows the episode's patient.
  - Answers autosave, so the patient can leave and come back (PX-05).
  - "Submit" is enabled once every required question is answered.
- **Submitting:**
  - The patient rows are frozen and `patient_submitted_at` is set.
  - The status becomes Ready for review, and the event "HQ completed by patient" is written.
  - After that, the link shows "Thank you, your answers have been sent".

### Step 4: the clinician validates
- **Routes:**
  - `GET /api/h/{hid}/episodes/{eid}/answers`: both actors' rows
  - `PUT /api/h/{hid}/episodes/{eid}/answers/{cid}`: the clinician row. On the first write, it starts as a copy of the patient row.
  - `POST …/answers/{cid}/validate`
  - `POST /api/h/{hid}/episodes/{eid}/complete-review`
- **Episode page** (`/h/[hospitalId]/episodes/[eid]`):
  - the patient header and the status;
  - the Question Sets, with Not started, In progress or "Validated by X on <date>";
  - a link to the POA Summary.
- **Validating a Question Set:**
  - The clinician view uses the existing two-column layout, with clinician questions and clinical summary boxes. Clinician-audience Question Sets have only a clinician row.
  - It is pre-filled from the clinician row, or from the patient row if there isn't one yet.
  - An answer that differs from the patient's gets a small "Patient answered: …" note under it.
  - "Validate" saves and stamps the set.
- **Completing the review:**
  - "Complete HQ review" is enabled when every visible Question Set is validated.
  - It sets Ready for POA and writes the event "HQ review completed by X".

### Step 5: the POA Summary
`/h/[hospitalId]/episodes/[eid]/poa` follows the Lifebox page in the screenshot (layout, not styling):

- **Header:**
  - the patient's name, hospital number and phone;
  - "POA Summary" with a **Print** button (`window.print()` and print CSS);
  - the **Episode status** dropdown, at the top and again at the bottom.
- **Tabs:** **Validated summary** (clinician answers) | **Patient answers** (patient answers). Both use one renderer.
- **Patient details grid:**
  - name, hospital number, date of birth, age, phone, consultant;
  - procedure and anaesthetic;
  - patient-reported BMI (from the BMI answer);
  - nurse and anaesthetist ASA grade.
- **General notes:** the episode events with user and time, plus "Add comment".
- **One collapsible section per Question Set,** in order:
  - **For each page with a clinical summary**, a "`<page>` Summary" block. It shows "`<user>` on `<date>`", then the page's disclosure notes, then the clinician's clinical comments (`cs_comments_<page>`).
  - **For each Medication, Admissions and BMI answer**, a block with "Validated by …" and a table: name, dosage and frequency; hospital, anaesthetic, reason and year; height, weight and BMI.
  - **On the validated tab**, a disclosure note that comes from a corrected answer gets a "changed by clinician" marker. Hovering it shows the patient's answer.
  - Pages and sets with nothing to show are left out, as in Lifebox.
- **Pre-operative assessment ASA grades:** Nurse and Anaesthetist dropdowns, each with Save.

**Not in this step:** Audit log, Work plan, Observations, Investigation results, Assessments, Files, Patient files, Send to EPR, the per-section Comment buttons, and hospital-reported BMI.

### Step 6: demo script and checks
- `docs/demo-workflow.md` walks the whole workflow with the HJE Full HQ (sets 1–4 until the real export arrives).
- `spikes/ui-workflow.ts` drives it in a headless browser.
- [manual-test-plan.md](manual-test-plan.md) gets a workflow section.

## Later

In rough order, once the workflow is accepted:

1. The workflow-related items from [pending.md](pending.md):
   - the real HJE export (sets 5–11 and disclosures);
   - Question Set conditions across sets, at runtime;
   - a whole-HQ preview.
2. The rest of the POA Summary:
   - Audit log, Work plan, section comments, ASA from disclosures, Files;
   - the remaining statuses (admitted, discharged, archive).
3. The authoring Musts from pending.md:
   - review and sign-off (SGN) on top of Step 1's publish;
   - code library, export/import, audit, accessibility.
4. The platform:
   - real auth and SSO;
   - hospital creation and settings;
   - user accounts and role management;
   - patient accounts, invitations by email or SMS, and a patient search;
   - Short HQ;
   - hosting;
   - whether disclosures need a server-side check before real patients use the system.
