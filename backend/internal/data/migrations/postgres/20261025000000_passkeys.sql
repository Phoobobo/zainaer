-- +goose Up
CREATE TABLE passkeys (
 id uuid PRIMARY KEY NOT NULL,
 created_at timestamptz NOT NULL,
 updated_at timestamptz NOT NULL,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 name varchar(80) NOT NULL,
 credential_id text NOT NULL UNIQUE,
 credential bytea NOT NULL,
 version integer NOT NULL DEFAULT 0,
 last_used_at timestamptz
);
CREATE INDEX passkey_user_id ON passkeys(user_id);

-- +goose Down
DROP TABLE passkeys;
