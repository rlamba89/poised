-- +goose Up
-- Patient sessions held in the database (S14 answers 4, Rahul 13 Oct 2026). Each session belongs
-- to the invite link it was opened with, so Sign out (ended_at), a Resend (the link's revoked_at)
-- or a lock (its locked_at) ends it at once, even for a copied cookie. The cookie names only the
-- session's id, signed; never the patient's record. Every patient request checks and touches its
-- row (last_seen_at: the 30-minute idle timeout), so no server keeps anything in memory (Lambda).
CREATE TABLE patient_sessions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id      uuid NOT NULL REFERENCES login_links,
    patient_id   uuid NOT NULL REFERENCES patients,
    created_at   timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    ended_at     timestamptz  -- signed out
);
CREATE INDEX patient_sessions_link_idx ON patient_sessions (link_id);

-- +goose Down
DROP TABLE patient_sessions;
