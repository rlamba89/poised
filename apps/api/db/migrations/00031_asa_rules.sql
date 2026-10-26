-- +goose Up
-- An HQ's ASA rules (plan C6, session S40; ASA-02, ASA-03): stored with each questionnaire
-- version, so they are versioned and published with it. One JSON object (internal/asarules):
--   rules:             the rules in order, each a "when" (the suggested ASA compared with a grade,
--                      and optional extras) and a "then" (a fixed list of actions);
--   noGradeAutoTriage: whether a patient whose answers give no ASA grade counts as ASA I and so can
--                      be auto-triaged (S40 answers 2: a setting per HQ, on by default);
--   testCases:         whole-HQ test cases (S40 answers 4), checking which rules fire.
-- Go checks only the shape; packages/clinical decides which rules fire (from S41, on the server
-- through the evaluate service). NULL: a version published before rules existed; it has none.
-- rules_revision is bumped on every save, so a stale save is refused (as chapters.revision).
ALTER TABLE questionnaire_versions
    ADD COLUMN rules          jsonb,
    ADD COLUMN rules_revision int NOT NULL DEFAULT 0;

-- Drafts get the platform's default rules (internal/asarules/defaults.json as on 26 Oct 2026),
-- as a new version of an older HQ does (S40 answers 1). Published and retired versions keep none.
UPDATE questionnaire_versions SET rules = '{
  "noGradeAutoTriage": true,
  "rules": [
    {"id": "low_risk", "name": "Low risk",
     "when": {"asa": {"op": "<=", "grade": 2}, "noRedFlags": true},
     "then": [{"kind": "autoTriage", "target": "ready_for_admission"}]},
    {"id": "high_risk", "name": "High risk",
     "when": {"asa": {"op": ">=", "grade": 3}},
     "then": [{"kind": "task", "title": "Book anaesthetic review", "team": "anaesthetist"},
              {"kind": "flag", "name": "Anaesthetic review"},
              {"kind": "requireAnaesthetistAsa"}]}
  ],
  "testCases": []
}'::jsonb
WHERE status = 'draft' AND rules IS NULL;

-- +goose Down
ALTER TABLE questionnaire_versions DROP COLUMN rules, DROP COLUMN rules_revision;
