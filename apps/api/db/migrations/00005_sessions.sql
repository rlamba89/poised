-- +goose Up
-- Staff sign-in sessions (auth plan Step 2). The cookie holds a random id; only its SHA-256
-- hash is stored here. A session ends at expires_at, or after an idle gap the API sets.
CREATE TABLE sessions (
    id_hash      bytea PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL
);
CREATE INDEX sessions_user_idx ON sessions (user_id);

-- +goose Down
DROP TABLE sessions;
