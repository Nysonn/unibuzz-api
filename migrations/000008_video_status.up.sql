-- Make video_url nullable so we can insert a pending row before processing completes.
ALTER TABLE videos ALTER COLUMN video_url DROP NOT NULL;

-- Add a status column to track processing state.
ALTER TABLE videos ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'pending';
