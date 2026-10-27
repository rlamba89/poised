-- +goose Up
-- What a sent form's answers give (plan C6, session S41; D-1), worked out by the evaluate service
-- (packages/clinical through apps/evaluate) when the patient, or staff for them, sends it. Go never
-- evaluates SurveyJS; it stores the service's reply whole, as JSON: the Question Sets shown, each
-- one's disclosures and scores, the suggested ASA with its reasons, the flags, the rules that fire
-- and whether each of the patient's sets is complete (internal/evaluate's Result). S42 and S43 act
-- on it. Nothing acts on it yet.
--
-- One row per send (a form re-opened and sent again gets another), newest last. A call that failed
-- (the service down, too slow or a bad reply) stores the error instead of a result, so the episode
-- still goes to Ready for review, with a General notes line saying the checks didn't run. The error
-- never holds an answer. The answers themselves stay in episode_answers.
CREATE TABLE form_outputs (
    id                bigserial PRIMARY KEY,
    episode_form_id   uuid NOT NULL REFERENCES episode_forms ON DELETE CASCADE,
    result            jsonb, -- the service's reply; NULL when the call failed
    evaluator_version text,  -- the build that worked it out (the reply's "evaluator"); NULL when it failed
    error             text,  -- why the call failed; NULL when it worked
    created_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT form_outputs_result_or_error CHECK ((result IS NULL) <> (error IS NULL)),
    CONSTRAINT form_outputs_version_with_result CHECK ((result IS NULL) = (evaluator_version IS NULL))
);
CREATE INDEX form_outputs_form_idx ON form_outputs (episode_form_id, id);

-- +goose Down
DROP TABLE form_outputs;
