CREATE TABLE endpoint_performance_metrics (
    id BIGSERIAL PRIMARY KEY,
    rpc_endpoint_id INTEGER NOT NULL REFERENCES rpc_endpoints(id) ON DELETE CASCADE,
    ping_timestamp TIMESTAMPTZ NOT NULL,
    response_time_ms INTEGER NOT NULL,
    http_status_code INTEGER,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_endpoint_performance_metrics_rpc_endpoint_id ON endpoint_performance_metrics(rpc_endpoint_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_performance_metrics_ping_timestamp ON endpoint_performance_metrics(ping_timestamp);
