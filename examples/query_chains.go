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
	fmt.Println("Viper Network Chains Query Example")
	fmt.Println("=================================")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Query supported chains
	fmt.Println("Querying supported chains...")
	resp, err := client.Post(
		"http://127.0.0.1:8082/v1/query/supportedchains",
		"application/json",
		bytes.NewBuffer([]byte("{}")),
	)
	if err != nil {
		log.Fatalf("Error querying chains: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Supported chains response: %s\n\n", string(body))

	// Query servicers
	fmt.Println("Querying available servicers...")
	servicerReq := map[string]interface{}{
		"height": 0,
		"opts": map[string]interface{}{
			"page":     1,
			"per_page": 10,
		},
	}
	servicerReqJSON, _ := json.Marshal(servicerReq)

	resp2, err := client.Post(
		"http://127.0.0.1:8082/v1/query/servicers",
		"application/json",
		bytes.NewBuffer(servicerReqJSON),
	)
	if err != nil {
		log.Fatalf("Error querying servicers: %v", err)
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	fmt.Printf("Servicers response: %s\n\n", string(body2))

	fmt.Println("Example completed successfully!")
}
