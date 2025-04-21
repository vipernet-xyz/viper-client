-- Remove Viper Network RPC endpoints
DELETE FROM rpc_endpoints WHERE chain_id = (SELECT id FROM chain_static WHERE chain_id = 0001);

-- Remove Viper Network chain
DELETE FROM chain_static WHERE chain_id = 0001;
