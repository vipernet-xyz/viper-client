DELETE FROM rpc_endpoints WHERE chain_id = (SELECT id FROM chain_static WHERE chain_id = 0002);

DELETE FROM chain_static WHERE chain_id = 0002;