# Servicer Performance Monitoring System - Implementation Plan

## Overview
Implement a cron job system that generates sessions, extracts servicers, monitors their performance through ping testing, and maintains performance metrics for optimal servicer selection using the existing `rpc_endpoints` infrastructure.

## Current Database Analysis

### Existing Tables:
1. **users** - User management (id, provider_user_id, email, name, created_at, updated_at)
2. **chain_static** - Blockchain definitions (id, chain_id, name, symbol, network_type, is_evm, chain_details, created_at, updated_at)
3. **apps** - Application management (id, api_key, user_id, name, description, allowed_origins, allowed_chains, rate_limit, created_at, updated_at)
4. **rpc_endpoints** - RPC endpoint management (id, chain_id, geozone, endpoint_url, provider, is_active, priority, health_check_timestamp, health_status, created_at, updated_at)
5. **logs** - Request logging (id, endpoint, api_key, chain_id, created_at, updated_at)

### Existing Chain Data:
- **Chain ID 0001**: Viper Network (VPR, testnet)
- **Chain ID 0002**: Ethereum (ETH, mainnet)

### Key Insight:
**Each RPC endpoint in `rpc_endpoints` represents a servicer**, so we can extend this table to store servicer-specific data and performance metrics instead of creating separate tables.

## Technical Architecture

### Components
1. **Database Schema Extensions** - Extend existing `rpc_endpoints` table with servicer-specific fields
2. **Cron Job Service** - Scheduled performance monitoring
3. **Performance Measurement** - HTTP ping and response time calculation stored in `health_status`
4. **Data Analysis** - Average response time calculation and ranking from existing health data
5. **Configuration Management** - Configurable parameters

---

## Epic: Servicer Performance Monitoring System

### Story 1: RPC Endpoints Table Extension for Servicer Monitoring
**As a** system administrator  
**I want** to extend the existing `rpc_endpoints` table to store servicer-specific data and performance metrics  
**So that** I can track servicer performance without creating redundant tables

#### Subtasks:
- [ ] Add `public_key` field to `rpc_endpoints` table for servicer identification
- [ ] Add `response_time_ms` field to store latest response time
- [ ] Add `last_ping_timestamp` field to track when servicer was last checked
- [ ] Extend `health_status` field to store JSON performance metrics (average response time, error rates)
- [ ] Add `servicer_type` field to distinguish between static RPC endpoints and discovered servicers
- [ ] Create migration 000008_extend_rpc_endpoints_for_servicers.up.sql
- [ ] Add appropriate indexes for servicer-specific queries

#### Acceptance Criteria:
- Extended table supports both existing RPC endpoints and new servicer entries
- Performance metrics stored efficiently in existing infrastructure
- Migration preserves all existing RPC endpoint data
- New fields have appropriate constraints and defaults

---

### Story 2: Servicer Discovery and Registration Service
**As a** monitoring system  
**I want** to generate sessions and register discovered servicers as RPC endpoints  
**So that** I can monitor all available servicers using existing infrastructure

#### Subtasks:
- [ ] Create ServicerDiscovery service that uses existing relay.Client from examples/simple_relay
- [ ] Implement session generation with configurable blockchain/geozone combinations
- [ ] Create servicer registration logic that adds discovered servicers to `rpc_endpoints` table
- [ ] Implement servicer deduplication based on `public_key` uniqueness in `rpc_endpoints`
- [ ] Add servicer validation using existing crypto functions from relay.go
- [ ] Create unit tests that work with existing relay client infrastructure

#### Acceptance Criteria:
- Generates sessions for existing blockchain IDs (0001, 0002) from chain_static table
- Discovered servicers stored as entries in `rpc_endpoints` with `servicer_type = 'discovered'`
- Handles network errors gracefully without breaking the monitoring cycle
- Integrates seamlessly with existing RPC endpoint management

---

### Story 3: Performance Measurement Engine for RPC Endpoints
**As a** monitoring system  
**I want** to measure performance of all servicers stored as RPC endpoints  
**So that** I can update their health status with real-time performance data

#### Subtasks:
- [ ] Create PerformanceMeasurer that queries `rpc_endpoints` for servicers to monitor
- [ ] Implement HTTP health check (GET request to endpoint_url + "/health" or direct ping)
- [ ] Add response time measurement with millisecond precision using time.Now()
- [ ] Update `response_time_ms`, `last_ping_timestamp`, and `health_check_timestamp` fields
- [ ] Store comprehensive performance metrics in `health_status` as JSON
- [ ] Create unit and integration tests with mock HTTP servers

