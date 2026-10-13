-- +goose Up
-- Staff invites (plan C1, session S09; STF-01, STF-02). An admin invites someone by email to the
-- whole trust or one hospital, with a role. The membership is added at once; the invite records
-- who invited whom and, for a new account, the 7-day deadline for its first sign-in (Cognito's
-- temporary password has the same life, S12).
--
-- An account is waiting for its first sign-in while it has an invite with no accepted_at. The
-- first sign-in accepts them all, unless every one has expired. Inviting an account that has
-- already signed in (or was seeded) is accepted when it's made: it only adds a membership.
CREATE TABLE staff_invites (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      uuid NOT NULL REFERENCES orgs ON DELETE CASCADE,
    hospital_id uuid,  -- NULL: the whole trust
    user_id     uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    role        text NOT NULL CHECK (role IN ('clinician', 'super_clinician', 'admin')),
    invited_by  uuid REFERENCES users ON DELETE SET NULL,
    created_at  timestamptz NOT NULL,
    expires_at  timestamptz NOT NULL,
    accepted_at timestamptz,
    CONSTRAINT staff_invites_hospital_fkey FOREIGN KEY (org_id, hospital_id)
        REFERENCES hospitals (org_id, id) ON DELETE CASCADE
);
CREATE INDEX staff_invites_pending_idx ON staff_invites (user_id) WHERE accepted_at IS NULL;
CREATE INDEX staff_invites_org_idx ON staff_invites (org_id);

-- One account per email, whatever its case (STF-02): Ann@X and ann@x are the same person.
ALTER TABLE users DROP CONSTRAINT users_email_key;
CREATE UNIQUE INDEX users_email_key ON users (lower(email));

-- +goose Down
DROP INDEX users_email_key;
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
DROP TABLE staff_invites;
