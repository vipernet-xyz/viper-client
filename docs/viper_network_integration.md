# Viper Network Integration

This document outlines how Viper Client integrates with the Viper Network decentralized RPC infrastructure.

## Overview

Viper Client acts as a middleware layer between applications and the Viper Network, providing:

1. Authentication and rate limiting
2. Format conversion between JSON-RPC and Viper Network's API format
3. Endpoint health monitoring and failover
4. A unified API for both Viper Network and other blockchains

## Supported Endpoints

The following Viper Network endpoints are available through Viper Client:

| Endpoint | Description | HTTP Method | URL |
|----------|-------------|------------|-----|
| Height | Get current block height | POST | `/viper/height` |
| Relay | Relay requests to blockchains | POST | `/viper/relay` |
| Servicers | Get information about servicers | POST | `/viper/servicers` |
| Block | Get block information | POST | `/viper/block` |
| TX | Get transaction information | POST | `/viper/tx` |
| Account | Get account information | POST | `/viper/account` |
| Supported Chains | Get list of supported chains | POST | `/viper/supportedchains` |
| Dispatch | Dispatch requests | POST | `/viper/dispatch` |
| Challenge | Generate challenge | POST | `/viper/challenge` |
| WebSocket | WebSocket connection | GET | `/viper/websocket` |

Additionally, standard JSON-RPC requests can be sent to `/rpc/1` (where 1 is the chain ID for Viper Network) and they will be automatically converted to the appropriate Viper Network format.

## Configuration

Viper Network is configured as a chain in the database with the following properties:

- Chain ID: 0001
- Name: Viper Network
- Symbol: VPR
- Network Type: testnet
- Is EVM: false

## Connection Details

Viper Client connects to the Viper Network via the following endpoints:

1. Primary: `http://localhost:8082` - API endpoint
2. Secondary: `http://localhost:26657` - Tendermint RPC endpoint

You can change these values by updating the `rpc_endpoints` table in the database.

## Authentication

All requests to Viper Network endpoints require authentication using one of the following methods:

1. HTTP Headers:
   - `X-App-ID`: Your application identifier
   - `X-API-Key`: Your API key

2. Query Parameters:
   - `appId`: Your application identifier
   - `apiKey`: Your API key

## Request/Response Format

### Standard JSON-RPC Format

When using `/rpc/1`, you can use standard JSON-RPC format:

```json
{
  "jsonrpc": "2.0",
  "method": "eth_blockNumber",
  "params": [],
  "id": 1
}
```

This will be automatically converted to the appropriate Viper Network format.

### Direct Viper Network Format

When using the `/viper/*` endpoints, you should use Viper Network's native format:

```json
{
  "height": 0,
  "blockchain": "0002",
  "data": "{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}",
  "method": "POST",
  "headers": {
    "Content-Type": "application/json"
  }
}
```

## Example: Getting the Current Height

Using JSON-RPC format:

```bash
curl -X POST http://localhost:8080/rpc/1 \
  -H "Content-Type: application/json" \
  -H "X-App-ID: your-app-id" \
  -H "X-API-Key: your-api-key" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

Using direct Viper Network format:

```bash
curl -X POST http://localhost:8080/viper/height \
  -H "Content-Type: application/json" \
  -H "X-App-ID: your-app-id" \
  -H "X-API-Key: your-api-key" \
  -d '{}'
