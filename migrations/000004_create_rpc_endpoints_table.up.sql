CREATE TABLE IF NOT EXISTS rpc_endpoints (
  id SERIAL PRIMARY KEY,
  chain_id INTEGER NOT NULL REFERENCES chain_static(chain_id) ON DELETE CASCADE,
  geozone VARCHAR(100),
  endpoint_url TEXT NOT NULL,
  provider VARCHAR(100),
  is_active BOOLEAN DEFAULT TRUE,
  priority INTEGER DEFAULT 1,
  health_check_timestamp TIMESTAMP WITH TIME ZONE,
  health_status VARCHAR(50),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
); 