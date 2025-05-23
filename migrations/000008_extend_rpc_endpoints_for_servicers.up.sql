-- Migration 000008: Extend rpc_endpoints for servicer monitoring
ALTER TABLE rpc_endpoints
ADD COLUMN IF NOT EXISTS public_key VARCHAR(255),
ADD COLUMN IF NOT EXISTS response_time_ms INTEGER,
ADD COLUMN IF NOT EXISTS last_ping_timestamp TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS servicer_type VARCHAR(20) DEFAULT 'static' CHECK (servicer_type IN ('static', 'discovered'));

-- Add indexes for servicer-specific queries
CREATE INDEX IF NOT EXISTS idx_rpc_endpoints_public_key ON rpc_endpoints(public_key);
CREATE INDEX IF NOT EXISTS idx_rpc_endpoints_servicer_type ON rpc_endpoints(servicer_type);
CREATE INDEX IF NOT EXISTS idx_rpc_endpoints_response_time ON rpc_endpoints(response_time_ms);
CREATE INDEX IF NOT EXISTS idx_rpc_endpoints_last_ping ON rpc_endpoints(last_ping_timestamp);

-- Add unique constraint for discovered servicers
-- Note: PostgreSQL syntax for partial unique constraint.
-- This constraint ensures that for 'discovered' servicers, the public_key is unique.
-- For 'static' servicers, public_key can be NULL or non-unique if desired.
ALTER TABLE rpc_endpoints
ADD CONSTRAINT unique_discovered_servicer_public_key
UNIQUE (public_key, servicer_type) WHERE (servicer_type = 'discovered');
