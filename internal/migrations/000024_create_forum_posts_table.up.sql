CREATE TABLE forum_posts (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id BIGINT REFERENCES forum_categories(id) ON DELETE SET NULL,
    category_name VARCHAR(50) REFERENCES forum_categories(name) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL, -- Markdown
    search_vector tsvector,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE, -- By admin
    view_count INT NOT NULL DEFAULT 0,
    like_count INT NOT NULL DEFAULT 0,
    comment_count INT NOT NULL DEFAULT 0,
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- For sorting by latest activity
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create a GIN index on search_vector for very fast searching
CREATE INDEX forum_posts_search_idx ON forum_posts USING GIN(search_vector);

-- Create a function that will be used to update the search_vector column
CREATE OR REPLACE FUNCTION update_forum_posts_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    -- Combine the title and content, giving more weight to the title (A) than the content (B)
    -- 'simple' is a good configuration for multi-language or technical content.
    NEW.search_vector :=
        setweight(to_tsvector('simple', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(NEW.content, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create a trigger that automatically calls the function whenever a post is created or updated
CREATE TRIGGER tsvectorupdate
BEFORE INSERT OR UPDATE ON forum_posts
FOR EACH ROW EXECUTE FUNCTION update_forum_posts_search_vector();

-- To populate the search_vector for existing posts, run this once after creating the trigger:
-- UPDATE forum_posts SET search_vector = to_tsvector('simple', COALESCE(title, '')) || to_tsvector('simple', COALESCE(content, ''));
