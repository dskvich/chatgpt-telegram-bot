-- +migrate Up
CREATE TABLE settings (
    chat_id BIGINT,
    key VARCHAR,
    value TEXT,
    PRIMARY KEY (chat_id, key)
);