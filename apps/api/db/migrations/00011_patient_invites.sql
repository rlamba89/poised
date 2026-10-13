-- +goose Up
-- Patient invites and sign-in links (plan C2, session S14; A-4, PAT-03…05, PAT-09).
--
-- A link is a 32-byte random token sent by SMS or email; only its SHA-256 is stored, so the
-- database alone can't open anyone's link. Opening one needs Continue (a POST, so a link scanner
-- can't use it) and the patient's date of birth; five wrong dates lock it. An invite link works
-- until its episode closes (no status closes an episode yet: C3 adds them), or until a resend
-- replaces it. Sign-in links (kind 'signin', PAT-06) come with S15.

CREATE TABLE login_links (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash          bytea NOT NULL UNIQUE,
    kind                text NOT NULL CHECK (kind IN ('invite', 'signin')),
    patient_id          uuid NOT NULL REFERENCES patients,
    episode_id          uuid REFERENCES episodes,  -- an invite's episode
    created_by          uuid REFERENCES users,     -- the staff member who created or resent the invite
    created_at          timestamptz NOT NULL DEFAULT now(),
    expires_at          timestamptz,               -- NULL: an invite, which lasts while its episode is open
    used_at             timestamptz,               -- the last time its date of birth was entered correctly
    revoked_at          timestamptz,               -- replaced by a newer invite (Resend)
    failed_dob_attempts int NOT NULL DEFAULT 0,
    locked_at           timestamptz,               -- after the 5th wrong date of birth
    CONSTRAINT login_links_invite_check CHECK (kind <> 'invite' OR episode_id IS NOT NULL)
);
CREATE INDEX login_links_episode_idx ON login_links (episode_id, created_at DESC);

-- The old link: the token itself, stored, opening the HQ with no date of birth. Replaced by
-- login_links; existing episodes get a link when staff resend the invite.
ALTER TABLE episodes DROP COLUMN patient_token;

-- Staff completing the HQ for the patient (PAT-09): a patient answers row whose updated_by is set
-- was entered by that staff member (it's NULL for the patient), and patient_submitted_by is who
-- sent the HQ on the patient's behalf (NULL: the patient did).
ALTER TABLE episodes ADD COLUMN patient_submitted_by uuid REFERENCES users;

-- MESSAGING=capture (local, QA): every SMS and email is stored here instead of being sent, for
-- the captured-messages page and GET /api/qa/messages. Nothing reads it in real sending (S17).
CREATE TABLE sent_messages (
    id        bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    channel   text NOT NULL CHECK (channel IN ('sms', 'email')),
    recipient text NOT NULL,              -- an E.164 mobile or an email address
    subject   text NOT NULL DEFAULT '',   -- emails only
    body      text NOT NULL,
    sent_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sent_messages_recipient_idx ON sent_messages (recipient, id DESC);

-- +goose Down
DROP TABLE sent_messages;
ALTER TABLE episodes DROP COLUMN patient_submitted_by;
ALTER TABLE episodes ADD COLUMN patient_token text;
UPDATE episodes SET patient_token = substr(replace(gen_random_uuid()::text || gen_random_uuid()::text, '-', ''), 1, 43); -- a new, random link
ALTER TABLE episodes ALTER COLUMN patient_token SET NOT NULL, ADD CONSTRAINT episodes_patient_token_key UNIQUE (patient_token);
DROP TABLE login_links;
