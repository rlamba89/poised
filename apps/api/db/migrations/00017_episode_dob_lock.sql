-- +goose Up
-- Wrong dates of birth are counted per episode, not per link (S16 test report 1, X-1). S14's rule
-- was "five wrong dates lock the link until staff resend"; S16's reminders add links beside the
-- invite, so per-link counts would give five tries per link. Now an episode's live invite links
-- share the five tries, and the fifth locks them all (login_links.locked_at on each, so every
-- existing check of a link's lock, and of a session opened from it, still applies). A right date
-- starts the count again; only a Resend (new link, older ones revoked) clears a lock.
-- Sign-in links (S15) keep their own count: they belong to a contact, not an episode.
ALTER TABLE episodes ADD COLUMN dob_failed_attempts int NOT NULL DEFAULT 0;

-- Existing episodes: the most tries any live invite link has used, and a lock on one locks all.
UPDATE episodes e SET dob_failed_attempts = l.tries
FROM (SELECT episode_id, max(failed_dob_attempts) AS tries FROM login_links
      WHERE kind = 'invite' AND revoked_at IS NULL GROUP BY episode_id) l
WHERE l.episode_id = e.id;
UPDATE login_links l SET locked_at = locked.at
FROM (SELECT episode_id, max(locked_at) AS at FROM login_links
      WHERE kind = 'invite' AND revoked_at IS NULL AND locked_at IS NOT NULL GROUP BY episode_id) locked
WHERE l.episode_id = locked.episode_id AND l.kind = 'invite' AND l.revoked_at IS NULL AND l.locked_at IS NULL;

-- +goose Down
ALTER TABLE episodes DROP COLUMN dob_failed_attempts;
