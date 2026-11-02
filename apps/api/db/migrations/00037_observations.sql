-- +goose Up
-- Observations (plan C7, session S46; OBS-01–03): a clinician's readings on an episode, one row per
-- reading. A reading is never changed or deleted (S46 answers 2, as S44b's ASA grades):
--   - a correction is a new row (corrects = the old one's id, with the reason), and the old row is
--     marked superseded_by the new one;
--   - a reading taken by mistake is marked entered in error, with who, when and why.
-- Either way the old row stays in the history, struck through, and no longer counts.
--
-- value (and value2, a blood pressure's diastolic) is in the kind's stored unit, which unit names
-- (internal/observation); ECG and urinalysis are notes (note, no value). BMI isn't stored: it's
-- worked out from the latest height and weight that count. The kinds, units and limits are Go's
-- (internal/observation, with the trust's own ranges from its settings).
CREATE TABLE observations (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_id    uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    kind          text NOT NULL CHECK (kind ~ '^[a-zA-Z0-9]{1,40}$'),
    value         double precision CHECK (value >= 0),
    value2        double precision CHECK (value2 >= 0),
    unit          text CHECK (unit IS NULL OR length(unit) BETWEEN 1 AND 20),
    note          text CHECK (note IS NULL OR length(note) BETWEEN 1 AND 2000),
    taken_at      timestamptz NOT NULL,
    entered_by    uuid NOT NULL REFERENCES users,
    entered_at    timestamptz NOT NULL DEFAULT now(),
    corrects      uuid REFERENCES observations,
    reason        text CHECK (reason IS NULL OR length(reason) BETWEEN 1 AND 1000),
    superseded_by uuid REFERENCES observations,
    error_by      uuid REFERENCES users,
    error_at      timestamptz,
    error_reason  text CHECK (error_reason IS NULL OR length(error_reason) BETWEEN 1 AND 1000),
    -- A measured reading has a value and a unit; a note has neither.
    CONSTRAINT observations_measured_or_note CHECK ((value IS NULL) = (unit IS NULL) AND (value IS NULL) = (note IS NOT NULL)
        AND (value2 IS NULL OR value IS NOT NULL)),
    CONSTRAINT observations_correction_reason CHECK ((corrects IS NULL) = (reason IS NULL)),
    CONSTRAINT observations_error_fields CHECK ((error_by IS NULL) = (error_at IS NULL)
        AND (error_at IS NULL) = (error_reason IS NULL)),
    -- Superseded or entered in error, never both.
    CONSTRAINT observations_counts_once CHECK (superseded_by IS NULL OR error_at IS NULL)
);
CREATE INDEX observations_episode_idx ON observations (episode_id, taken_at);
-- A reading is corrected once: only one row may correct it.
CREATE UNIQUE INDEX observations_corrects_idx ON observations (corrects) WHERE corrects IS NOT NULL;

-- No edit: the only change a row may take is being superseded, or marked entered in error, once.
-- +goose StatementBegin
CREATE FUNCTION observations_no_edit() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.id <> OLD.id OR NEW.episode_id <> OLD.episode_id OR NEW.kind <> OLD.kind
       OR NEW.value IS DISTINCT FROM OLD.value OR NEW.value2 IS DISTINCT FROM OLD.value2
       OR NEW.unit IS DISTINCT FROM OLD.unit OR NEW.note IS DISTINCT FROM OLD.note
       OR NEW.taken_at <> OLD.taken_at OR NEW.entered_by <> OLD.entered_by OR NEW.entered_at <> OLD.entered_at
       OR NEW.corrects IS DISTINCT FROM OLD.corrects OR NEW.reason IS DISTINCT FROM OLD.reason
       OR OLD.superseded_by IS NOT NULL OR OLD.error_at IS NOT NULL THEN
        RAISE EXCEPTION 'an observation can''t be changed, only corrected or marked entered in error once';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER observations_no_edit BEFORE UPDATE ON observations
    FOR EACH ROW EXECUTE FUNCTION observations_no_edit();

-- Nor deleted: only the episode's own deletion takes its readings with it.
-- +goose StatementBegin
CREATE FUNCTION observations_no_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM episodes WHERE id = OLD.episode_id) THEN
        RAISE EXCEPTION 'an observation can''t be deleted';
    END IF;
    RETURN OLD;
END $$;
-- +goose StatementEnd
CREATE TRIGGER observations_no_delete BEFORE DELETE ON observations
    FOR EACH ROW EXECUTE FUNCTION observations_no_delete();

-- +goose Down
DROP TABLE observations;
DROP FUNCTION observations_no_edit();
DROP FUNCTION IF EXISTS observations_no_delete();
