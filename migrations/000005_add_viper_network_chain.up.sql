-- Add Viper Network as a supported chain
INSERT INTO chain_static (chain_id, name, symbol, network_type, is_evm, chain_details)
VALUES 
    (0001, 'Viper Network', 'VPR', 'testnet', false, '{"description": "Viper Network decentralized RPC chain", "explorer_url": ""}');

-- Add Viper Network RPC endpoints
INSERT INTO rpc_endpoints (chain_id, endpoint_url, provider, is_active, priority, geozone)
VALUES 
    ((SELECT id FROM chain_static WHERE chain_id = 0001), 'http://localhost:8082', 'local-node', true, 10, 'IND'),
    ((SELECT id FROM chain_static WHERE chain_id = 0001), 'http://localhost:26657', 'tendermint-rpc', true, 5, 'IND');
