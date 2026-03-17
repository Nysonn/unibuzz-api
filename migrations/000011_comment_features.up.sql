-- Feature: per-video comment section toggle
ALTER TABLE videos ADD COLUMN comments_disabled BOOLEAN NOT NULL DEFAULT FALSE;

-- Feature: user-level comment keyword filters (max 50 per user, enforced at app layer)
CREATE TABLE user_comment_filters (
    id         UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    keyword    TEXT      NOT NULL CHECK (char_length(keyword) BETWEEN 1 AND 100),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, keyword)
);

CREATE INDEX idx_user_comment_filters_user_id ON user_comment_filters(user_id);
