-- Migration 000008: Revert extension of rpc_endpoints for servicer monitoring
DROP INDEX IF EXISTS idx_unique_discovered_servicer_public_key;

DROP INDEX IF EXISTS idx_rpc_endpoints_public_key;
DROP INDEX IF EXISTS idx_rpc_endpoints_servicer_type;
DROP INDEX IF EXISTS idx_rpc_endpoints_response_time;
DROP INDEX IF EXISTS idx_rpc_endpoints_last_ping;

ALTER TABLE rpc_endpoints
DROP COLUMN IF EXISTS servicer_type,
DROP COLUMN IF EXISTS last_ping_timestamp,
DROP COLUMN IF EXISTS response_time_ms,
DROP COLUMN IF EXISTS public_key;

DELETE FROM rpc_endpoints WHERE chain_id = (SELECT id FROM chain_static WHERE chain_id = 0002);

DELETE FROM chain_static WHERE chain_id = 0002;

-- No explicit down migration for logs table, assuming it can be dropped if needed.
-- Or, if it should persist, this down migration does nothing for it.

-- Remove Viper Network RPC endpoints
DELETE FROM rpc_endpoints WHERE chain_id = (SELECT id FROM chain_static WHERE chain_id = 0001);

-- Remove Viper Network chain
DELETE FROM chain_static WHERE chain_id = 0001;

DROP TABLE IF EXISTS rpc_endpoints;
DROP TABLE IF EXISTS apps;
DROP TABLE IF EXISTS chain_static;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS logs; -- Added drop for logs table as per create_stats_table up migration 