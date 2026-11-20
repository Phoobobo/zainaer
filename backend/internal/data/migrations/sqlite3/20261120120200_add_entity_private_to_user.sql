-- +goose Up
ALTER TABLE entities ADD COLUMN private_to_user_id TEXT NULL REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX entities_private_to_user_id ON entities(private_to_user_id);

-- +goose Down
DROP INDEX IF EXISTS entities_private_to_user_id;
ALTER TABLE entities DROP COLUMN private_to_user_id;
