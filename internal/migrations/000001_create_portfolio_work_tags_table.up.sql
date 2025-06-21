CREATE TABLE portfolio_work_tags (
    portfolio_work_id INT NOT NULL REFERENCES portfolio_works(id) ON DELETE CASCADE,
    tag_id INT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (portfolio_work_id, tag_id)
);
