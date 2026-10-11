-- +goose Up
-- Each answers row counts its saves, so a save from a stale tab or device gets a 409 instead of
-- silently overwriting newer answers (S02a). A save sends the revision it started from. Rows saved
-- before this migration are at 0, which is also what a browser sends for a set it hasn't saved yet.
ALTER TABLE episode_answers ADD COLUMN revision int NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE episode_answers DROP COLUMN revision;
