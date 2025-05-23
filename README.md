# Apollo Project

This project includes various services for interacting with and monitoring RPC endpoints.

## Performance Monitoring Cron Job

### Overview

The Performance Monitoring Cron Job is a background service that periodically pings active RPC endpoints (servicers) to measure their performance and availability. It records metrics such as response time and HTTP status codes, storing this data for analysis and alerting. This helps in understanding the reliability and speed of the monitored RPC services.

### New Database Table: `endpoint_performance_metrics`

To store the historical performance data, a new database table has been introduced:

*   **Table Name:** `endpoint_performance_metrics`
*   **Purpose:** This table stores the results of each ping attempt to an RPC endpoint, providing a time-series dataset of performance.
*   **Key Columns:**
    *   `id`: Primary key for the record.
    *   `rpc_endpoint_id`: Foreign key referencing the `rpc_endpoints` table, linking the metric to a specific servicer.
    *   `ping_timestamp`: Timestamp indicating when the ping was performed.
    *   `response_time_ms`: The duration of the ping request in milliseconds.
    *   `http_status_code`: The HTTP status code returned by the endpoint. This can be 0 if the request failed before receiving an HTTP response (e.g., timeout).
    *   `error_message`: Any error message captured during the ping attempt (e.g., timeout error, HTTP error details). Null if the ping was successful.
    *   `created_at`: Timestamp of when the record was created.

### Configuration

The cron job is configured using environment variables. Below are the key variables used by the `cmd/perf-monitor/main.go` application:

*   **`PERFMON_DB_URL`**: PostgreSQL database connection string.
    *   Example: `postgres://user:password@host:port/dbname?sslmode=disable`
*   **`PERFMON_CRON_SCHEDULE`**: Cron schedule string that defines how often the monitoring task runs. Uses standard cron format or predefined schedules like `@every <duration>`.
    *   Example: `@every 15m` (runs every 15 minutes)
    *   Example: `0 * * * *` (runs at the beginning of every hour)
*   **`PERFMON_PING_JSON_PAYLOAD`**: The JSON-RPC payload string to use for pinging the endpoints.
    *   Default: `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
    *   **Note:** If setting this directly as an environment variable in a shell, the JSON string must be properly escaped. It's often easier to manage this in a `.env` file or a deployment configuration system that handles JSON strings correctly.
*   **`PERFMON_PING_TIMEOUT_SECONDS`**: Timeout for each ping request in seconds.
    *   Default: `10`
*   **`PERFMON_TARGET_CHAIN_IDS`**: A comma-separated list of specific chain IDs to monitor.
    *   Example: `"1,56,137"`
    *   If left empty or not set, the monitor will fetch all distinct chain IDs from the `rpc_endpoints` table that are marked `is_active = true` and belong to the `geozone = 'IND'`, and monitor those.
*   **`PERFMON_LOG_LEVEL`**: (This variable was not explicitly in the `main.go` config struct, but is good practice to mention if logging is configurable, or could be added) Controls the logging verbosity.
    *   Example: `"info"`, `"debug"`, `"warn"`, `"error"` (Actual available levels depend on the logger implementation, e.g. logrus). *Assuming logrus is used as per `main.go`, this would be set via code or a different env var if not directly supported by `envconfig` for logrus level.* For `main.go` as implemented, log level is hardcoded to `InfoLevel` but can be changed in code.

### How to Run

1.  **Build the Executable:**
    Navigate to the project root and run:
    ```bash
    go build -o perf-monitor ./cmd/perf-monitor
    ```

2.  **Run the Executable:**
    Ensure all required environment variables (listed above) are set, then execute the compiled binary:
    ```bash
    ./perf-monitor
    ```
    The application will start, and the cron job will trigger based on the `PERFMON_CRON_SCHEDULE`.

3.  **Running with Docker (Optional):**
    If a Dockerfile is available for the `perf-monitor` service, you can build and run it as a container. This typically involves:
    ```bash
    docker build -t perf-monitor-app -f <path_to_dockerfile> .
    docker run -d --env-file .env perf-monitor-app 
    ```
    (The exact Docker commands might vary based on the Dockerfile's content and your environment setup. A Dockerfile was an optional item in the plan, so its availability is conditional.)
