-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (user_id, family_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4);

-- name: GetRefreshToken :one
SELECT rt.id, rt.user_id, rt.family_id, rt.expires_at, rt.used_at, rt.revoked_at, u.role, u.carrier_id
FROM refresh_tokens rt
JOIN users u ON u.id = rt.user_id
WHERE rt.token_hash = $1;

-- Marks the token used only if nobody did it first, so two refreshes racing
-- with the same token cannot both succeed.
-- name: UseRefreshToken :execrows
UPDATE refresh_tokens
SET used_at = now()
WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL;

-- name: RevokeRefreshFamily :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE family_id = $1 AND revoked_at IS NULL;
