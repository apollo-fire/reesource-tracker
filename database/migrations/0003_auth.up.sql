CREATE TABLE IF NOT EXISTS user_roles (
    user_id BYTEA NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'maintainer', 'user')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role)
);

CREATE TABLE IF NOT EXISTS passkeys (
    id BIGSERIAL PRIMARY KEY,
    credential_id BYTEA NOT NULL UNIQUE,
    user_id BYTEA NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key BYTEA NOT NULL,
    sign_counter BIGINT NOT NULL DEFAULT 0,
    transports JSONB NOT NULL DEFAULT '[]'::jsonb,
    label TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS passkey_assignment_links (
    id BIGSERIAL PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    user_id BYTEA NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_by_user_id BYTEA REFERENCES users(id) ON DELETE SET NULL,
    purpose TEXT NOT NULL CHECK (purpose IN ('bootstrap', 'standard')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    consumed_at TIMESTAMP,
    revoked_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS auth_challenges (
    challenge_token TEXT PRIMARY KEY,
    challenge_bytes BYTEA NOT NULL,
    user_id BYTEA REFERENCES users(id) ON DELETE CASCADE,
    flow_type TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    actor_user_id BYTEA REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_passkeys_user_id ON passkeys(user_id);
CREATE INDEX IF NOT EXISTS idx_assignment_links_user_id ON passkey_assignment_links(user_id);
CREATE INDEX IF NOT EXISTS idx_assignment_links_expires_at ON passkey_assignment_links(expires_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_assignment_links_active_token ON passkey_assignment_links(token_hash)
WHERE consumed_at IS NULL AND revoked_at IS NULL;

-- Ensure we cannot remove the last admin role.
CREATE OR REPLACE FUNCTION prevent_last_admin_removal()
RETURNS TRIGGER AS $$
DECLARE
    admin_count BIGINT;
    allow_override TEXT;
BEGIN
    IF OLD.role <> 'admin' THEN
        RETURN OLD;
    END IF;

    allow_override := current_setting('app.allow_remove_last_admin', true);
    IF allow_override = 'on' THEN
        RETURN OLD;
    END IF;

    SELECT COUNT(*) INTO admin_count FROM user_roles WHERE role = 'admin';
    IF admin_count <= 1 THEN
        RAISE EXCEPTION 'cannot remove last admin';
    END IF;

    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_last_admin_removal ON user_roles;
CREATE TRIGGER trg_prevent_last_admin_removal
BEFORE DELETE ON user_roles
FOR EACH ROW
EXECUTE FUNCTION prevent_last_admin_removal();
