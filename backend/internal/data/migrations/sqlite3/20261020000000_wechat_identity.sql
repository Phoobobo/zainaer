-- +goose Up
ALTER TABLE users ADD COLUMN wechat_identity text;
CREATE UNIQUE INDEX user_wechat_identity ON users (wechat_identity);

-- +goose Down
DROP INDEX user_wechat_identity;
ALTER TABLE users DROP COLUMN wechat_identity;
