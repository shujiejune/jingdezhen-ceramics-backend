CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL, -- User who performed the action
    notification_type VARCHAR(50) NOT NULL, -- 'like_post', 'comment_post', 'kudo_artwork', 'artwork_chosen_by_editor', 'saved_post_update', 'direct_message'
    entity_type VARCHAR(50), -- 'forum_post', 'forum_comment', 'portfolio_work', 'user_note'
    entity_id BIGINT,
    message TEXT NOT NULL, -- Optional custom message
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Create indexes for faster lookups
CREATE INDEX idx_notifications_recipient_user_id ON notifications(recipient_user_id);
