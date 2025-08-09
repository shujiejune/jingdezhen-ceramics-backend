CREATE TABLE portfolio_work_upvotes (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    portfolio_work_id BIGINT NOT NULL REFERENCES portfolio_works(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- A user can only upvote a specific work once
    PRIMARY KEY (user_id, portfolio_work_id)
);
-- Create indexes for faster lookups on foreign keys
CREATE INDEX idx_portfolio_work_upvotes_work_id ON portfolio_work_upvotes(portfolio_work_id);
