-- +goose Up
-- Procedures (plan C3, session S19; PRC-01, EPI-01). The trust keeps a list of procedures, and an
-- episode names one of them instead of a free-text procedure. An episode also gets its planned
-- procedure date and its POA appointment date; the admitted and discharged dates come in S20.
--
-- hospital_id NULL: offered at every hospital of the trust; otherwise at that hospital only (S19
-- answers 2), as for questionnaires and option lists. A retired procedure isn't offered for new
-- episodes, but stays on the episodes that have it (answers 5): nothing is deleted.
--
-- default_questionnaire_id is the HQ chosen first when the procedure is picked on a new episode
-- (answers 6): its published version at the time. It names the questionnaire, not one version, as
-- the hospital's defaultHq setting does (S10), so publishing a new version doesn't strand it.
--
-- info_content (INF-04) and proms_plan (PRM-01) are added now (answers 3, Rahul 20 Oct) and used
-- from C5 and C9, which define their items. Until then they're empty lists.
CREATE TABLE procedures (
    id                       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                   uuid NOT NULL REFERENCES orgs ON DELETE CASCADE,
    hospital_id              uuid,
    name                     text NOT NULL CHECK (name <> ''),
    code                     text NOT NULL DEFAULT '',
    specialty                text NOT NULL DEFAULT '',
    default_questionnaire_id uuid REFERENCES questionnaires ON DELETE SET NULL,
    info_content             jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(info_content) = 'array'),
    proms_plan               jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(proms_plan) = 'array'),
    retired                  boolean NOT NULL DEFAULT false,
    updated_by               uuid REFERENCES users,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT procedures_hospital_fkey FOREIGN KEY (org_id, hospital_id) REFERENCES hospitals (org_id, id) ON DELETE CASCADE
);
-- Names are unique in the trust, in any case, retired ones included.
CREATE UNIQUE INDEX procedures_name_key ON procedures (org_id, lower(name));

ALTER TABLE episodes
    ADD COLUMN procedure_id uuid REFERENCES procedures,
    ADD COLUMN planned_date date, -- the planned procedure date
    ADD COLUMN poa_date     date; -- the POA appointment date
CREATE INDEX episodes_procedure_idx ON episodes (procedure_id);

-- The free-text procedure is replaced by procedure_id (answers 4a): each trust's distinct texts
-- become its procedures (offered at every hospital, with no default HQ), and the episodes are
-- linked to them. Migration 00019 removed every episode before it, so this carries across only
-- those made since.
INSERT INTO procedures (org_id, name)
SELECT DISTINCT ON (e.org_id, lower(btrim(e.procedure))) e.org_id, btrim(e.procedure)
FROM episodes e
WHERE btrim(e.procedure) <> ''
ORDER BY e.org_id, lower(btrim(e.procedure)), e.created_at;
UPDATE episodes e SET procedure_id = p.id
FROM procedures p
WHERE p.org_id = e.org_id AND lower(p.name) = lower(btrim(e.procedure));
ALTER TABLE episodes DROP COLUMN procedure;

-- +goose Down
ALTER TABLE episodes ADD COLUMN procedure text NOT NULL DEFAULT '';
UPDATE episodes e SET procedure = p.name FROM procedures p WHERE p.id = e.procedure_id;
ALTER TABLE episodes DROP COLUMN procedure_id, DROP COLUMN planned_date, DROP COLUMN poa_date;
DROP TABLE procedures;
