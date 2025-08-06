CREATE TABLE portfolio_works (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- Student creator
    title VARCHAR(255) NOT NULL,
    description TEXT, -- Creator's introduction
    is_editors_choice BOOLEAN DEFAULT FALSE, -- Set by admin/teacher
    upvotes_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
