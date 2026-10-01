-- +goose Up
-- Reusable option lists (OPT-07): saved once per hospital, copied into questions with new IDs.
CREATE TABLE option_lists (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    hospital_id uuid NOT NULL REFERENCES hospitals ON DELETE CASCADE,
    name        text NOT NULL,
    options     jsonb NOT NULL,  -- [{text, score?, special?, isExclusive?, clinicalOutputs?}]
    updated_by  uuid NOT NULL REFERENCES users,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (hospital_id, name)
);

-- +goose Down
DROP TABLE option_lists;
