-- +goose Up
-- S28 (Rahul, 24–25 Oct; decisions.md §2):
--
-- 1. The whole pre-fill runs once per form, on its first open (S28 answers 3): this trust's
--    answers and other trusts' alike. prefilled_at says it ran; later opens never read earlier HQs
--    again, so each source trust gets one prefill.read row per form. Narrows S25's "every open,
--    while a set has no answers". A form that already has the patient's answers (pre-filled or
--    typed) was opened before, so it counts as pre-filled; one with none is pre-filled at its next
--    open, as before.
ALTER TABLE episode_forms ADD COLUMN prefilled_at timestamptz;
UPDATE episode_forms f SET prefilled_at = now()
WHERE f.status = 'submitted' OR EXISTS (SELECT 1 FROM episode_answers a WHERE a.episode_form_id = f.id AND a.actor = 'patient');

-- 2. Another trust's audit log names only the trust, not the staff member (24 Oct): a row that
--    staff at one trust cause in another trust's log (prefill.read, when they fill in for the
--    patient) holds the staff member's trust (actor_org_id), and no user_id. Their own trust's row
--    for the same request still names them. Older rows stay as written (the log is append-only).
ALTER TABLE audit_log ADD COLUMN actor_org_id uuid REFERENCES orgs,
    DROP CONSTRAINT audit_log_check,
    ADD CONSTRAINT audit_log_check CHECK ((actor = 'staff') = (user_id IS NOT NULL OR actor_org_id IS NOT NULL)),
    ADD CONSTRAINT audit_log_actor_org_check CHECK (user_id IS NULL OR actor_org_id IS NULL);

-- +goose Down
-- The old rule can't hold for the rows written since (staff with no user_id), and they can't be
-- changed (append-only), so it comes back for new rows only (NOT VALID).
ALTER TABLE audit_log DROP CONSTRAINT audit_log_actor_org_check,
    DROP CONSTRAINT audit_log_check,
    DROP COLUMN actor_org_id;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_check CHECK ((actor = 'staff') = (user_id IS NOT NULL)) NOT VALID;
ALTER TABLE episode_forms DROP COLUMN prefilled_at;
