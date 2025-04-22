package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/illegalcall/viper-client/internal/relay"
)

// Constants for relay configuration
const (
	// Default chain parameters
	BlockchainID  = "0001" // Ethereum
	GeoZoneID     = "0001" // Global zone
	ServicerCount = 1      // Number of servicers to include
)

// Signer struct for handling cryptographic signing
type Signer struct {
	address    string
	publicKey  string
	privateKey string
}

// NewRandomSigner returns a Signer with random keys
func NewRandomSigner() (*Signer, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}

	address, err := GetAddressFromDecodedPublickey(publicKey)
	if err != nil {
		return nil, err
	}

	return &Signer{
		address:    address,
		publicKey:  hex.EncodeToString(publicKey),
		privateKey: hex.EncodeToString(privateKey),
	}, nil
}

// NewSignerFromPrivateKey returns Signer from private key
func NewSignerFromPrivateKey(privateKey string) (*Signer, error) {
	if !ValidatePrivateKey(privateKey) {
		return nil, fmt.Errorf("invalid private key")
	}

	publicKey := PublicKeyFromPrivate(privateKey)

	address, err := GetAddressFromPublickey(publicKey)
	if err != nil {
		return nil, err
	}

	return &Signer{
		address:    address,
		publicKey:  publicKey,
		privateKey: privateKey,
	}, nil
}

// GetAddress returns address value
func (s *Signer) GetAddress() string {
	return s.address
}

// GetPublicKey returns public key value
func (s *Signer) GetPublicKey() string {
	return s.publicKey
}

// GetPrivateKey returns private key value
func (s *Signer) GetPrivateKey() string {
	return s.privateKey
}

// Sign returns a signed request as encoded hex string
func (s *Signer) Sign(payload []byte) (string, error) {
	decodedKey, err := hex.DecodeString(s.privateKey)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(ed25519.Sign(decodedKey, payload)), nil
}

// Helper functions for address and key handling
func GetAddressFromDecodedPublickey(publicKey ed25519.PublicKey) (string, error) {
	return AddressFromHex(hex.EncodeToString(publicKey)), nil
}

func GetAddressFromPublickey(publicKey string) (string, error) {
	return AddressFromHex(publicKey), nil
}

// AddressFromHex derives an address from a hex public key by taking the last 20 bytes
func AddressFromHex(hexString string) string {
	// Take the last 40 characters (20 bytes) of the public key
	if len(hexString) >= 40 {
		return strings.ToLower(hexString[len(hexString)-40:])
	}
	return ""
}

func PublicKeyFromPrivate(privateKey string) string {
	privKeyBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return ""
	}

	// For ED25519, if the private key is 64 bytes, the public key is the last 32 bytes
	if len(privKeyBytes) == 64 {
		return hex.EncodeToString(privKeyBytes[32:])
	}

	// If it's a 32-byte seed, derive the public key
	if len(privKeyBytes) == 32 {
		privKey := ed25519.NewKeyFromSeed(privKeyBytes)
		pubKey := privKey[32:]
		return hex.EncodeToString(pubKey)
	}

	return ""
}

func ValidatePrivateKey(privateKey string) bool {
	if len(privateKey) == 0 {
		return false
	}

	// Check if it's a valid hex string
	_, err := hex.DecodeString(privateKey)
	if err != nil {
		return false
	}

	// ED25519 private keys are 64 bytes (full key) or 32 bytes (seed)
	return len(privateKey) == 128 || len(privateKey) == 64
}

