# Performance Monitoring Cron Job - Jira Tasks

## Story: Database Schema for Performance Metrics
- [ ] Define `endpoint_performance_metrics` table schema (1 pointer)
- [ ] Create database migration files for `endpoint_performance_metrics` (1 pointer)

## Story: Servicer Pinging Logic
- [ ] Develop `PerformanceMonitor` service/struct (1 pointer)
- [ ] Implement `fetchServicersToPing(chainID int)` method in `PerformanceMonitor` (1 pointer)
- [ ] Implement `pingServicer(endpoint models.RpcEndpoint)` method in `PerformanceMonitor` (1 pointer)
- [ ] Implement `recordPerformanceMetric(...)` method in `PerformanceMonitor` (1 pointer)

## Story: Cron Job Implementation
- [ ] Create main package for cron job (e.g., `cmd/perf-monitor/main.go`) (1 pointer)
- [ ] Setup cron scheduling using a library (e.g., `robfig/cron/v3`) (1 pointer)
- [ ] Implement the cron job's main task function (1 pointer)
- [ ] Add configuration management (DB details, cron schedule, ping request, target `chain_id`s) (1 pointer)
- [ ] (Optional) Add Dockerfile for the cron job (1 pointer)
- [ ] Write unit tests for `pingServicer` (mock HTTP) and `PerformanceMonitor` methods (mock DB) (1 pointer)

## Story: Documentation and Jira Output
- [ ] Update project README (info on new cron job, config, `endpoint_performance_metrics` table) (1 pointer)
- [ ] Generate a separate Markdown file (`perf_monitor_jira_tasks.md`) for Jira (This task) (1 pointer)
