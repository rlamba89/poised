-- +goose Up
-- Pre-fill (plan C4, session S25; PRE-01–03). When the patient (or staff for them, PAT-09) first
-- opens an HQ form, Go fills each patient Question Set they haven't answered yet from their
-- earlier submitted HQs (internal/prefill), and stores the result as the patient's answers: a
-- patient row with revision 0, which their first save makes 1. "Started" (reminders and the home's
-- "In progress") is revision > 0 or a card answered (prefill_choice): an untouched pre-filled set
-- doesn't count.
--   - prefill: where each pre-filled answer came from, {key: {fromForm, answeredAt, value,
--     comment?, matchedBy}} (S25 answers 4). It never changes: the patient's answer differing from
--     `value` is "changed by patient" (S26). NULL on a row Go didn't pre-fill.
--   - prefill_choice: the opening card's button (PRE-02, S25 answers 5), recorded once per set for
--     the clinician (S26) and the clinical-safety hazard log (OPS-04): check_each ("Check each
--     answer") or still_correct ("Everything is still correct"), when, and by whom (staff on the
--     patient's behalf; NULL: the patient).
ALTER TABLE episode_answers
    ADD COLUMN prefill           jsonb CHECK (prefill IS NULL OR jsonb_typeof(prefill) = 'object'),
    ADD COLUMN prefill_choice    text CHECK (prefill_choice IN ('check_each', 'still_correct')),
    ADD COLUMN prefill_choice_at timestamptz,
    ADD COLUMN prefill_choice_by uuid REFERENCES users,
    ADD CONSTRAINT episode_answers_prefill_choice_at_check CHECK ((prefill_choice IS NULL) = (prefill_choice_at IS NULL));

-- +goose Down
ALTER TABLE episode_answers
    DROP CONSTRAINT episode_answers_prefill_choice_at_check,
    DROP COLUMN prefill_choice_by,
    DROP COLUMN prefill_choice_at,
    DROP COLUMN prefill_choice,
    DROP COLUMN prefill;
