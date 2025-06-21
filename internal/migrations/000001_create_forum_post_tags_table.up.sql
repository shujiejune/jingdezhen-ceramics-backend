CREATE TABLE forum_post_tags (
    post_id INT NOT NULL REFERENCES forum_posts(id) ON DELETE CASCADE,
    tag_id INT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);
