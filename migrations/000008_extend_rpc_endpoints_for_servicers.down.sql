-- Migration 000008: Revert extension of rpc_endpoints for servicer monitoring
ALTER TABLE rpc_endpoints
DROP CONSTRAINT IF EXISTS unique_discovered_servicer_public_key;

DROP INDEX IF EXISTS idx_rpc_endpoints_public_key;
DROP INDEX IF EXISTS idx_rpc_endpoints_servicer_type;
DROP INDEX IF EXISTS idx_rpc_endpoints_response_time;
DROP INDEX IF EXISTS idx_rpc_endpoints_last_ping;

ALTER TABLE rpc_endpoints
DROP COLUMN IF EXISTS servicer_type,
DROP COLUMN IF EXISTS last_ping_timestamp,
DROP COLUMN IF EXISTS response_time_ms,
DROP COLUMN IF EXISTS public_key;
