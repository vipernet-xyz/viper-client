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
	fmt.Println("Viper Network Direct Query Example")
	fmt.Println("================================")

	viperURL := "http://127.0.0.1:8082"
	client := &http.Client{Timeout: 10 * time.Second}

	// Perform a simple height query - this works without authentication
	fmt.Println("1. Querying current height...")
	resp, err := client.Post(
		viperURL+"/v1/query/height",
		"application/json",
		bytes.NewBuffer([]byte("{}")),
	)
	if err != nil {
		log.Fatalf("Error querying height: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Height response: %s\n\n", string(body))

	// Query supported chains - this works without authentication
	fmt.Println("2. Querying supported chains...")
	resp, err = client.Post(
		viperURL+"/v1/query/supportedchains",
		"application/json",
		bytes.NewBuffer([]byte("{}")),
	)
	if err != nil {
		log.Fatalf("Error querying chains: %v", err)
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	fmt.Printf("Chains response: %s\n\n", string(body))

	// Query servicers - this works without authentication
	fmt.Println("3. Querying servicers...")
	servicerReq := map[string]interface{}{
		"height": 0,
		"opts": map[string]interface{}{
			"page":     1,
			"per_page": 10,
		},
	}
	servicerJSON, _ := json.Marshal(servicerReq)
	resp, err = client.Post(
		viperURL+"/v1/query/servicers",
		"application/json",
		bytes.NewBuffer(servicerJSON),
	)
	if err != nil {
		log.Fatalf("Error querying servicers: %v", err)
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	fmt.Printf("Servicers response: %s\n\n", string(body))

	fmt.Println("Direct query example completed successfully!")
	fmt.Println("This confirms that viper-network is accessible for basic queries.")
	fmt.Println("For relay operations, please use the viper-go client which has the proper")
	fmt.Println("implementation of the relay protocol with correct signing and hashing.")
}
