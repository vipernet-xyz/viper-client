package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	ViperNetworkURL = "http://127.0.0.1:8082" // Default viper-network endpoint
	RPCMethod       = "eth_blockNumber"       // Simple RPC method to test
)

// Simple client that connects directly to viper-network
func main() {
	fmt.Println("Viper Network Direct Connect Demo")
	fmt.Println("=================================")

	// Create HTTP client
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Check if viper-network is running by checking height
	fmt.Println("1. Testing connection to viper-network...")
	heightResp, err := client.Post(
		ViperNetworkURL+"/v1/query/height",
		"application/json",
		bytes.NewBuffer([]byte("{}")),
	)

	if err != nil {
		log.Fatalf("Error connecting to viper-network: %v", err)
	}
	defer heightResp.Body.Close()

	heightBody, _ := io.ReadAll(heightResp.Body)

	if heightResp.StatusCode != http.StatusOK {
		log.Fatalf("Error from viper-network: %s (status %d)", string(heightBody), heightResp.StatusCode)
	}

	var heightData struct {
		Height int64 `json:"height"`
	}
	if err := json.Unmarshal(heightBody, &heightData); err != nil {
		log.Fatalf("Error parsing height response: %v", err)
	}

	fmt.Printf("Connection successful! Current height: %d\n\n", heightData.Height)

	// Try a simple JSON-RPC call with proper relay format
	fmt.Println("2. Testing blockchain RPC call...")

	// Create the RPC call data
	rpcData := fmt.Sprintf(`{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}`)

	// Create the relay request with all required fields
	relayRequest := map[string]interface{}{
		"blockchain":    "0001",  // Chain ID
		"geo_zone":      "0001",  // GeoZone
		"num_servicers": 1,       // Number of servicers
		"data":          rpcData, // The actual RPC data
		"method":        "POST",  // HTTP method
		"path":          "",      // Path (empty for standard RPC)
		"headers": map[string]string{
			"Content-Type": "application/json",
		},
	}

	relayJSON, _ := json.Marshal(relayRequest)

	fmt.Printf("Sending relay request: %s\n\n", string(relayJSON))

	relayResp, err := client.Post(
		ViperNetworkURL+"/v1/client/relay",
		"application/json",
		bytes.NewBuffer(relayJSON),
	)

	if err != nil {
		log.Fatalf("Error making relay request: %v", err)
	}
	defer relayResp.Body.Close()

	relayBody, _ := io.ReadAll(relayResp.Body)

	fmt.Printf("Relay response: %s\n\n", string(relayBody))

	fmt.Println("Demo completed successfully!")
	fmt.Println("This confirms that viper-network is accessible at", ViperNetworkURL)
}
