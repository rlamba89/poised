-- +goose Up
-- Trusts own hospitals (plan C1, decision A-2). Memberships move to the role ladder
-- clinician < super_clinician < admin (A-5), held for a whole trust or one hospital.

CREATE TABLE orgs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    code        text NOT NULL UNIQUE,
    status      text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
    data_region text NOT NULL DEFAULT 'eu-west-2',  -- A-19: London only for now
    settings    jsonb NOT NULL DEFAULT '{}'
);

-- Hospitals created before trusts existed go into one placeholder trust; the seed (or, later,
-- the platform admin) moves each to its real trust.
INSERT INTO orgs (id, name, code)
SELECT '00000000-0000-4000-8000-0000000000f0', 'Hospitals from before trusts', 'PRE-TRUSTS'
WHERE EXISTS (SELECT 1 FROM hospitals);

ALTER TABLE hospitals ADD COLUMN org_id uuid REFERENCES orgs;
UPDATE hospitals SET org_id = '00000000-0000-4000-8000-0000000000f0';
ALTER TABLE hospitals ALTER COLUMN org_id SET NOT NULL;
ALTER TABLE hospitals ADD CONSTRAINT hospitals_id_org_key UNIQUE (id, org_id);
CREATE INDEX hospitals_org_idx ON hospitals (org_id);

ALTER TABLE memberships RENAME TO memberships_old;
ALTER TABLE memberships_old RENAME CONSTRAINT memberships_pkey TO memberships_old_pkey;

CREATE TABLE memberships (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    org_id      uuid NOT NULL REFERENCES orgs ON DELETE CASCADE,
    hospital_id uuid,  -- NULL: the whole trust
    role        text NOT NULL CHECK (role IN ('clinician', 'super_clinician', 'admin')),
    -- One role per user and scope; the hospital must belong to the same trust. Moving a
    -- hospital to another trust moves its memberships with it.
    CONSTRAINT memberships_scope_key UNIQUE NULLS NOT DISTINCT (user_id, org_id, hospital_id),
    CONSTRAINT memberships_hospital_fkey FOREIGN KEY (hospital_id, org_id)
        REFERENCES hospitals (id, org_id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE INDEX memberships_user_idx ON memberships (user_id);

-- Old roles onto the ladder, keeping the highest per user and hospital:
-- author, reviewer, publisher → super_clinician; hospital_admin → admin; viewer, clinician → clinician.
INSERT INTO memberships (user_id, org_id, hospital_id, role)
SELECT m.user_id, h.org_id, m.hospital_id,
       (ARRAY['clinician', 'super_clinician', 'admin'])[max(CASE m.role
           WHEN 'hospital_admin' THEN 3
           WHEN 'author' THEN 2 WHEN 'reviewer' THEN 2 WHEN 'publisher' THEN 2
           ELSE 1 END)]
FROM memberships_old m
JOIN hospitals h ON h.id = m.hospital_id
GROUP BY m.user_id, h.org_id, m.hospital_id;

DROP TABLE memberships_old;

-- +goose Down
ALTER TABLE memberships RENAME TO memberships_ladder;
-- Free the index name, so the old table gets memberships_pkey back (Up renames it by that name).
ALTER TABLE memberships_ladder RENAME CONSTRAINT memberships_pkey TO memberships_ladder_pkey;

CREATE TABLE memberships (
    user_id     uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    hospital_id uuid NOT NULL REFERENCES hospitals ON DELETE CASCADE,
    role        text NOT NULL CHECK (role IN ('viewer', 'author', 'reviewer', 'publisher', 'hospital_admin', 'clinician')),
    PRIMARY KEY (user_id, hospital_id, role)
);

-- Back to the old roles: trust-wide memberships become one per hospital in the trust.
INSERT INTO memberships (user_id, hospital_id, role)
SELECT DISTINCT m.user_id, h.id, r.old
FROM memberships_ladder m
JOIN hospitals h ON h.org_id = m.org_id AND (m.hospital_id IS NULL OR h.id = m.hospital_id)
JOIN (VALUES ('clinician', 'clinician'),
             ('super_clinician', 'clinician'), ('super_clinician', 'author'), ('super_clinician', 'publisher'),
             ('admin', 'clinician'), ('admin', 'author'), ('admin', 'publisher'), ('admin', 'hospital_admin')
     ) AS r(ladder, old) ON r.ladder = m.role;

DROP TABLE memberships_ladder;
DROP INDEX hospitals_org_idx;
ALTER TABLE hospitals DROP CONSTRAINT hospitals_id_org_key;
ALTER TABLE hospitals DROP COLUMN org_id;
DROP TABLE orgs;
