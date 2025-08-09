CREATE TABLE assignments (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    attachment_urls JSONB, -- Use JSONB to store an array of URL strings
    apply_deadline BOOLEAN NOT NULL DEFAULT FALSE,
    deadline TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
