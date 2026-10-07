CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS posts (
    id         bigserial PRIMARY KEY,
    source     text UNIQUE NOT NULL,
    content    text NOT NULL,
    embedding  vector(768) NOT NULL
);
