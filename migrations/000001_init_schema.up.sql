-- Enable UUID support
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


-- -------------------------
-- USERS
-- -------------------------
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    full_name TEXT NOT NULL,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,

    password_hash TEXT NOT NULL,

    university_name TEXT,
    course TEXT,
    year_of_study INTEGER,

    profile_photo_url TEXT,

    is_admin BOOLEAN DEFAULT FALSE,
    is_suspended BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);


-- -------------------------
-- SESSIONS
-- -------------------------
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    refresh_token_hash TEXT NOT NULL,

    user_agent TEXT,
    ip_address TEXT,

    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),

    revoked BOOLEAN DEFAULT FALSE
);


-- -------------------------
-- VIDEOS
-- -------------------------
CREATE TABLE videos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    caption TEXT,
    video_url TEXT NOT NULL,
    thumbnail_url TEXT,
    hls_url TEXT,
    dash_url TEXT,

    duration_seconds INTEGER,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);


-- -------------------------
-- HASHTAGS
-- -------------------------
CREATE TABLE hashtags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tag TEXT UNIQUE NOT NULL
);


-- -------------------------
-- VIDEO_HASHTAGS
-- -------------------------
CREATE TABLE video_hashtags (
    video_id UUID REFERENCES videos(id) ON DELETE CASCADE,
    hashtag_id UUID REFERENCES hashtags(id) ON DELETE CASCADE,

    PRIMARY KEY (video_id, hashtag_id)
);


-- -------------------------
-- VOTES
-- -------------------------
CREATE TABLE votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,

    vote_type SMALLINT NOT NULL, -- 1 = upvote, -1 = downvote

    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE (user_id, video_id)
);


-- -------------------------
-- COMMENTS
-- -------------------------
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,

    content TEXT NOT NULL,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);


-- -------------------------
-- REPORTS
-- -------------------------
CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    reporter_id UUID REFERENCES users(id) ON DELETE SET NULL,
    video_id UUID REFERENCES videos(id) ON DELETE CASCADE,

    reason TEXT,
    description TEXT,

    created_at TIMESTAMP DEFAULT NOW(),
    resolved BOOLEAN DEFAULT FALSE
);


-- -------------------------
-- ADMIN ACTIONS
-- -------------------------
CREATE TABLE admin_actions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    admin_id UUID REFERENCES users(id),
    target_user_id UUID REFERENCES users(id),

    action_type TEXT,
    notes TEXT,

    created_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE users ADD COLUMN suspended BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN banned BOOLEAN DEFAULT FALSE;