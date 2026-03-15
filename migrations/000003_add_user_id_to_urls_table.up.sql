ALTER TABLE urls ADD COLUMN IF NOT EXISTS user_id TEXT;
CREATE INDEX IF NOT EXISTS idx_urls_user_id ON urls(user_id);