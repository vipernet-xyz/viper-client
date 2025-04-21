package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"time"
)

func main() {
	fmt.Println("Complete Viper Network Relay Example")
	fmt.Println("==================================")

	// Configuration based on your actual viper-network setup
	viperURL := "http://127.0.0.1:8082"
	requestorPubKey := "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7"
	chain := "0001"
	geoZone := "0001"
	numServicers := 1

	client := &http.Client{Timeout: 30 * time.Second}

	// Step 1: Get current height
	fmt.Println("1. Getting current height...")
	heightResp, err := client.Post(
		viperURL+"/v1/query/height",
		"application/json",
		bytes.NewBuffer([]byte("{}")),
	)
	if err != nil {
		log.Fatalf("Error getting height: %v", err)
	}
	defer heightResp.Body.Close()

	heightBody, _ := io.ReadAll(heightResp.Body)
	var heightData struct {
		Height int64 `json:"height"`
	}
	if err := json.Unmarshal(heightBody, &heightData); err != nil {
		log.Fatalf("Error parsing height: %v", err)
	}
	height := heightData.Height
	fmt.Printf("Current height: %d\n\n", height)

	// Step 2: Create the relay with exact format from viper-go
	fmt.Println("2. Creating relay request with exact viper-go format...")

	// Generate entropy (random number)
	entropyInt, _ := rand.Int(rand.Reader, big.NewInt(9000000000000000000))
	entropy := entropyInt.Int64()

	// RPC data to send
	rpcData := `{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}`

	// Create the relay payload - directly matching viper-go format
	relayRequest := map[string]interface{}{
		"payload": map[string]interface{}{
			"data":   rpcData,
			"method": "POST",
			"path":   "",
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
		},
		"meta": map[string]interface{}{
			"block_height": height,
			"subscription": false,
			"ai":           false,
		},
		"proof": map[string]interface{}{
			"entropy":          entropy,
			"blockchain":       chain,
			"geo_zone":         geoZone,
			"servicer_pub_key": requestorPubKey,
			"token": map[string]interface{}{
				"version":              "1.0",
				"requestor_public_key": requestorPubKey,
				"client_public_key":    requestorPubKey,
			},
			"request_hash":         "dummy_hash", // This would normally be generated properly
			"session_block_height": height,
			"num_servicers":        numServicers,
			"relay_type":           1, // Regular relay
			"weight":               1,
			"address":              "86bbdc3387d63e3a9ee8d4ec3676a2fdbdc10cc2",
		},
	}

	relayJSON, _ := json.Marshal(relayRequest)
	fmt.Printf("Request: %s\n\n", string(relayJSON))

	// Step 3: Send the relay request
	fmt.Println("3. Sending relay request...")
	relayResp, err := client.Post(
		viperURL+"/v1/client/relay",
		"application/json",
		bytes.NewBuffer(relayJSON),
	)
	if err != nil {
		log.Fatalf("Error sending relay: %v", err)
	}
	defer relayResp.Body.Close()

	relayBody, _ := io.ReadAll(relayResp.Body)
	fmt.Printf("Response: %s\n\n", string(relayBody))

	fmt.Println("Example completed!")
}
