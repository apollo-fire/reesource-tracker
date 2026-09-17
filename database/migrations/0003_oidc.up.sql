ALTER TABLE users ADD COLUMN oidc_sub TEXT;
-- Partial index allows multiple NULL oidc_sub values (pre-OIDC users).
CREATE UNIQUE INDEX users_oidc_sub_idx ON users(oidc_sub) WHERE oidc_sub IS NOT NULL;

CREATE TABLE sessions (
    id                      TEXT PRIMARY KEY,
    user_id                 BYTEA NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    roles                   TEXT[] NOT NULL DEFAULT '{}',
    oidc_sid                TEXT,
    refresh_token           TEXT,
    access_token_expires_at TIMESTAMPTZ NOT NULL,
    expires_at              TIMESTAMPTZ NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX sessions_user_id_idx  ON sessions(user_id);
CREATE INDEX sessions_oidc_sid_idx ON sessions(oidc_sid) WHERE oidc_sid IS NOT NULL;
