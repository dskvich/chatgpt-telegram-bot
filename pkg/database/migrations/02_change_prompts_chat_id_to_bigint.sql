-- +migrate Up
ALTER TABLE prompts ALTER COLUMN chat_id TYPE BIGINT;