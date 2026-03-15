-- Indexes to support hashtag search and lookup performance
CREATE INDEX IF NOT EXISTS idx_hashtags_tag ON hashtags(tag);
CREATE INDEX IF NOT EXISTS idx_video_hashtags_hashtag_id ON video_hashtags(hashtag_id);
CREATE INDEX IF NOT EXISTS idx_video_hashtags_video_id ON video_hashtags(video_id);

-- Support structured report reasons going forward.
-- The existing `reason` column stores the enum value (e.g. 'harassment').
-- The existing `description` column stores the optional custom message (only used when reason = 'other').
-- No structural changes needed; validation is enforced at the application layer.
