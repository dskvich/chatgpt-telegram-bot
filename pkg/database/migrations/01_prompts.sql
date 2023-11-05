-- +migrate Up
CREATE TABLE prompts (
    id SERIAL PRIMARY KEY,
    message_id INT,
    chat_id INT,
    text TEXT,
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);