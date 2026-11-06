-- +goose Up
-- Assessments (plan C7, session S47; §9 "Assessments through the HQ"). An assessment, such as
-- STOP-Bang, is a platform questionnaire: no trust, no author, `assessment` naming it, with locked
-- wording built from packages/clinical's ready-made scores (internal/assessment holds the JSON).
-- Go publishes a new version of it when the wording changes; an episode's form keeps the version it
-- was given, so it always shows what the clinician saw.
--
-- A clinician adds one to an episode (S47 answers 1): an episode_forms row of kind 'assessment',
-- at most one of each per episode. The clinician's answers are the form's clinician row in
-- episode_answers, saved with the revision check; `sources` says where each item's value came from
-- (the HQ, an observation, or the clinician). Confirm is the row's validated_at, by updated_by: a
-- later save without it clears it, so a changed item needs a new Confirm. Patients never see these
-- forms: every patient route finds only the episode's HQ (kind = 'hq').
ALTER TABLE questionnaires
    ADD COLUMN assessment text CHECK (assessment ~ '^[a-z0-9]{1,40}$'),
    ALTER COLUMN org_id DROP NOT NULL,
    ALTER COLUMN created_by DROP NOT NULL,
    -- a trust's questionnaire has an author; the platform's assessments have neither trust nor author
    ADD CONSTRAINT questionnaires_owner_check CHECK (
        (assessment IS NULL AND org_id IS NOT NULL AND created_by IS NOT NULL)
        OR (assessment IS NOT NULL AND org_id IS NULL AND hospital_id IS NULL AND created_by IS NULL));
CREATE UNIQUE INDEX questionnaires_assessment_key ON questionnaires (assessment) WHERE assessment IS NOT NULL;
ALTER TABLE questionnaire_versions ALTER COLUMN updated_by DROP NOT NULL;
ALTER TABLE chapters ALTER COLUMN updated_by DROP NOT NULL;

ALTER TABLE episode_forms
    ADD COLUMN assessment text,
    ADD CONSTRAINT episode_forms_assessment_check CHECK ((kind = 'assessment') = (assessment IS NOT NULL));
CREATE UNIQUE INDEX episode_forms_assessment_key ON episode_forms (episode_id, assessment) WHERE kind = 'assessment';

ALTER TABLE episode_answers
    ADD COLUMN sources jsonb CHECK (sources IS NULL OR jsonb_typeof(sources) = 'object');

-- +goose Down
ALTER TABLE episode_answers DROP COLUMN sources;
DELETE FROM episode_forms WHERE kind = 'assessment';
DROP INDEX episode_forms_assessment_key;
ALTER TABLE episode_forms DROP CONSTRAINT episode_forms_assessment_check, DROP COLUMN assessment;
DELETE FROM questionnaires WHERE assessment IS NOT NULL;
DROP INDEX questionnaires_assessment_key;
ALTER TABLE chapters ALTER COLUMN updated_by SET NOT NULL;
ALTER TABLE questionnaire_versions ALTER COLUMN updated_by SET NOT NULL;
ALTER TABLE questionnaires
    DROP CONSTRAINT questionnaires_owner_check,
    DROP COLUMN assessment,
    ALTER COLUMN created_by SET NOT NULL,
    ALTER COLUMN org_id SET NOT NULL;
