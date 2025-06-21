CREATE TABLE portfolio_work_kudos (
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    portfolio_work_id INT NOT NULL REFERENCES portfolio_works(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, portfolio_work_id)
);
