CREATE TABLE user_subscriptions (
    subscriber_user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- The one who subscribes
    target_user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- The one being subscribed to
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (subscriber_user_id, target_user_id)
);
