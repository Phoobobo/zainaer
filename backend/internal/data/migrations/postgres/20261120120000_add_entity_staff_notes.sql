-- +goose Up
ALTER TABLE entities ADD COLUMN staff_notes VARCHAR(1000) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE entities DROP COLUMN staff_notes;
