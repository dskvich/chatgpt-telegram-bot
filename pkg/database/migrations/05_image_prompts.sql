-- +migrate Up
CREATE TABLE image_prompts (
   id SERIAL PRIMARY KEY,
   prompt TEXT NOT NULL,
   created_by TEXT,
   created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
   image_id SERIAL UNIQUE
);