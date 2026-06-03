-- name: GetLocation :one
SELECT *
FROM
    locations
WHERE
    id = $1;

-- name: GetLocations :many
SELECT *
FROM
    locations
ORDER BY
    name;

-- name: UpsertLocation :exec
INSERT INTO
    locations (id, name, description, parent_location_id)
VALUES
    ($1, $2, $3, $4) ON CONFLICT (id) DO
UPDATE
SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    parent_location_id = EXCLUDED.parent_location_id;

-- name: ListSamples :many
SELECT
    samples.*,
    COALESCE(
        (
            SELECT
                STRING_AGG(sample_mods.name, ', ')
            FROM
                sample_mods
            WHERE
                sample_mods.sample_id = samples.id
                AND sample_mods.time_removed IS NULL
        ),
        ''
    ) AS current_mods_summary,
    users.name AS owner_name
FROM
    samples
LEFT JOIN users ON samples.owner_id = users.id
ORDER BY
    time_registered;

-- name: ListSampleMods :many
SELECT
    *
FROM
    sample_mods
WHERE
    sample_mods.sample_id = $1
ORDER BY
    time_added;

-- name: AddSampleMod :exec
INSERT INTO
    sample_mods (id, sample_id, name, time_added, time_removed)
VALUES
    ($1, $2, $3, $4, NULL);

-- name: RemoveSampleMod :exec
UPDATE sample_mods
SET
    time_removed = $1
WHERE
    id = $2;

-- name: DeleteProductByID :exec
DELETE FROM products
WHERE
    id = $1;

-- name: DeleteLocationByID :exec
DELETE FROM locations
WHERE
    id = $1;

-- name: GetProducts :many
SELECT
    *
FROM
    products
ORDER BY
    name;

-- name: UpdateOrCreateSample :one
INSERT INTO
    samples (
        id,
        location_id,
        product_id,
        time_registered,
        last_update,
        state,
        owner_id,
        product_issue
    )
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (id) DO
UPDATE
SET
    location_id = EXCLUDED.location_id,
    product_id = EXCLUDED.product_id,
    last_update = EXCLUDED.last_update,
    owner_id = EXCLUDED.owner_id,
    product_issue = EXCLUDED.product_issue,
    state = EXCLUDED.state RETURNING *;

-- name: GetSampleById :one
SELECT
    *
FROM
    samples
WHERE
    id = $1;

-- name: ListProducts :many
SELECT
    *
FROM
    products
ORDER BY
    name;

-- name: GetProductByID :one
SELECT *
FROM
    products
WHERE
    id = $1;

-- name: UpsertProduct :exec
INSERT INTO
    products (id, name, parent_product_id, part_number)
VALUES
    ($1, $2, $3, $4) ON CONFLICT (id) DO
UPDATE
SET
    name = EXCLUDED.name,
    parent_product_id = EXCLUDED.parent_product_id,
    part_number = EXCLUDED.part_number;


-- name: GetUserByID :one
SELECT *
FROM
    users
WHERE
    id = $1;

-- name: GetUsers :many
SELECT *
FROM
    users
ORDER BY
    name;

-- name: UpsertUser :exec
INSERT INTO
    users (id, name)
VALUES
    ($1, $2) ON CONFLICT (id) DO
UPDATE
SET
    name = EXCLUDED.name;

-- name: DeleteUserByID :exec
DELETE FROM users
WHERE
    id = $1;

-- name: AnyAdminExists :one
SELECT EXISTS(
        SELECT 1
        FROM user_roles
        WHERE role = 'admin'
);

-- name: CountAdmins :one
SELECT COUNT(*)
FROM user_roles
WHERE role = 'admin';

-- name: HasRole :one
SELECT EXISTS(
        SELECT 1
        FROM user_roles
        WHERE user_id = $1
            AND role = $2
);

-- name: ListUserRoles :many
SELECT role
FROM user_roles
WHERE user_id = $1
ORDER BY role;

-- name: SetUserRole :exec
INSERT INTO user_roles (user_id, role)
VALUES ($1, $2)
ON CONFLICT (user_id, role) DO NOTHING;

-- name: RemoveUserRole :exec
DELETE FROM user_roles
WHERE user_id = $1
    AND role = $2;

-- name: ListUsersWithoutAdmin :many
SELECT users.id, users.name
FROM users
WHERE NOT EXISTS (
        SELECT 1
        FROM user_roles
        WHERE user_roles.user_id = users.id
            AND user_roles.role = 'admin'
)
ORDER BY users.name;

