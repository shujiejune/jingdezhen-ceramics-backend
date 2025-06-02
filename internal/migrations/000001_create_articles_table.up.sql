-- for events
CREATE TABLE articles ( -- Detailed content for activities, potentially other long-form content
    id SERIAL PRIMARY KEY,
    slug VARCHAR(255) UNIQUE NOT NULL, -- Matches activity_slug or can be independent
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL, -- Markdown or HTML
    author_id INT REFERENCES users(id) ON DELETE SET NULL, -- If written by a platform user/admin
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
