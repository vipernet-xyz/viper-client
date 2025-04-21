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

func main() {
	// Configuration based on your actual viper-network setup
	viperURL := "http://127.0.0.1:8082"
	servicerPublicKey := "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7"
	servicerAddress := "86bbdc3387d63e3a9ee8d4ec3676a2fdbdc10cc2"
	chain := "0001"
	// geoZone := "0001"

	fmt.Println("Simple Viper Network Example")
	fmt.Println("===========================")

	client := &http.Client{Timeout: 10 * time.Second}

	// Try the simplest blockchain request possible
	fmt.Println("Sending a simple blockchain request...")

	// Create the JSON-RPC request
	rpcData := `{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}`

	// Create the relay request
	relayReq := map[string]interface{}{
		"blockchain": chain,
		"data":       rpcData,
		"address":    servicerAddress,
		"pub_key":    servicerPublicKey,
	}

	reqJSON, _ := json.Marshal(relayReq)

	// Send the request
	resp, err := client.Post(
		viperURL+"/v1/client/relay",
		"application/json",
		bytes.NewBuffer(reqJSON),
	)

	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: %s\n", string(respBody))

	fmt.Println("Example completed!")
}
