package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dhruvsharma/viper-client/internal/relay"
)

const (
	// After registering, replace this with your actual private key
	// Only use the SEED portion (first 64 hex chars) of the ED25519 key
	privateKey = "b3cc669e939f6c8d51d34129d4445777eb3caf9c311c1947e0e927178848205d" // Leave empty to generate a new key

	// Sample servicer URL
	servicerURL = "http://127.0.0.1:8082"
)

func main() {
	var client *relay.Client
	var err error

	if privateKey == "" {
		// Generate a new random key (for first-time setup)
		client, err = relay.NewClient("", "", "")
		if err != nil {
			log.Fatalf("Failed to create relay client: %v", err)
		}
		fmt.Println("=== FIRST TIME SETUP - REGISTRATION NEEDED ===")
	} else {
		// Use the registered key
		client, err = relay.NewClientWithSigner("", "", "", privateKey)
		if err != nil {
			log.Fatalf("Failed to create relay client with signer: %v", err)
		}
		fmt.Println("=== USING REGISTERED KEY ===")
	}

	// Get and print key information
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
	if privateKey == "" {
		privateKey, err := client.GetPrivateKey()
		if err != nil {
			log.Fatalf("Failed to get private key: %v", err)
		}
		fmt.Printf("Client private key (seed only): %s\n", privateKey)
		fmt.Println("\n=== IMPORTANT: REGISTRATION STEPS ===")
		fmt.Println("1. Copy the private key above")
		fmt.Printf("2. Run: viper wallet create-account %s\n", privateKey)

		// This will create a new address in the Viper wallet
		fmt.Println("3. Note the new address created by the wallet command")
		fmt.Println("4. Fund that address (not the client address above):")
		fmt.Println("   viper wallet transfer <funded_address> <NEW_WALLET_ADDRESS> 120000000000 viper-test \"\"")
		fmt.Println("5. Stake that address as a requestor:")
		fmt.Println("   viper requestors stake <NEW_WALLET_ADDRESS> 120000000000 0001,0002 0001 1 viper-test")
		fmt.Println("6. Update this program with your private key in the privateKey constant")

		// Instructions for advanced users to access relays with their own address
		fmt.Println("\n==== FOR ADVANCED USERS ONLY ====")
		fmt.Println("If you want to use the client's address directly (not recommended):")
		fmt.Printf("- Update the PubKey field in your relay options to use the registered wallet's public key\n")
		fmt.Println("==== END ADVANCED SECTION ====")

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
		// If using a registered wallet, you should replace this with that wallet's public key
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
	directOpts.GeoZone = "0001" // Ensure this is explicitly set

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
