-- name: CreateUser :one
INSERT INTO users (name, email, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserRole :one 
SELECT role FROM users
WHERE id = $1
LIMIT 1;

-- name: EmailExists :one
SELECT EXISTS (
    SELECT 1 FROM users WHERE email = $1
) AS exists;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2
WHERE id = $1;

-- name: UpsertOAuthUser :one
INSERT INTO users (name, email, oauth_provider, oauth_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (email)
DO UPDATE SET name = EXCLUDED.name
RETURNING *;


-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (
    token_id,
    user_id,
    token_hash,
    expires_at,
    user_agent,
    ip_address
)
VALUES ($1, $2, $3, $4,$5,$6);

-- name: RefreshTokenExists :one
SELECT EXISTS (
    SELECT 1 FROM refresh_tokens
    WHERE user_id = $1
    AND token_hash = $2
    AND expires_at > NOW()
) AS exists;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens
WHERE token_hash = $1;

-- name: DeleteAllRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE user_id = $1;

-- name: DeleteAllRefreshTokensExcept :exec
DELETE FROM refresh_tokens
WHERE user_id = $1
AND token_id != $2;

-- name: UpsertPasswordResetToken :exec
INSERT INTO password_reset_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (user_id)
DO UPDATE SET
    token = EXCLUDED.token,
    expires_at = EXCLUDED.expires_at;

-- name: GetUserIDByResetToken :one
SELECT user_id FROM password_reset_tokens
WHERE token = $1
AND expires_at > NOW()
LIMIT 1;

-- name: DeletePasswordResetToken :exec
DELETE FROM password_reset_tokens
WHERE user_id = $1;

-- name: GetUserSessions :many
SELECT
    token_id,
    user_agent,
    ip_address,
    created_at,
    expires_at
FROM refresh_tokens
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: DeleteSession :exec
DELETE FROM refresh_tokens
WHERE token_id = $1
AND user_id = $2;
