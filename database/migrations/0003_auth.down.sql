DROP TRIGGER IF EXISTS trg_prevent_last_admin_removal ON user_roles;
DROP FUNCTION IF EXISTS prevent_last_admin_removal;

DROP INDEX IF EXISTS idx_assignment_links_active_token;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_assignment_links_expires_at;
DROP INDEX IF EXISTS idx_assignment_links_user_id;
DROP INDEX IF EXISTS idx_passkeys_user_id;

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS auth_challenges;
DROP TABLE IF EXISTS passkey_assignment_links;
DROP TABLE IF EXISTS passkeys;
DROP TABLE IF EXISTS user_roles;
