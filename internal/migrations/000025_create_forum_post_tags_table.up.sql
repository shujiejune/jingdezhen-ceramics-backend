CREATE TABLE forum_post_tags (
    post_id BIGINT NOT NULL REFERENCES forum_posts(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);
