-- +migrate Up
CREATE TABLE settings (
    chat_id BIGINT PRIMARY KEY,
    text_model VARCHAR,
    system_prompt VARCHAR,
    image_model VARCHAR,
    ttl INT DEFAULT 0
);