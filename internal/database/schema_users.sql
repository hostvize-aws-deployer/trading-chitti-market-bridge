-- Multi-User Authentication and Session Management Schema
-- Part of Phase 2: Multi-Account Integration

-- Users table
CREATE TABLE IF NOT EXISTS auth.users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_users_email ON auth.users(email);
CREATE INDEX idx_users_active ON auth.users(is_active);

-- Sessions table for JWT token management
CREATE TABLE IF NOT EXISTS auth.sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    refresh_token_hash TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address TEXT,
    user_agent TEXT,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_sessions_user ON auth.sessions(user_id);
CREATE INDEX idx_sessions_token ON auth.sessions(token_hash);
CREATE INDEX idx_sessions_expires ON auth.sessions(expires_at);

-- Update brokers.config to support multi-user accounts
ALTER TABLE brokers.config
    ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES auth.users(user_id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS account_name TEXT,
    ADD COLUMN IF NOT EXISTS is_default BOOLEAN DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_broker_config_user ON brokers.config(user_id);

-- Ensure each user can only have one default broker config per broker_name
CREATE UNIQUE INDEX IF NOT EXISTS idx_broker_config_user_default
    ON brokers.config(user_id, broker_name)
    WHERE is_default = TRUE;

-- User API keys for external integrations (optional)
CREATE TABLE IF NOT EXISTS auth.api_keys (
    api_key_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    key_hash TEXT NOT NULL UNIQUE,
    key_name TEXT NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{"read"}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX idx_api_keys_user ON auth.api_keys(user_id);
CREATE INDEX idx_api_keys_hash ON auth.api_keys(key_hash);

-- Audit log for security tracking
CREATE TABLE IF NOT EXISTS auth.audit_log (
    log_id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES auth.users(user_id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    ip_address TEXT,
    user_agent TEXT,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_user ON auth.audit_log(user_id, created_at DESC);
CREATE INDEX idx_audit_log_action ON auth.audit_log(action, created_at DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to auto-update updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE auth.users IS 'User accounts for multi-tenant Market Bridge';
COMMENT ON TABLE auth.sessions IS 'Active JWT sessions with refresh tokens';
COMMENT ON TABLE auth.api_keys IS 'API keys for programmatic access';
COMMENT ON TABLE auth.audit_log IS 'Security audit trail';
