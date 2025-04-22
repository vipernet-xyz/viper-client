package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/illegalcall/viper-client/internal/relay"
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
	fmt.Println("--------------------------")
	fmt.Printf("Using viper-client at: %s\n\n", clientURL)

	var client *relay.Client
	var err error

	// Try to use an existing private key from environment variable
	privateKeyEnv := os.Getenv("VIPER_PRIVATE_KEY")
	if privateKeyEnv != "" {
		fmt.Println("=== USING REGISTERED KEY ===")
		client, err = relay.NewClientWithSigner("", "", "", privateKeyEnv)
		if err != nil {
			log.Fatalf("Failed to create relay client with signer: %v", err)
		}
	} else {
		// Generate a new random key (for first-time setup)
		fmt.Println("=== FIRST TIME SETUP - REGISTRATION NEEDED ===")
		client, err = relay.NewClient("", "", "")
		if err != nil {
			log.Fatalf("Failed to create relay client: %v", err)
		}
	}

	// Get client information
	pubKey, err := client.GetPublicKey()
	if err != nil {
		log.Fatalf("Failed to get public key: %v", err)
	}
	fmt.Printf("Client public key: %s\n", pubKey)

	address, err := client.GetAddress()
	if err != nil {
		log.Fatalf("Failed to get address: %v", err)
	}
	fmt.Printf("Client address: %s\n", address)

	// Print registration instructions if using a new key
	if privateKeyEnv == "" {
		privateKey, err := client.GetPrivateKey()
		if err != nil {
			log.Fatalf("Failed to get private key: %v", err)
		}
		fmt.Printf("Client private key: %s\n", privateKey)
		fmt.Println("\n=== IMPORTANT: REGISTRATION STEPS ===")
		fmt.Println("1. Copy the private key above")
		fmt.Printf("2. Run: viper wallet create-account %s\n", privateKey)
		fmt.Printf("3. Run: viper wallet transfer <funded_address> %s 120000000000 viper-test \"\"\n", address)
		fmt.Printf("4. Wait ~15 seconds for the transaction to confirm\n")
		fmt.Printf("5. Run: viper requestors stake %s 120000000000 0001,0002 0001 1 viper-test\n", address)
		fmt.Println("6. Set the VIPER_PRIVATE_KEY environment variable with your private key for next run:")
		fmt.Printf("   export VIPER_PRIVATE_KEY=%s\n", privateKey)
		fmt.Println("=== END REGISTRATION STEPS ===\n")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First, get the current height from viper network
	height, err := client.GetHeight(ctx)
	if err != nil {
		log.Fatalf("Failed to get height: %v", err)
	}
	fmt.Printf("Current height: %d\n", height)

	// Create relay options
	opts := relay.Options{
		PubKey:       pubKey,
		Blockchain:   "0001", // Ethereum
		GeoZone:      "0001",
		NumServicers: 1,
		Method:       "POST",
		Headers:      map[string]string{"Content-Type": "application/json"},
		Path:         "",
	}

	// Execute a blockchain RPC call
	fmt.Println("\nExecuting blockchain RPC...")
	ethereumBlockNumber, err := client.BlockchainRPC(ctx, opts.Blockchain, "eth_blockNumber", []interface{}{})
	if err != nil {
		log.Printf("Failed to execute blockchain RPC: %v", err)
	} else {
		fmt.Printf("Latest Ethereum block number: %v\n", ethereumBlockNumber)
	}

	// Execute a direct relay to a specific servicer
	fmt.Println("\nExecuting direct relay...")
	directOpts := opts

	// Create a JSON-RPC request
	rpcRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
	}

	// Convert to JSON
	rpcJSON, err := json.Marshal(rpcRequest)
	if err != nil {
		log.Fatalf("Failed to marshal RPC request: %v", err)
	}

	directOpts.Data = string(rpcJSON)

	// Get the servicer URL from environment or use default
	servicerURL := os.Getenv("VIPER_SERVICER_URL")
	if servicerURL == "" {
		servicerURL = "http://127.0.0.1:8082"
	}

	directResp, err := client.DirectRelay(ctx, directOpts, servicerURL, pubKey)
	if err != nil {
		log.Printf("Failed to execute direct relay: %v", err)
	} else {
		fmt.Printf("Direct relay response: %s\n", directResp.Response)
	}

	// Execute a complete relay operation
	fmt.Println("\nExecuting complete relay...")
	completeOpts := opts
	completeOpts.Data = string(rpcJSON)

	relayResp, err := client.ExecuteRelay(ctx, completeOpts)
	if err != nil {
		log.Printf("Failed to execute complete relay: %v", err)
	} else {
		fmt.Printf("Complete relay response: %s\n", relayResp.Response)
	}

	fmt.Println("Example completed successfully!")
}
