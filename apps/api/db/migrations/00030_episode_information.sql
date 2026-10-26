-- +goose Up
-- The patient's Information for an episode (plan C5, session S36; INF-04, PRC-01's information
-- content): videos and leaflets from the content library (S33), with the hospital's phone and email.
--
-- A procedure's list is procedures.info_content (S19): a JSON array of content item ids, in order,
-- set by the trust's whole-trust super clinicians and admins.
--
-- An episode's list is its own (episode_info_items): the procedure's list is copied onto the
-- episode when it's made (S36 answers 2, Rahul: a booked patient keeps the list from when they were
-- booked), and a clinician may add items to that one episode, or remove any.
--   from_procedure: copied from the procedure; when a clinician changes the episode's procedure,
--                   these are replaced by the new procedure's list, and a clinician's own stay;
--   position:       the order shown: the procedure's own order, then a clinician's in the order added;
--   added_by:       the clinician who added it; NULL when copied from the procedure.
-- A retired item stays in the list but isn't shown to the patient (S36 brief: it disappears from
-- every patient's Information); brought back, it shows again.
CREATE TABLE episode_info_items (
    episode_id      uuid NOT NULL REFERENCES episodes ON DELETE CASCADE,
    content_item_id uuid NOT NULL REFERENCES content_items,
    from_procedure  boolean NOT NULL,
    position        integer NOT NULL,
    added_by        uuid REFERENCES users,
    added_at        timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (episode_id, content_item_id),
    CHECK (NOT (from_procedure AND added_by IS NOT NULL))
);

-- +goose Down
DROP TABLE episode_info_items;