func main() {
	log.Println("Viper Network Simple Relay Example")
	log.Println("--------------------------------")

	// Check if a private key was provided via environment variable
	privateKeyEnv := os.Getenv("VIPER_PRIVATE_KEY")

	// We'll keep the signer for reference but we primarily use the relay client
	var err error

	// Create a new relay client or use the signer with an existing client
	var client *relay.Client

	if privateKeyEnv != "" {
		// Use the private key with the client
		client, err = relay.NewClientWithSigner("", "", "", privateKeyEnv)
		if err != nil {
			log.Fatalf("Error creating client with signer: %v", err)
		}
		log.Println("Using provided private key from environment")
	} else {
		// Create a new client
		client, err = relay.NewClient("", "", "")
		if err != nil {
			log.Fatalf("Error creating client: %v", err)
		}
		log.Println("Generating new random keys...")
	}

	// Get client information
	pubKey, err := client.GetPublicKey()
	if err != nil {
		log.Fatalf("Error getting public key: %v", err)
	}

	address, err := client.GetAddress()
	if err != nil {
		log.Fatalf("Error getting address: %v", err)
	}

	log.Printf("Client address: %s", address)
	log.Printf("Client public key: %s", pubKey)

	// Show registration instructions if using a new key
	if privateKeyEnv == "" {
		privateKey, err := client.GetPrivateKey()
		if err != nil {
			log.Fatalf("Error getting private key: %v", err)
		}
		log.Printf("Client private key: %s", privateKey)

		log.Println("\n=== REGISTRATION INSTRUCTIONS ===")
		log.Printf("1. Create account: viper wallet create-account %s", privateKey)
		log.Printf("2. Fund account: viper wallet transfer <funded_addr> %s 120000000000 viper-test \"\"", address)
		log.Printf("3. Wait: sleep 15")
		log.Printf("4. Stake account: viper requestors stake %s 120000000000 0001,0002 0001 1 viper-test", address)
		log.Println("=== END REGISTRATION INSTRUCTIONS ===\n")
	} else {
		log.Println("\n=== USING REGISTERED ACCOUNT ===")
		log.Printf("Using registered account with address: %s", address)
		log.Println("===================================\n")
	}

	// Check current network height
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	height, err := client.GetHeight(ctx)
	if err != nil {
		log.Printf("Error getting height: %v", err)
	} else {
		log.Printf("Current network height: %d", height)
	}

	// Try to dispatch a session
	log.Printf("Dispatching session...")

	opts := relay.Options{
		PubKey:       pubKey,
		Blockchain:   BlockchainID,
		GeoZone:      GeoZoneID,
		NumServicers: ServicerCount,
		Method:       "POST",
		Headers:      map[string]string{"Content-Type": "application/json"},
	}

	// Use SyncedDispatch to ensure session has correct block height
	dispatchResp, err := client.SyncedDispatch(ctx, opts)
	if err != nil {
		log.Printf("Error dispatching session: %v", err)
	} else {
		log.Printf("Session dispatched successfully!")
		log.Printf("Session key: %s", dispatchResp.Session.Key)
		log.Printf("Session header: %+v", dispatchResp.Session.Header)
		log.Printf("Servicers in session: %d", len(dispatchResp.Session.Servicers))

		// If we have servicers, print their info
		for i, servicer := range dispatchResp.Session.Servicers {
			log.Printf("Servicer %d: PublicKey=%s, URL=%s",
				i+1, servicer.PublicKey, servicer.NodeURL)
		}

		// Try a simple RPC query if we have a session
		if len(dispatchResp.Session.Servicers) > 0 {
			// Create a simple JSON-RPC request
			rpcRequest := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "eth_blockNumber",
				"params":  []interface{}{},
			}

			// Convert to JSON
			rpcJSON, err := json.Marshal(rpcRequest)
			if err != nil {
				log.Fatalf("Error marshaling RPC request: %v", err)
			}

			// Set the data for the relay
			opts.Data = string(rpcJSON)

			// Try to build a relay
			relay, err := client.BuildRelay(ctx, &dispatchResp.Session, opts)
			if err != nil {
				log.Printf("Error building relay: %v", err)
			} else {
				log.Printf("Relay built successfully")
				log.Printf("Relay proof hash: %s", relay.Proof.RequestHash)

				// Send the relay to the first servicer
				servicer := dispatchResp.Session.Servicers[0]
				resp, err := client.SendRelay(ctx, relay, servicer.NodeURL)
				if err != nil {
					log.Printf("Error sending relay: %v", err)
				} else {
					log.Printf("Relay response received: %s", resp.Response)
				}
			}

			// Try a blockchain RPC call using the simplified method
			blockNumberResp, err := client.BlockchainRPC(ctx, opts.Blockchain, "eth_blockNumber", []interface{}{})
			if err != nil {
				log.Printf("Error executing blockchain RPC: %v", err)
			} else {
				log.Printf("Blockchain RPC successful: Latest block number = %v", blockNumberResp)
			}
		}
	}

	log.Println("Example completed.")
}
