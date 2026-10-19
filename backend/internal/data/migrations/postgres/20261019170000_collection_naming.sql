-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
-- +goose Up
ALTER TABLE users ADD COLUMN collection_naming_id UUID NULL;
