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
	ViperNetworkURL = "http://127.0.0.1:8082"
	PubKey          = "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7"
	Chain           = "0001"
	GeoZone         = "0001"
	NumServicers    = 1
)

type Session struct {
	Key       string     `json:"key"`
	Header    Header     `json:"header"`
	Servicers []Servicer `json:"servicers"`
}

type Header struct {
	RequestorPubKey    string `json:"requestor_public_key"`
	Chain              string `json:"chain"`
	GeoZone            string `json:"geo_zone"`
	NumServicers       int64  `json:"num_servicers"`
	SessionBlockHeight int64  `json:"session_block_height"`
}

type Servicer struct {
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
	NodeURL   string `json:"node_url"`
}

type DispatchResponse struct {
	Session     Session `json:"session"`
	BlockHeight int64   `json:"block_height"`
}

func main() {
	fmt.Println("Viper Network Full Relay Example")
	fmt.Println("===============================")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 1. Get current height
	fmt.Println("1. Getting current block height...")
	heightResp, err := client.Post(
		ViperNetworkURL+"/v1/query/height",
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

	// 2. Dispatch session
	fmt.Println("2. Dispatching session...")
	dispatch := map[string]interface{}{
		"requestor_public_key": PubKey,
		"chain":                Chain,
		"geo_zone":             GeoZone,
		"num_servicers":        NumServicers,
	}
	dispatchJSON, _ := json.Marshal(dispatch)

	dispatchResp, err := client.Post(
		ViperNetworkURL+"/v1/client/dispatch",
		"application/json",
		bytes.NewBuffer(dispatchJSON),
	)
	if err != nil {
		log.Fatalf("Error dispatching session: %v", err)
	}
	defer dispatchResp.Body.Close()

	dispatchBody, _ := io.ReadAll(dispatchResp.Body)
	fmt.Printf("Dispatch response: %s\n\n", string(dispatchBody))

	var dispatchData DispatchResponse
	if err := json.Unmarshal(dispatchBody, &dispatchData); err != nil {
		log.Fatalf("Error parsing dispatch: %v", err)
	}

	if len(dispatchData.Session.Servicers) == 0 {
		log.Fatalf("No servicers in session")
	}

	servicer := dispatchData.Session.Servicers[0]
	fmt.Printf("Using servicer: %s at %s\n\n", servicer.PublicKey, servicer.NodeURL)

	// 3. Execute a simple relay
	fmt.Println("3. Executing relay...")

	// Generate entropy
	// entropyInt, _ := rand.Int(rand.Reader, big.NewInt(9000000000000000000))
	// entropy := entropyInt.Int64()

	// Data to relay
	rpcData := `{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}`

	// Create a simpler relay request just for testing
	simpleRelay := map[string]interface{}{
		"blockchain":     Chain,
		"geo_zone":       GeoZone,
		"data":           rpcData,
		"session_header": dispatchData.Session.Header,
		"servicer":       servicer,
	}

	relayJSON, _ := json.Marshal(simpleRelay)

	fmt.Printf("Relay request: %s\n\n", string(relayJSON))

	relayResp, err := client.Post(
		ViperNetworkURL+"/v1/client/relay",
		"application/json",
		bytes.NewBuffer(relayJSON),
	)
	if err != nil {
		log.Fatalf("Error executing relay: %v", err)
	}
	defer relayResp.Body.Close()

	relayBody, _ := io.ReadAll(relayResp.Body)
	fmt.Printf("Relay response: %s\n\n", string(relayBody))

	fmt.Println("Example completed!")
}
