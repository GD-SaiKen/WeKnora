-- Migration: 000135_directory_identity
DO $$ BEGIN RAISE NOTICE '[Migration 000135] Directory identity and sync runs'; END $$;

ALTER TABLE users ADD COLUMN IF NOT EXISTS password_login_enabled BOOLEAN NOT NULL DEFAULT true;

CREATE TABLE IF NOT EXISTS user_directory_identities (
    user_id VARCHAR(36) PRIMARY KEY,
    ai_hub_user_id VARCHAR(128) NOT NULL UNIQUE,
    subject VARCHAR(255) NOT NULL UNIQUE,
    directory_email VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    organization_name VARCHAR(255) NOT NULL DEFAULT '',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_email_aliases (
    email VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_user_email_aliases_user_id ON user_email_aliases(user_id);

CREATE TABLE IF NOT EXISTS directory_sync_runs (
    id VARCHAR(36) PRIMARY KEY,
    status VARCHAR(16) NOT NULL,
    full_sync BOOLEAN NOT NULL DEFAULT false,
    cursor TEXT NOT NULL DEFAULT '',
    synchronized_users INTEGER NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    failures JSONB NOT NULL DEFAULT '[]',
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP WITH TIME ZONE
);