#### Acceptance Criteria:
- Measures HTTP response time with high precision
- Updates existing RPC endpoint health fields with performance data
- Stores detailed metrics in `health_status` JSON for trend analysis
- Supports concurrent measurement for improved performance

---

### Story 4: Cron Job Orchestrator for RPC Endpoint Monitoring
**As a** system administrator  
**I want** a scheduled job that discovers servicers and monitors all RPC endpoints  
**So that** servicer performance data is collected automatically on a regular schedule

#### Subtasks:
- [ ] Create CronJobOrchestrator using existing DB connection patterns from internal/db/db.go
- [ ] Implement complete workflow: discover servicers → register in rpc_endpoints → measure performance
- [ ] Add job execution metrics and comprehensive logging using existing logging patterns
- [ ] Implement graceful shutdown and context cancellation for clean stops
- [ ] Add database connection management using existing pool patterns
- [ ] Create integration tests that validate the complete end-to-end workflow

#### Acceptance Criteria:
- Runs on configurable cron schedule using environment variables
- Executes complete workflow without manual intervention
- Updates both existing RPC endpoints and newly discovered servicers
- Uses existing database connection and error handling patterns

---

### Story 5: Performance Analytics and Ranking System
**As a** user  
**I want** to query the best performing servicers from the `rpc_endpoints` table based on performance data  
**So that** I can make informed decisions about servicer selection and routing

#### Subtasks:
- [ ] Create PerformanceAnalyzer that reads performance data from `rpc_endpoints.health_status`
- [ ] Implement servicer ranking by average response time from `rpc_endpoints.response_time_ms`
- [ ] Add filtering capabilities by `chain_id`, `geozone`, and `servicer_type`
- [ ] Create REST API endpoints following existing API patterns from internal/api/
- [ ] Add performance trend analysis using historical data in `health_status` JSON
- [ ] Create comprehensive unit tests for analytics calculations

#### Acceptance Criteria:
- Calculates accurate performance metrics from existing RPC endpoint data
- Provides ranked servicer lists with performance scores
- Filters results by blockchain using existing chain_static data
- Returns well-formatted JSON responses using existing API patterns

---

### Story 6: Configuration Management and Environment Integration
**As a** system administrator  
**I want** configurable parameters that integrate with existing configuration patterns  
**So that** I can tune the monitoring system for different environments

#### Subtasks:
- [ ] Extend existing Config struct in internal/utils/config.go with monitoring parameters
- [ ] Add environment variable support for cron schedule, session count, and timeout values
- [ ] Implement configuration validation with sensible defaults for all parameters
- [ ] Create configuration documentation with examples for all environments
- [ ] Add runtime configuration validation that prevents invalid blockchain_id references
- [ ] Create tests for configuration parsing using existing test patterns

#### Acceptance Criteria:
- All monitoring parameters configurable via environment variables
- Configuration validates against existing chain_static data
- Provides clear error messages for invalid configurations
- Documentation includes examples for development, staging, and production

---

### Story 7: Monitoring and Observability Integration
**As a** system administrator  
**I want** comprehensive monitoring that integrates with existing logging infrastructure  
**So that** I can track system health and troubleshoot issues effectively

#### Subtasks:
- [ ] Add structured logging throughout the application using existing logging patterns
- [ ] Implement metrics collection for job duration, success rates, and error counts
- [ ] Create health check endpoints that integrate with existing API structure
- [ ] Add alerting integration points for critical monitoring failures
- [ ] Implement database metrics collection (connection pool usage, query performance)
- [ ] Create monitoring dashboard recommendations and alerting thresholds

#### Acceptance Criteria:
- Structured logs integrate with existing application logging
- Metrics available for external monitoring systems (Prometheus, etc.)
- Health checks provide actionable system status information
- Critical errors trigger appropriate alerts and notifications

---

## Updated Technical Specifications

### Extended RPC Endpoints Table (Single Migration)

