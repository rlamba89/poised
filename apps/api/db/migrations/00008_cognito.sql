-- +goose Up
-- Staff sign-in with Cognito (auth-first Step 4): a user is matched by Cognito's stable subject,
-- saved when they are invited or, for people who existed before, at their first sign-in by email.
ALTER TABLE users ADD COLUMN cognito_sub text UNIQUE;
CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email));

-- +goose Down
DROP INDEX users_email_lower_key;
ALTER TABLE users DROP COLUMN cognito_sub;
