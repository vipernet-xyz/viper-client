package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"time"

	"golang.org/x/crypto/sha3"
)

// hashRequest creates a hash of the relay payload and meta
func hashRequest(payload, meta map[string]interface{}) string {
	// Combine payload and meta
	combined, _ := json.Marshal(map[string]interface{}{
		"payload": payload,
		"meta":    meta,
	})

	// Hash using SHA3-256
	h := sha3.New256()
	h.Write(combined)
	return hex.EncodeToString(h.Sum(nil))
}

func main() {
	fmt.Println("Viper Network Relay Example with Proper Hashing")
	fmt.Println("=============================================")

	// Configuration based on your actual viper-network setup
	viperURL := "http://127.0.0.1:8082"
	requestorPubKey := "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7"
	chain := "0001"
	geoZone := "0001"
	numServicers := 1
	address := "86bbdc3387d63e3a9ee8d4ec3676a2fdbdc10cc2"

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

	// Step 2: Create the relay with proper hashing
	fmt.Println("2. Creating relay request with proper hash...")

	// Generate entropy (random number)
	entropyInt, _ := rand.Int(rand.Reader, big.NewInt(9000000000000000000))
	entropy := entropyInt.Int64()

	// RPC data to send
	rpcData := `{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}`

	// Create payload and meta
	payload := map[string]interface{}{
		"data":   rpcData,
		"method": "POST",
		"path":   "",
		"headers": map[string]string{
			"Content-Type": "application/json",
		},
	}

	meta := map[string]interface{}{
		"block_height": height,
		"subscription": false,
		"ai":           false,
	}

	// Calculate the request hash properly
	requestHash := hashRequest(payload, meta)
	fmt.Printf("Calculated request hash: %s\n", requestHash)

	// Create token
	token := map[string]interface{}{
		"version":              "1.0",
		"requestor_public_key": requestorPubKey,
		"client_public_key":    requestorPubKey,
		"signature":            "", // This would normally be properly signed
	}

	// Create the proof
	proof := map[string]interface{}{
		"request_hash":         requestHash,
		"entropy":              entropy,
		"session_block_height": height,
		"servicer_pub_key":     requestorPubKey,
		"blockchain":           chain,
		"token":                token,
		"geo_zone":             geoZone,
		"num_servicers":        numServicers,
		"relay_type":           1, // Regular relay
		"weight":               1,
		"address":              address,
	}

	// Create the complete relay request
	relayRequest := map[string]interface{}{
		"payload": payload,
		"meta":    meta,
		"proof":   proof,
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
