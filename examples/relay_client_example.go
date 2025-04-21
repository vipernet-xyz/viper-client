package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/dhruvsharma/viper-client/internal/rpc"
)

const (
	// Default parameters
	DefaultBlockchain = "0001" // Viper Chain ID in hex
	DefaultGeoZone    = "0001" // Global GeoZone in hex
	DefaultPubKey     = "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7"
)

func main() {
	// Get client configuration from environment or use defaults
	clientURL := os.Getenv("VIPER_CLIENT_URL")
	if clientURL == "" {
		clientURL = "http://localhost:8080"
	}

	appID := os.Getenv("VIPER_APP_ID")
	if appID == "" {
		appID = "test_app"
	}

	apiKey := os.Getenv("VIPER_API_KEY")
	if apiKey == "" {
		apiKey = "test_key"
	}

	fmt.Println("Viper Network Client Example")
	fmt.Println("----------------------------")
	fmt.Printf("Using viper-client at: %s\n\n", clientURL)

	// Create a new relay client
	client := rpc.NewRelayClient(clientURL, appID, apiKey)

	// 1. First test: Dispatch a session
	fmt.Println("1. Dispatching a session...")
	opts := rpc.RelayOptions{
		PubKey:       DefaultPubKey,
		Blockchain:   DefaultBlockchain,
		GeoZone:      DefaultGeoZone,
		NumServicers: 1,
	}

	dispatchResp, err := client.Dispatch(context.Background(), opts)
	if err != nil {
		log.Fatalf("Error dispatching session: %v", err)
	}

	fmt.Printf("Session dispatched successfully!\n")
	fmt.Printf("Session key: %s\n", dispatchResp.Session.Key)
	fmt.Printf("Block height: %d\n", dispatchResp.BlockHeight)
	fmt.Printf("Num servicers: %d\n\n", len(dispatchResp.Session.Servicers))

	// Print servicer information
	if len(dispatchResp.Session.Servicers) > 0 {
		for i, servicer := range dispatchResp.Session.Servicers {
			fmt.Printf("Servicer %d: PubKey=%s, NodeURL=%s\n",
				i+1, servicer.PublicKey, servicer.NodeURL)
		}
	} else {
		fmt.Println("Warning: No servicers returned in session")
	}
	fmt.Println()

	// 2. Second test: Execute a simple blockchain RPC call
	fmt.Println("2. Executing blockchain RPC call...")
	blockNumberResp, err := client.BlockchainRPC(
		context.Background(),
		DefaultBlockchain,
		"eth_blockNumber",
		[]interface{}{},
	)
	if err != nil {
		log.Printf("Error executing blockchain RPC: %v", err)
		fmt.Println("Attempting direct relay instead...")
	} else {
		fmt.Printf("Block number: %v\n\n", blockNumberResp)
	}

	// 3. Third test: Execute direct relay
	if len(dispatchResp.Session.Servicers) > 0 {
		servicer := dispatchResp.Session.Servicers[0]

		fmt.Println("3. Executing direct relay...")
		directOpts := rpc.RelayOptions{
			PubKey:       DefaultPubKey,
			Blockchain:   DefaultBlockchain,
			GeoZone:      DefaultGeoZone,
			NumServicers: 1,
			Data:         `{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}`,
			Method:       "POST",
			Headers:      map[string]string{"Content-Type": "application/json"},
		}

		directResp, err := client.DirectRelay(
			context.Background(),
			directOpts,
			servicer.NodeURL,
			servicer.PublicKey,
		)
		if err != nil {
			log.Printf("Error executing direct relay: %v", err)
		} else {
			fmt.Printf("Direct relay successful!\n")
			fmt.Printf("Response: %s\n\n", directResp.Response)
		}
	}

	// 4. Fourth test: Execute a complete relay
	fmt.Println("4. Executing complete relay...")
	relayOpts := rpc.RelayOptions{
		PubKey:       DefaultPubKey,
		Blockchain:   DefaultBlockchain,
		GeoZone:      DefaultGeoZone,
		NumServicers: 1,
		Data:         `{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}`,
		Method:       "POST",
		Headers:      map[string]string{"Content-Type": "application/json"},
	}

	relayResp, err := client.ExecuteRelay(context.Background(), relayOpts)
	if err != nil {
		log.Fatalf("Error executing relay: %v", err)
	}

	fmt.Printf("Relay executed successfully!\n")
	fmt.Printf("Response: %s\n", relayResp.Response)
	fmt.Printf("Signature: %s\n", relayResp.Signature)

	fmt.Println("\nExample completed successfully!")
}
