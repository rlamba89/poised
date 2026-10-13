-- +goose Up
-- The audit log (plan C1, session S11; STF-06, AUD-01, NFR-06, NFR-07). One row per request that
-- reads or changes patient data, list screens included (Rahul, 12 Oct, S11 answers 1): who, when,
-- the trust, the hospital, what (an action and the route) and the IDs of the records it touched.
-- No patient values are stored, only IDs; names are looked up when the log is read.
--
-- Append-only: a trigger refuses UPDATE, DELETE and TRUNCATE. Nothing is deleted for now (S11
-- answers 2); later, rows go after the trust's retention period, by a migration that says so.
CREATE TABLE audit_log (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    at          timestamptz NOT NULL DEFAULT now(),
    org_id      uuid NOT NULL REFERENCES orgs,       -- never cascades: the log outlives nothing
    hospital_id uuid REFERENCES hospitals,           -- NULL: a trust route ("All hospitals")
    actor       text NOT NULL CHECK (actor IN ('staff', 'patient')),
    user_id     uuid REFERENCES users,               -- the staff member; NULL for the patient's link
    action      text NOT NULL,                       -- e.g. episode.view (internal/httpapi/audit.go)
    route       text NOT NULL,                       -- the route pattern, never the path: a link's path holds its token
    status      smallint NOT NULL,                   -- the HTTP status answered
    patient_ids uuid[] NOT NULL DEFAULT '{}',
    episode_ids uuid[] NOT NULL DEFAULT '{}',
    CHECK ((actor = 'staff') = (user_id IS NOT NULL))
);
CREATE INDEX audit_log_org_at_idx ON audit_log (org_id, at DESC);
CREATE INDEX audit_log_patients_idx ON audit_log USING gin (patient_ids);

-- +goose StatementBegin
CREATE FUNCTION audit_log_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only: % is not allowed', TG_OP;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER audit_log_no_change BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_append_only();
CREATE TRIGGER audit_log_no_truncate BEFORE TRUNCATE ON audit_log
    FOR EACH STATEMENT EXECUTE FUNCTION audit_log_append_only();

-- +goose Down
DROP TABLE audit_log;
DROP FUNCTION audit_log_append_only();
