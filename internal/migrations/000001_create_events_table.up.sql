-- festivals, fairs, museums
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'Festival', 'Fair', 'Museum', 'Exhibition'
    brief_introduction TEXT,
    photograph_url TEXT,
    article_slug VARCHAR(255) UNIQUE NOT NULL, -- For the internal link
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
