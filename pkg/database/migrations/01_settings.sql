-- +migrate Up
CREATE TABLE settings (
    chat_id BIGINT NOT NULL,
    topic_id INTEGER NOT NULL,
    text_model VARCHAR,
    system_prompt VARCHAR,
    image_model VARCHAR,
    ttl BIGINT DEFAULT 0,
    PRIMARY KEY (chat_id, topic_id)
);