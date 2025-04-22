# Viper Client Relay Functionality

This document explains how the relay functionality works in the Viper Client, which enables communication with the Viper Network.

## Overview

The relay system in Viper Client provides a bridge between clients and the Viper Network, allowing for authenticated and secure communication with blockchain networks. The relay functionality follows a specific flow:

1. **Session Dispatch**: Establishes a session with the Viper Network to get servicer information
2. **Authentication**: Generates and signs an Application Authentication Token (AAT)
3. **Relay Building**: Constructs a properly formatted relay request with cryptographic proofs
4. **Relay Execution**: Sends the relay to the appropriate servicer node in the Viper Network

## API Endpoints

The relay functionality exposes the following endpoints:

- **POST /relay/dispatch**: Creates a new session with the Viper Network
- **POST /relay/direct**: Sends a relay directly to a specific servicer
- **POST /relay/execute**: Executes the complete relay process (dispatch + relay)

## Client Usage

The viper-client library provides a convenient client for interacting with the relay API. Here's how to use it:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/illegalcall/viper-client/internal/relay"
)

func main() {
	// Create a new relay client
	client, err := relay.NewClient(
		"http://localhost:8080",
		"your_app_id",
		"your_api_key",
	)
	if err != nil {
		log.Fatalf("Error creating relay client: %v", err)
	}

	// Generate a random signer or use an existing private key
	// client, err := relay.NewClientWithSigner(
	//     "http://localhost:8080", 
	//     "your_app_id", 
	//     "your_api_key",
	//     "your_private_key"
	// )

	// Set up relay options
	opts := relay.Options{
		PubKey:       "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7",
		Blockchain:   "0001", // Viper chain ID
		GeoZone:      "0001", // Global geo zone
		NumServicers: 1,
		Data:         `{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}`,
		Method:       "POST",
		Headers:      map[string]string{"Content-Type": "application/json"},
	}

	// Execute the relay
	resp, err := client.ExecuteRelay(context.Background(), opts)
	if err != nil {
		log.Fatalf("Error executing relay: %v", err)
	}

	fmt.Printf("Relay response: %s\n", resp.Response)

	// Alternatively, use the simplified RPC method
	blockNumber, err := client.BlockchainRPC(
		context.Background(),
		"0001",            // Blockchain ID
		"eth_blockNumber", // RPC method
		[]interface{}{},   // Params
	)
	if err != nil {
		log.Fatalf("Error getting block number: %v", err)
	}

	fmt.Printf("Current block number: %v\n", blockNumber)

	// Get current height of the Viper network
	height, err := client.GetHeight(context.Background())
	if err != nil {
		log.Fatalf("Error getting height: %v", err)
	}
	fmt.Printf("Current Viper network height: %d\n", height)
}
```

## Implementation Details

### Core Components

1. **Relay Client (`relay.Client`)**: 
   - Handles the communication with the Viper Network
   - Manages session dispatching, AAT generation, and relay requests
   - Signs and verifies cryptographic proofs

2. **Crypto Signer (`utils.Signer`)**:
   - Generates and manages cryptographic keys (ED25519)
   - Signs messages with the private key
   - Provides address and public key information

3. **Relay Handler (`api.RelayHandler`)**:
   - Exposes the relay functionality through HTTP endpoints
   - Handles authentication and request validation
   - Coordinates the relay process

### Data Flow

1. Client submits a relay request to one of the endpoints
2. The server authenticates the request using the provided API keys
3. For full execution (/relay/execute):
   - A session is dispatched to get servicer information
   - An AAT is generated and signed for authentication
   - A relay request is constructed with the necessary proofs
   - The relay is sent to the servicer node
   - The response is returned to the client

## Request Format

### Dispatch Request

```json
{
  "pub_key": "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7",
  "blockchain": "0001",
  "geo_zone": "0001",
  "num_servicers": 1
}
```

### Relay Execution Request

```json
{
  "pub_key": "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7",
  "blockchain": "0001",
  "geo_zone": "0001",
  "num_servicers": 1,
  "data": "{\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1,\"jsonrpc\":\"2.0\"}",
  "method": "POST",
  "path": "",
  "headers": {
    "Content-Type": "application/json"
  }
}
```

### Direct Relay Request

```json
{
  "pub_key": "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7",
  "blockchain": "0001",
  "geo_zone": "0001",
  "num_servicers": 1,
  "data": "{\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1,\"jsonrpc\":\"2.0\"}",
  "method": "POST",
  "path": "",
  "headers": {
    "Content-Type": "application/json"
  },
  "servicer_url": "http://node1.viper.network",
  "servicer_pub_key": "b1c789d0ba265dced08739da7a33d2d79aeacc98358e560b4b751de7586b93e8"
}
```

## Testing the Relay Functionality

Follow these steps to test the relay functionality:

1. **Start the Viper Client server**:
   ```
   make run
   ```

2. **Send a test dispatch request**:
   ```bash
   curl -X POST http://localhost:8080/relay/dispatch \
     -H "Content-Type: application/json" \
     -H "X-App-ID: your_app_id" \
     -H "X-API-Key: your_api_key" \
     -d '{
        "pub_key": "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7",
        "blockchain": "0001",
        "geo_zone": "0001",
        "num_servicers": 1
     }'
   ```

3. **Execute a complete relay**:
   ```bash
   curl -X POST http://localhost:8080/relay/execute \
     -H "Content-Type: application/json" \
     -H "X-App-ID: your_app_id" \
     -H "X-API-Key: your_api_key" \
     -d '{
        "pub_key": "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7",
        "blockchain": "0001",
        "geo_zone": "0001",
        "num_servicers": 1,
        "data": "{\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1,\"jsonrpc\":\"2.0\"}",
        "method": "POST",
        "headers": {
          "Content-Type": "application/json"
        }
     }'
   ```

## Error Handling

The relay endpoints return standard HTTP status codes with descriptive error messages:

- **400 Bad Request**: Invalid request format or missing required fields
- **401 Unauthorized**: Missing or invalid authentication credentials
- **500 Internal Server Error**: Error processing the relay or communicating with the Viper Network

## Security Considerations

1. **Authentication**: All relay endpoints require valid API keys
2. **Cryptographic Security**: All relay requests are cryptographically signed
3. **Validation**: Request parameters are validated before processing

## Troubleshooting

Common issues and solutions:

1. **"Failed to dispatch session"**: Check that the Viper Network is accessible and the blockchain ID is valid
2. **"Failed to generate AAT"**: Ensure the public key format is correct
3. **"No servicers available"**: The Viper Network may not have servicers for the requested blockchain

## Future Improvements

Potential enhancements for the relay functionality:

1. Add support for batch relay requests
2. Implement caching for session information
3. Add retry logic for failed relay attempts
4. Enhance metrics and logging 