-- +goose Up
CREATE TABLE passkeys (
 id text PRIMARY KEY NOT NULL,
 created_at datetime NOT NULL,
 updated_at datetime NOT NULL,
 user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 name varchar(80) NOT NULL,
 credential_id text NOT NULL UNIQUE,
 credential blob NOT NULL,
 version integer NOT NULL DEFAULT 0,
 last_used_at datetime
);
CREATE INDEX passkey_user_id ON passkeys(user_id);

-- +goose Down
DROP TABLE passkeys;
