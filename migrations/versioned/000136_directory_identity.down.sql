DROP TABLE IF EXISTS directory_sync_runs;
DROP TABLE IF EXISTS user_email_aliases;
DROP TABLE IF EXISTS user_directory_identities;
ALTER TABLE users DROP COLUMN IF EXISTS password_login_enabled;
