INSERT INTO chain_static (chain_id, name, symbol, network_type, is_evm, chain_details)
VALUES 
    (0002, 'Ethereum', 'ETH', 'mainnet', true, '{"description": "Ethereum mainnet", "explorer_url": "https://etherscan.io"}');

INSERT INTO rpc_endpoints (chain_id, endpoint_url, provider, is_active, priority, geozone)
VALUES 
    ((SELECT id FROM chain_static WHERE chain_id = 0002), 'https://eth-mainnet.g.alchemy.com/v2/IpUziTXbC3yeVTYO6I71KRGtcS9QGUuv', 'Alchemy', true, 10, 'IND'),