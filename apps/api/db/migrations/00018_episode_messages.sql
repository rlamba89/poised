-- +goose Up
-- Every text and email sent to a patient about an episode (plan C2, session S17; OPS-05): the
-- invite, a Resend and each reminder, one row per message. A message that couldn't be sent, or
-- that AWS later reports as undelivered (a bounced email, a text the network couldn't deliver),
-- gets failed_at and a reason for staff, and the episode shows the invite as undelivered.
--
-- provider_id is the sender's message ID (SES's or End User Messaging SMS's; "capture-<n>" for a
-- captured message), which the delivery events name. NULL when the send itself failed.
CREATE TABLE episode_messages (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_id  uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    kind        text NOT NULL CHECK (kind IN ('invite', 'reminder')), -- an invite or Resend, or a reminder
    channel     text NOT NULL CHECK (channel IN ('sms', 'email')),
    provider_id text,
    created_at  timestamptz NOT NULL,   -- when it was sent: the same for a send's text and email
    failed_at   timestamptz,
    failure     text NOT NULL DEFAULT '' -- why, in words for staff
);
CREATE UNIQUE INDEX episode_messages_provider_key ON episode_messages (provider_id) WHERE provider_id IS NOT NULL;
CREATE INDEX episode_messages_episode_idx ON episode_messages (episode_id, created_at);

-- +goose Down
DROP TABLE episode_messages;
