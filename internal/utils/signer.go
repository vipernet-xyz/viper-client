package utils

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/sha3"
)

// Signer provides cryptographic signing operations
type Signer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	address    string
}

// NewRandomSigner creates a new signer with randomly generated keys
func NewRandomSigner() (*Signer, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keys: %w", err)
	}

	// Generate address from public key (first 20 bytes of SHA-256 of public key)
	h := sha3.New256()
	h.Write(publicKey)
	address := hex.EncodeToString(h.Sum(nil)[:20])

	return &Signer{
		privateKey: privateKey,
		publicKey:  publicKey,
		address:    address,
	}, nil
}

// NewSignerFromPrivateKey creates a signer from a hex-encoded private key or seed
func NewSignerFromPrivateKey(privateKeyHex string) (*Signer, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key hex: %w", err)
	}

	var privateKey ed25519.PrivateKey

	// If we receive just the seed (32 bytes), convert it to a full private key
	if len(privateKeyBytes) == 32 {
		// Generate the full key from the seed
		privateKey = ed25519.NewKeyFromSeed(privateKeyBytes)
	} else if len(privateKeyBytes) == ed25519.PrivateKeySize {
		// Use the full key as is
		privateKey = ed25519.PrivateKey(privateKeyBytes)
	} else {
		return nil, fmt.Errorf("invalid private key size, expected 32 or %d bytes", ed25519.PrivateKeySize)
	}

	publicKey := privateKey.Public().(ed25519.PublicKey)

	// Generate address from public key (first 20 bytes of SHA-256 of public key)
	h := sha3.New256()
	h.Write(publicKey)
	address := hex.EncodeToString(h.Sum(nil)[:20])

	return &Signer{
		privateKey: privateKey,
		publicKey:  publicKey,
		address:    address,
	}, nil
}

// Sign signs the given message with the private key
func (s *Signer) Sign(message []byte) (string, error) {
	signature := ed25519.Sign(s.privateKey, message)
	return hex.EncodeToString(signature), nil
}

// GetAddress returns the address associated with this signer
func (s *Signer) GetAddress() string {
	return s.address
}

// GetPublicKey returns the hex-encoded public key
func (s *Signer) GetPublicKey() string {
	return hex.EncodeToString(s.publicKey)
}

// GetPrivateKey returns the hex-encoded private key
func (s *Signer) GetPrivateKey() string {
	return hex.EncodeToString(s.privateKey)
}
