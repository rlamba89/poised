-- +goose Up
-- S44c (Rahul, 28 Oct, answer 8): the patient's form saves a Question Set on its own when another
-- set's change hides answers there (it only removes answers). Such a save isn't the patient changing
-- the set, so a pre-filled set it touched can still be started fresh: Start fresh allows the
-- revision Go filled it at plus these saves.
ALTER TABLE episode_answers ADD COLUMN cleared_saves int NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE episode_answers DROP COLUMN cleared_saves;
