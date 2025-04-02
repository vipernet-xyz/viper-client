CREATE TABLE IF NOT EXISTS apps (
  id SERIAL PRIMARY KEY,
  app_identifier VARCHAR(64) NOT NULL UNIQUE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  allowed_origins TEXT[],
  allowed_chains INTEGER[],
  api_key_hash VARCHAR(255) NOT NULL,
  rate_limit INTEGER DEFAULT 10000,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
); 