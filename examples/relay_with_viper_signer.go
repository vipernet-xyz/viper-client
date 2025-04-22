package main

import (
	"context"
	"fmt"
	"log"
	"os"

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

	fmt.Println("Viper Network Client with Viper Signer Example")
	fmt.Println("---------------------------------------------")
	fmt.Printf("Using viper-client at: %s\n\n", clientURL)

	// Either load an existing private key or generate a new one
	var viperSigner *signer.Signer
	var err error

	// Use an existing private key if provided (from a registered account)
	privateKeyEnv := os.Getenv("VIPER_PRIVATE_KEY")
	if privateKeyEnv != "" {
		fmt.Println("Using provided private key")
		viperSigner, err = signer.NewSignerFromPrivateKey(privateKeyEnv)
		if err != nil {
			log.Fatalf("Error creating signer from private key: %v", err)
		}
	} else {
		// Create a new random signer
		fmt.Println("Generating new random keys")
		viperSigner, err = signer.NewRandomSigner()
		if err != nil {
			log.Fatalf("Error creating random signer: %v", err)
		}
	}

	// Print signer information
	fmt.Printf("Address: %s\n", viperSigner.GetAddress())
	fmt.Printf("Public Key: %s\n", viperSigner.GetPublicKey())

	// Only print private key for new keys - for registration purposes
	if privateKeyEnv == "" {
		fmt.Printf("Private Key: %s\n", viperSigner.GetPrivateKey())
		fmt.Println("\n=== IMPORTANT: REGISTRATION STEPS ===")
		fmt.Println("1. Copy the private key above")
		fmt.Printf("2. Run: viper wallet create-account %s\n", viperSigner.GetPrivateKey())
		fmt.Printf("3. Run: viper wallet transfer <funded_address> %s 120000000000 viper-test \"\"\n", viperSigner.GetAddress())
		fmt.Printf("4. Run: viper requestors stake %s 120000000000 0001,0002 0001 1 viper-test\n", viperSigner.GetAddress())
		fmt.Println("5. Set the VIPER_PRIVATE_KEY environment variable with your private key for next run")
		fmt.Println("=== END REGISTRATION STEPS ===\n")
	}

	// Create a new relay client - we'll use the direct client instead of through our own signer
	client, err := relay.NewClient("", "", "")
	if err != nil {
		log.Fatalf("Failed to create relay client: %v", err)
	}

	// Get the current height from viper network
	ctx := context.Background()
	height, err := client.GetHeight(ctx)
	if err != nil {
		log.Fatalf("Failed to get height: %v", err)
	}
	fmt.Printf("Current height: %d\n", height)

	// Create relay options using the Viper signer's public key
	opts := relay.Options{
		PubKey:       viperSigner.GetPublicKey(),
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

	// Execute a complete relay operation
	fmt.Println("\nExecuting complete relay...")
	relayResp, err := client.ExecuteRelay(ctx, opts)
	if err != nil {
		log.Printf("Failed to execute complete relay: %v", err)
	} else {
		fmt.Printf("Complete relay response: %s\n", relayResp.Response)
	}

	fmt.Println("\nExample completed successfully!")
}