```sql
-- Migration 000008: Extend rpc_endpoints for servicer monitoring
ALTER TABLE rpc_endpoints 
ADD COLUMN IF NOT EXISTS public_key VARCHAR(255),
ADD COLUMN IF NOT EXISTS response_time_ms INTEGER,
ADD COLUMN IF NOT EXISTS last_ping_timestamp TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS servicer_type VARCHAR(20) DEFAULT 'static' CHECK (servicer_type IN ('static', 'discovered'));

-- Update health_status to store JSON performance metrics
-- Example health_status JSON:
-- {
--   "status": "healthy",
--   "response_time_ms": 150,
--   "error_rate": 0.02,
--   "last_error": null,
--   "performance_history": [
--     {"timestamp": "2023-01-01T10:00:00Z", "response_time": 145},
--     {"timestamp": "2023-01-01T10:05:00Z", "response_time": 155}
--   ]
-- }

-- Add indexes for servicer-specific queries
CREATE INDEX IF NOT EXISTS idx_rpc_endpoints_public_key ON rpc_endpoints(public_key);
CREATE INDEX IF NOT EXISTS idx_rpc_endpoints_servicer_type ON rpc_endpoints(servicer_type);
CREATE INDEX IF NOT EXISTS idx_rpc_endpoints_response_time ON rpc_endpoints(response_time_ms);
CREATE INDEX IF NOT EXISTS idx_rpc_endpoints_last_ping ON rpc_endpoints(last_ping_timestamp);

-- Add unique constraint for discovered servicers
ALTER TABLE rpc_endpoints 
ADD CONSTRAINT unique_discovered_servicer_public_key 
UNIQUE (public_key) 
WHERE servicer_type = 'discovered';
```

### Updated Configuration (Extending existing Config)

```go
// Extension to internal/utils/config.go
type MonitoringConfig struct {
    // Cron job configuration
    CronSchedule          string `env:"MONITORING_CRON_SCHEDULE" default:"*/5 * * * *"`
    SessionCount          int    `env:"MONITORING_SESSION_COUNT" default:"10"`
    PingTimeout           string `env:"MONITORING_PING_TIMEOUT" default:"5s"`
    MaxConcurrentPings    int    `env:"MONITORING_MAX_CONCURRENT_PINGS" default:"50"`
    
    // Database configuration
    CleanupInterval       string `env:"MONITORING_CLEANUP_INTERVAL" default:"24h"`
    MaxRetentionDays      int    `env:"MONITORING_MAX_RETENTION_DAYS" default:"30"`
    
    // Servicer discovery (references existing chain_static)
    BlockchainIDs         []int  `env:"MONITORING_BLOCKCHAIN_IDS" default:"1,2"` // References chain_static.chain_id
    GeozoneID            string `env:"MONITORING_GEOZONE_ID" default:"0001"`
    ServicerCount        int    `env:"MONITORING_SERVICER_COUNT" default:"1"`
    
    // Performance measurement
    PingRetries          int    `env:"MONITORING_PING_RETRIES" default:"3"`
    PingInterval         string `env:"MONITORING_PING_INTERVAL" default:"1s"`
}
```

### Data Flow Simplified:

1. **Discovery**: Generate sessions → Extract servicers → Insert into `rpc_endpoints` with `servicer_type='discovered'`
2. **Monitoring**: Query all `rpc_endpoints` → Ping each endpoint → Update `health_status`, `response_time_ms`, `last_ping_timestamp`
3. **Analytics**: Query `rpc_endpoints` with performance filters → Rank by `response_time_ms` → Return best performing servicers

### Integration Points with Existing System

1. **Database Integration**: Uses existing `rpc_endpoints` table structure
2. **API Integration**: Follows existing `internal/api/` patterns for REST endpoints  
3. **Configuration**: Extends existing `internal/utils/config.go` structure
4. **Models**: Uses existing `internal/models/rpc_endpoint.go` with field extensions
5. **Relay Client**: Uses existing relay client from `examples/simple_relay/relay.go`
6. **Chain Data**: References existing blockchain data in `chain_static` table

---

## Implementation Priority (Updated)

1. **Phase 1**: Extend rpc_endpoints table (Story 1) - 2-3 days
2. **Phase 2**: Servicer discovery and registration (Story 2) - 3-5 days  
3. **Phase 3**: Performance measurement for RPC endpoints (Story 3) - 3-5 days
4. **Phase 4**: Cron job orchestration (Story 4) - 3-5 days
5. **Phase 5**: Analytics API and configuration (Stories 5-6) - 3-5 days
6. **Phase 6**: Monitoring and observability (Story 7) - 2-3 days

## Estimated Timeline
- **Phase 1-2**: 1 week
- **Phase 3-4**: 1-2 weeks  
- **Phase 5-6**: 1 week
- **Total**: 3-4 weeks

## Dependencies
- Existing relay client functionality (`examples/simple_relay/relay.go`)
- Current PostgreSQL database with existing migrations
- Existing chain data in `chain_static` table (Viper Network, Ethereum)
- Current `rpc_endpoints` table structure and management 