```

## Implementation Details

1. **Dispatcher**: The `Dispatcher` component checks if a request is for the Viper Network and routes it accordingly.

2. **Format Conversion**: The `ConvertJSONRPCToViperFormat` and `ConvertViperResponseToJSONRPC` functions handle conversion between formats.

3. **ViperNetworkHandler**: Handles direct interaction with the Viper Network endpoints.

4. **Health Monitoring**: Viper Client tracks the health of Viper Network endpoints and fails over to alternatives if needed.

## Swagger Documentation

To facilitate the development of client applications that integrate with the Viper Network, we provide a comprehensive OpenAPI (Swagger) specification that documents all available endpoints and their usage.

### OpenAPI Specification

The complete OpenAPI specification is available in the `docs/swagger/viper-api.yaml` file. This specification documents:

1. All available endpoints
2. Request and response schemas
3. Authentication requirements
4. Error responses
5. Examples

### Using the Swagger Documentation

Developers can utilize this documentation in several ways:

1. **Swagger UI**: You can visualize and interact with the API by loading the YAML file into Swagger UI:
   - Use the online Swagger Editor at [https://editor.swagger.io/](https://editor.swagger.io/)
   - Or integrate Swagger UI into your application using libraries like `swagger-ui-express` for Node.js

2. **Code Generation**: Generate client libraries for various programming languages:
   ```bash
   # Using OpenAPI Generator
   openapi-generator generate -i docs/swagger/viper-api.yaml -g javascript -o ./generated-client
   ```

3. **Postman Collection**: Import the YAML file into Postman to create a collection of requests:
   - In Postman, go to Import > File > Upload Files
   - Select the YAML file
   - Start making API requests

### Key Endpoints

The OpenAPI specification documents the following key endpoints:

| Endpoint | Description | Request Schema | Response Schema |
|----------|-------------|----------------|-----------------|
| `/viper/height` | Get current block height | `ViperHeightRequest` | `ViperHeightResponse` |
| `/viper/relay` | Relay a request to a blockchain | `ViperRelayRequest` | `ViperRelayResponse` |
| `/viper/servicers` | Get servicers information | `ViperServicersRequest` | `ViperServicersResponse` |
| `/viper/supportedchains` | Get supported chains | `ViperSupportedChainsRequest` | `ViperSupportedChainsResponse` |
| `/viper/block` | Get block information | `ViperBlockRequest` | `ViperBlockResponse` |
| `/viper/tx` | Get transaction information | `ViperTxRequest` | `ViperTxResponse` |
| `/viper/account` | Get account information | `ViperAccountRequest` | `ViperAccountResponse` |
| `/viper/dispatch` | Dispatch requests | Object | Object |
| `/viper/challenge` | Generate challenge | Object | Object |
| `/viper/websocket` | WebSocket connection | N/A | WebSocket connection |
| `/rpc/{chainId}` | Send JSON-RPC request | JSON-RPC 2.0 request | JSON-RPC 2.0 response |

### Authentication in Swagger

All endpoints require authentication using API keys:

```yaml
security:
  - ApiKeyAuth: []
    AppIdAuth: []
parameters:
  - name: X-App-ID
    in: header
    required: true
    schema:
      type: string
  - name: X-API-Key
    in: header
    required: true
    schema:
      type: string
```

### Example Swagger Request and Response

#### Example for `/viper/height` endpoint:

Request:
```json
{
  "blockchain": "0001"
}
```

Response:
```json
{
  "height": 1234567,
  "hash": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
}
```

#### Example for `/rpc/{chainId}` endpoint:

Request:
```json
{
  "jsonrpc": "2.0",
  "method": "eth_blockNumber",
  "params": [],
  "id": 1
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "result": "0x12d687",
  "id": 1
}
```

### Error Handling

The Swagger documentation also describes standardized error responses:

```yaml
ErrorResponse:
  type: object
  properties:
    error:
      type: string
      description: Error message
    code:
      type: integer
      description: Error code
```

Common error codes:
- 401: Unauthorized
- 500: Server error

## CRUD Applications with Viper Network

CRUD applications can:

1. Use the `/rpc/1` endpoint for standard JSON-RPC interactions with smart contracts on networks supported by Viper
2. Use the `/viper/*` endpoints for direct access to Viper Network's API
3. Leverage Viper Client for authentication, rate limiting, and endpoint management

All interactions are routed through the decentralized Viper Network instead of centralized RPC providers.
