CREATE TABLE users (
    -- Fields for registeration
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nickname VARCHAR(100),
    email VARCHAR(255) UNIQUE, -- Can be null if using other auth methods primarily
    password_hash VARCHAR(255), -- If handling email/password directly

    -- Fields for email activation and password reset
    is_active BOOLEAN NOT NULL DEFAULT FALSE, -- If register by email and password
    activation_token TEXT,  -- Sent by email
    activation_token_expires_at TIMESTAMPTZ,
    password_reset_token TEXT,
    password_reset_expires_at TIMESTAMPTZ,

    -- Fields for OAuth and profile
    avatar_url TEXT,
    auth_provider VARCHAR(50) NOT NULL DEFAULT 'email', -- e.g., 'email', 'google', 'wechat'
    auth_provider_id TEXT, -- User ID from the OAuth provider
    role VARCHAR(20) NOT NULL DEFAULT 'normal_user' CHECK (role IN ('guest', 'normal_user', 'admin')),
    profile_data JSONB, -- For heatmap, badges, other contacts

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX ON users(activation_token) WHERE activation_token IS NOT NULL;
CREATE UNIQUE INDEX ON users(password_reset_token) WHERE password_reset_token IS NOT NULL;
CREATE UNIQUE INDEX ON users (auth_provider, auth_provider_id) WHERE auth_provider_id IS NOT NULL;