-- name: UpsertUserName :exec
INSERT INTO users (id, name)
VALUES ($1, $2)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name;

-- name: InsertPasskey :exec
INSERT INTO passkeys (
        credential_id,
        user_id,
        public_key,
        sign_counter,
        transports,
        label,
        revoked_at
)
VALUES ($1, $2, $3, $4, $5::jsonb, $6, NULL)
ON CONFLICT (credential_id) DO UPDATE
SET user_id = EXCLUDED.user_id,
        public_key = EXCLUDED.public_key,
        sign_counter = EXCLUDED.sign_counter,
        transports = EXCLUDED.transports,
        label = EXCLUDED.label,
        revoked_at = NULL;

-- name: ListPasskeysByUser :many
SELECT id, credential_id, user_id, public_key, sign_counter, transports, label, created_at, revoked_at
FROM passkeys
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetPasskeyByCredentialID :one
SELECT id, credential_id, user_id, public_key, sign_counter, transports, label, created_at, revoked_at
FROM passkeys
WHERE credential_id = $1;

-- name: RevokePasskey :exec
UPDATE passkeys
SET revoked_at = NOW()
WHERE credential_id = $1
    AND revoked_at IS NULL;

-- name: RevokeAllPasskeysForUser :exec
UPDATE passkeys
SET revoked_at = NOW()
WHERE user_id = $1
    AND revoked_at IS NULL;

-- name: UpdatePasskeySignCounter :exec
UPDATE passkeys
SET sign_counter = $2
WHERE credential_id = $1;

-- name: CreateAssignmentLink :one
INSERT INTO passkey_assignment_links (
        token_hash,
        user_id,
        created_by_user_id,
        purpose,
        expires_at
)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, token_hash, user_id, created_by_user_id, purpose, created_at, expires_at, consumed_at, revoked_at;

-- name: GetActiveAssignmentLinkByTokenHash :one
SELECT id, token_hash, user_id, created_by_user_id, purpose, created_at, expires_at, consumed_at, revoked_at
FROM passkey_assignment_links
WHERE token_hash = $1
    AND consumed_at IS NULL
    AND revoked_at IS NULL
    AND (expires_at IS NULL OR expires_at > NOW())
LIMIT 1;

-- name: GetAssignmentLinkByID :one
SELECT id, token_hash, user_id, created_by_user_id, purpose, created_at, expires_at, consumed_at, revoked_at
FROM passkey_assignment_links
WHERE id = $1;

-- name: RevokeAssignmentLink :exec
UPDATE passkey_assignment_links
SET revoked_at = NOW()
WHERE id = $1
    AND consumed_at IS NULL
    AND revoked_at IS NULL;

-- name: GetActiveStandardAssignmentLinkByUserID :one
SELECT id, token_hash, user_id, created_by_user_id, purpose, created_at, expires_at, consumed_at, revoked_at
FROM passkey_assignment_links
WHERE user_id = $1
    AND purpose = 'standard'
    AND consumed_at IS NULL
    AND revoked_at IS NULL
    AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY created_at DESC
LIMIT 1;

-- name: RevokeActiveStandardAssignmentLinksForUser :execrows
UPDATE passkey_assignment_links
SET revoked_at = NOW()
WHERE user_id = $1
    AND purpose = 'standard'
    AND consumed_at IS NULL
    AND revoked_at IS NULL
    AND (expires_at IS NULL OR expires_at > NOW());

-- name: ConsumeAssignmentLink :exec
UPDATE passkey_assignment_links
SET consumed_at = NOW()
WHERE id = $1
    AND consumed_at IS NULL
    AND revoked_at IS NULL;

-- name: GetOrCreateBootstrapLink :one
WITH active_bootstrap AS (
        SELECT id, token_hash, user_id, created_by_user_id, purpose, created_at, expires_at, consumed_at, revoked_at
        FROM passkey_assignment_links
        WHERE purpose = 'bootstrap'
            AND consumed_at IS NULL
            AND revoked_at IS NULL
        ORDER BY created_at DESC
        LIMIT 1
), inserted AS (
        INSERT INTO passkey_assignment_links (token_hash, user_id, created_by_user_id, purpose, expires_at)
        SELECT $1, $2, NULL, 'bootstrap', NULL
        WHERE NOT EXISTS (SELECT 1 FROM active_bootstrap)
        RETURNING id, token_hash, user_id, created_by_user_id, purpose, created_at, expires_at, consumed_at, revoked_at
)
SELECT * FROM active_bootstrap
UNION ALL
SELECT * FROM inserted
LIMIT 1;

