CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  provider_user_id VARCHAR(255) NOT NULL UNIQUE,
  email VARCHAR(255) UNIQUE,
  name VARCHAR(255),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
); 

CREATE TABLE IF NOT EXISTS chain_static (
  id SERIAL PRIMARY KEY,
  chain_id VARCHAR(50) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  symbol VARCHAR(50) NOT NULL,
  network_type VARCHAR(20) NOT NULL, -- mainnet, testnet
  chain_details JSONB,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
); 

CREATE TABLE IF NOT EXISTS apps (
  id SERIAL PRIMARY KEY,
  api_key VARCHAR(64) NOT NULL UNIQUE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  allowed_origins TEXT[],
  allowed_chains TEXT[],
  rate_limit INTEGER DEFAULT 10000,
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
); 

CREATE TABLE IF NOT EXISTS rpc_endpoints (
  id SERIAL PRIMARY KEY,
  chain_id INTEGER NOT NULL REFERENCES chain_static(chain_id) ON DELETE CASCADE,
  geozone VARCHAR(100),
  endpoint_url TEXT NOT NULL,
  is_active BOOLEAN DEFAULT TRUE,
  health_status VARCHAR(50),  -- online, offline
  response_time_ms INTEGER,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
); 

CREATE TABLE IF NOT EXISTS logs (
    id SERIAL PRIMARY KEY,
    endpoint VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) NOT NULL,
    chain_id INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add Viper Network as a supported chain
INSERT INTO chain_static (chain_id, name, symbol, network_type, is_evm, chain_details)
VALUES 
    (0001, 'Viper Network', 'VPR', 'testnet', false, '{"description": "Viper Network decentralized RPC chain", "explorer_url": ""}');

-- Add Viper Network RPC endpoints
INSERT INTO rpc_endpoints (chain_id, endpoint_url, provider, is_active, priority, geozone)
VALUES 
    ((SELECT id FROM chain_static WHERE chain_id = 0001), 'http://localhost:8082', 'local-node', true, 10, 'IND'),
    ((SELECT id FROM chain_static WHERE chain_id = 0001), 'http://localhost:26657', 'tendermint-rpc', true, 5, 'IND');



INSERT INTO chain_static (chain_id, name, symbol, network_type, is_evm, chain_details)
VALUES 
    (0002, 'Ethereum', 'ETH', 'mainnet', true, '{"description": "Ethereum mainnet", "explorer_url": "https://etherscan.io"}');

INSERT INTO rpc_endpoints (chain_id, endpoint_url, provider, is_active, priority, geozone)
VALUES 
    ((SELECT id FROM chain_static WHERE chain_id = 0002), 'https://eth-mainnet.g.alchemy.com/v2/IpUziTXbC3yeVTYO6I71KRGtcS9QGUuv', 'Alchemy', true, 10, 'IND');

