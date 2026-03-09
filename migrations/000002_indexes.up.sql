CREATE INDEX idx_videos_created_at ON videos(created_at DESC);

CREATE INDEX idx_comments_video_id ON comments(video_id);

CREATE INDEX idx_votes_video_id ON votes(video_id);

CREATE INDEX idx_reports_video_id ON reports(video_id);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);

CREATE INDEX idx_users_email ON users(email);