-- name: GetActiveBootstrapLink :one
SELECT id, token_hash, user_id, created_by_user_id, purpose, created_at, expires_at, consumed_at, revoked_at
FROM passkey_assignment_links
WHERE purpose = 'bootstrap'
    AND consumed_at IS NULL
    AND revoked_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: RevokeActiveBootstrapLinks :exec
UPDATE passkey_assignment_links
SET revoked_at = NOW()
WHERE purpose = 'bootstrap'
    AND consumed_at IS NULL
    AND revoked_at IS NULL;

-- name: InsertAuthChallenge :exec
INSERT INTO auth_challenges (
        challenge_token,
        challenge_bytes,
        user_id,
        flow_type,
        expires_at,
        used_at
)
VALUES ($1, $2, $3, $4, $5, NULL)
ON CONFLICT (challenge_token) DO UPDATE
SET challenge_bytes = EXCLUDED.challenge_bytes,
        user_id = EXCLUDED.user_id,
        flow_type = EXCLUDED.flow_type,
        expires_at = EXCLUDED.expires_at,
        used_at = NULL;

-- name: GetActiveAuthChallenge :one
SELECT challenge_token, challenge_bytes, user_id, flow_type, expires_at, used_at, created_at
FROM auth_challenges
WHERE challenge_token = $1
    AND used_at IS NULL
    AND expires_at > NOW();

-- name: MarkAuthChallengeUsed :exec
UPDATE auth_challenges
SET used_at = NOW()
WHERE challenge_token = $1
    AND used_at IS NULL;

-- name: DeleteExpiredAuthChallenges :exec
DELETE FROM auth_challenges
WHERE expires_at <= NOW();

-- name: InsertAuditLog :exec
INSERT INTO audit_logs (actor_user_id, action, target_type, target_id, metadata)
VALUES ($1, $2, $3, $4, $5::jsonb);

-- name: ListAuditLogs :many
SELECT id, actor_user_id, action, target_type, target_id, metadata, created_at
FROM audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: DeleteAuditLogsOlderThan :exec
DELETE FROM audit_logs
WHERE created_at < $1;

-- name: GetUsersWithRoles :many
SELECT users.id, users.name,
    ARRAY_REMOVE(ARRAY_AGG(user_roles.role ORDER BY user_roles.role), NULL)::text[] AS roles
FROM users
LEFT JOIN user_roles ON user_roles.user_id = users.id
GROUP BY users.id, users.name
ORDER BY users.name;

-- name: ClearAllAdminRoles :exec
DELETE FROM user_roles
WHERE role = 'admin';

-- name: EnableAdminRemovalOverride :exec
SELECT set_config('app.allow_remove_last_admin', 'on', true);

-- name: InsertUserEmail :exec
INSERT INTO user_emails (user_id, email)
VALUES ($1, $2)
ON CONFLICT (user_id, email) DO NOTHING;

-- name: DeleteUserEmail :exec
DELETE FROM user_emails
WHERE user_id = $1
    AND email = $2;

-- name: ListUserEmails :many
SELECT id, user_id, email, created_at
FROM user_emails
WHERE user_id = $1
ORDER BY created_at ASC;

-- name: GetUserByEmail :one
SELECT users.id, users.name
FROM users
INNER JOIN user_emails ON user_emails.user_id = users.id
WHERE LOWER(user_emails.email) = LOWER($1)
LIMIT 1;

-- name: CreateMagicLink :one
INSERT INTO magic_links (token_hash, user_id, expires_at)
VALUES ($1, $2, $3)
RETURNING id, token_hash, user_id, created_at, expires_at, used_at;

-- name: GetActiveMagicLinkByTokenHash :one
SELECT id, token_hash, user_id, created_at, expires_at, used_at
FROM magic_links
WHERE token_hash = $1
    AND used_at IS NULL
    AND expires_at > NOW();

-- name: ConsumeMagicLink :exec
UPDATE magic_links
SET used_at = NOW()
WHERE id = $1
    AND used_at IS NULL;

-- name: DeleteMagicLinksForUser :exec
DELETE FROM magic_links
WHERE user_id = $1
    AND used_at IS NULL;

-- name: GetLatestMagicLinkForUser :one
SELECT id, token_hash, user_id, created_at, expires_at, used_at
FROM magic_links
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;
