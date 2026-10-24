-- Mirrors versioned migration 000134_shared_mcp_login.

CREATE TABLE IF NOT EXISTS shared_mcp_configs (
    tenant_id INTEGER PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 0,
    tools TEXT NOT NULL DEFAULT '[]',
    knowledge_base_ids TEXT NOT NULL DEFAULT '[]',
    default_agent_id VARCHAR(36) NOT NULL DEFAULT '',
    rate_limit_per_minute INTEGER NOT NULL DEFAULT 60,
    redirect_uri_allowlist TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS shared_mcp_clients (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL DEFAULT '',
    redirect_uris TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS shared_mcp_auth_codes (
    code_hash VARCHAR(64) PRIMARY KEY,
    client_id VARCHAR(36) NOT NULL,
    redirect_uri TEXT NOT NULL DEFAULT '',
    code_challenge VARCHAR(128) NOT NULL DEFAULT '',
    user_id VARCHAR(36) NOT NULL,
    tenant_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shared_mcp_auth_codes_client ON shared_mcp_auth_codes (client_id);
CREATE INDEX IF NOT EXISTS idx_shared_mcp_auth_codes_user ON shared_mcp_auth_codes (user_id);
CREATE INDEX IF NOT EXISTS idx_shared_mcp_auth_codes_tenant ON shared_mcp_auth_codes (tenant_id);

CREATE TABLE IF NOT EXISTS shared_mcp_grants (
    id VARCHAR(36) PRIMARY KEY,
    client_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    tenant_id INTEGER NOT NULL,
    refresh_token_hash VARCHAR(64) NOT NULL DEFAULT '',
    refresh_expires_at DATETIME NOT NULL,
    absolute_expires_at DATETIME NOT NULL,
    revoked_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shared_mcp_grants_client ON shared_mcp_grants (client_id);
CREATE INDEX IF NOT EXISTS idx_shared_mcp_grants_user ON shared_mcp_grants (user_id);
CREATE INDEX IF NOT EXISTS idx_shared_mcp_grants_tenant ON shared_mcp_grants (tenant_id);
CREATE INDEX IF NOT EXISTS idx_shared_mcp_grants_refresh_hash ON shared_mcp_grants (refresh_token_hash);
