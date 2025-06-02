CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(), -- For external reference, if Supabase doesn't provide one you like
    nickname VARCHAR(100),
    email VARCHAR(255) UNIQUE, -- Can be null if using other auth methods primarily
    password_hash VARCHAR(255), -- If handling email/password directly, less needed with Supabase for primary auth
    avatar_url TEXT,
    auth_provider VARCHAR(50) NOT NULL DEFAULT 'email', -- e.g., 'email', 'google', 'wechat'
    auth_provider_id TEXT, -- User ID from the OAuth provider
    role VARCHAR(20) NOT NULL DEFAULT 'normal_user' CHECK (role IN ('guest', 'normal_user', 'admin')),
    profile_data JSONB, -- For heatmap, badges, other contacts
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX ON users (auth_provider, auth_provider_id) WHERE auth_provider_id IS NOT NULL;
