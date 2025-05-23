-- +goose Up
CREATE TABLE chirps(
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users (id) ON DELETE CASCADE NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULl,
    body TEXT
);


-- +goose Down
DROP TABLE chirps;