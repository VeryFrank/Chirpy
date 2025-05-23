-- name: CreateChirp :one
INSERT INTO chirps (id, user_id, created_at, updated_at, body)
VALUES(gen_random_uuid(), $1, NOW(), NOW(), $2)
RETURNING *;

-- name: GetAllChirps :many
SELECT * FROM chirps
ORDER BY user_id, created_at DESC;

-- name: GetChirpForUser :many
SELECT * FROM chirps
WHERE user_id=$1
ORDER BY created_at DESC;

-- name: GetChirp :one
SELECT * FROM chirps
WHERE id=$1;