-- +migrate Up
CREATE TABLE chat_styles (
     id SERIAL PRIMARY KEY,
     chat_id BIGINT NOT NULL,
     name VARCHAR,
     description TEXT,
     is_active BOOLEAN DEFAULT FALSE,
     created_by TEXT,
     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
     UNIQUE (chat_id, name)
);