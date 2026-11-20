-- +goose Up
ALTER TABLE user_groups ADD COLUMN privileged BOOLEAN NOT NULL DEFAULT false;
-- Owners are always privileged; backfill so stored state matches authorization.
UPDATE user_groups SET privileged = true WHERE role = 'owner';

-- +goose Down
ALTER TABLE user_groups DROP COLUMN privileged;
