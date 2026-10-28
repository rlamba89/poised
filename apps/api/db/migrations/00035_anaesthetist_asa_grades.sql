-- +goose Up
-- The anaesthetist's ASA grade as a history (plan C6, session S44b; Rahul, 27 Oct). A grade, once
-- given, is never changed or cleared, in any status: each one is kept with who gave it and when; a
-- later grade is added on top (with a reason, unless no grade counts yet), and the earlier stays
-- visible. A grade given by mistake is marked "entered in error" with a reason (who, when and why
-- are kept): it no longer counts, and the one before it (if any) counts again.
--
-- The grade that counts is the latest one not entered in error. It replaces
-- episodes.anaesthetist_asa: GetEpisode works it out from this table, so the episode's JSON keeps
-- anaesthetistAsa; the locked paths read it after LockEpisodeStatus (CountingAnaesthetistGrade).
--
-- given_by is NULL only for a grade carried over from episodes.anaesthetist_asa below ("Recorded
-- before the history"); its time is that of the General notes line that set it, else now.
CREATE TABLE anaesthetist_asa_grades (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_id   uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    grade        int NOT NULL CHECK (grade BETWEEN 1 AND 6),
    given_by     uuid REFERENCES users,
    given_at     timestamptz NOT NULL DEFAULT now(),
    reason       text CHECK (reason IS NULL OR length(reason) BETWEEN 1 AND 1000),
    error_by     uuid REFERENCES users,
    error_at     timestamptz,
    error_reason text CHECK (error_reason IS NULL OR length(error_reason) BETWEEN 1 AND 1000),
    CONSTRAINT anaesthetist_asa_grades_error_check CHECK ((error_by IS NULL) = (error_at IS NULL)
        AND (error_at IS NULL) = (error_reason IS NULL))
);
CREATE INDEX anaesthetist_asa_grades_episode_idx ON anaesthetist_asa_grades (episode_id, given_at);

-- No edit, anywhere: the only change a grade's row may ever take is being marked entered in error,
-- once. Anything else is refused here too, whatever the code asks.
-- +goose StatementBegin
CREATE FUNCTION anaesthetist_asa_grades_no_edit() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.id <> OLD.id OR NEW.episode_id <> OLD.episode_id OR NEW.grade <> OLD.grade
       OR NEW.given_by IS DISTINCT FROM OLD.given_by OR NEW.given_at <> OLD.given_at
       OR NEW.reason IS DISTINCT FROM OLD.reason OR OLD.error_at IS NOT NULL THEN
        RAISE EXCEPTION 'an anaesthetist''s ASA grade can''t be changed, only marked entered in error once';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER anaesthetist_asa_grades_no_edit BEFORE UPDATE ON anaesthetist_asa_grades
    FOR EACH ROW EXECUTE FUNCTION anaesthetist_asa_grades_no_edit();

-- Nor deleted: only the episode's own deletion takes its grades with it (ON DELETE CASCADE runs
-- after the episode's row is gone, so the episode no longer exists here).
-- +goose StatementBegin
CREATE FUNCTION anaesthetist_asa_grades_no_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM episodes WHERE id = OLD.episode_id) THEN
        RAISE EXCEPTION 'an anaesthetist''s ASA grade can''t be deleted';
    END IF;
    RETURN OLD;
END $$;
-- +goose StatementEnd
CREATE TRIGGER anaesthetist_asa_grades_no_delete BEFORE DELETE ON anaesthetist_asa_grades
    FOR EACH ROW EXECUTE FUNCTION anaesthetist_asa_grades_no_delete();

INSERT INTO anaesthetist_asa_grades (episode_id, grade, given_at)
SELECT e.id, e.anaesthetist_asa,
       coalesce((SELECT max(ev.created_at) FROM episode_events ev
                 WHERE ev.episode_id = e.id AND ev.text LIKE 'Anaesthetist ASA grade set to %'), now())
FROM episodes e
WHERE e.anaesthetist_asa IS NOT NULL;

ALTER TABLE episodes DROP COLUMN anaesthetist_asa;

-- +goose Down
ALTER TABLE episodes ADD COLUMN anaesthetist_asa int CHECK (anaesthetist_asa BETWEEN 1 AND 6);
UPDATE episodes e SET anaesthetist_asa = (
    SELECT g.grade FROM anaesthetist_asa_grades g WHERE g.episode_id = e.id AND g.error_at IS NULL
    ORDER BY g.given_at DESC, g.id DESC LIMIT 1);
DROP TABLE anaesthetist_asa_grades;
DROP FUNCTION anaesthetist_asa_grades_no_edit();
DROP FUNCTION IF EXISTS anaesthetist_asa_grades_no_delete();
