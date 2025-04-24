package models

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/sha3"
)

// RelayPayload represents the data payload of a relay request
type RelayPayload struct {
	Data    string            `json:"data"`
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
}

// RelayMeta contains metadata for a relay request
type RelayMeta struct {
	BlockHeight  int64 `json:"block_height"`
	Subscription bool  `json:"subscription"`
	AI           bool  `json:"ai"`
}

// ViperAAT is the Application Authentication Token
type ViperAAT struct {
	Version         string `json:"version"`
	RequestorPubKey string `json:"requestor_pub_key"`
	ClientPubKey    string `json:"client_pub_key"`
	Signature       string `json:"signature"`
}

// "ID" - Returns the merkleHash of the AAT bytes
func (a ViperAAT) Hash() []byte {
	return Hash(a.Bytes())
}

// "Bytes" - Returns the bytes representation of the AAT
func (a ViperAAT) Bytes() []byte {
	// using standard json bz
	b, err := json.Marshal(ViperAAT{
		Signature:       "",
		RequestorPubKey: a.RequestorPubKey,
		ClientPubKey:    a.ClientPubKey,
		Version:         a.Version,
	})
	if err != nil {
		log.Fatal(fmt.Sprintf("an error occured hashing the aat:\n%v", err))
	}
	return b
}

func Hash(b []byte) []byte {
	hasher := sha3.New256()
	hasher.Write(b) //nolint:golint,errcheck
	return hasher.Sum(nil)
}

// RelayProof contains proof information for a relay request
type RelayProof struct {
	RequestHash        string    `json:"request_hash"`
	Entropy            int64     `json:"entropy"`
	SessionBlockHeight int64     `json:"session_block_height"`
	ServicerPubKey     string    `json:"servicer_pub_key"`
	Blockchain         string    `json:"blockchain"`
	Token              *ViperAAT `json:"aat"`
	Signature          string    `json:"signature"`
	GeoZone            string    `json:"zone"`
	NumServicers       int64     `json:"num_servicers"`
	RelayType          int64     `json:"relay_type"`
	Weight             int64     `json:"weight"`
}

// Relay represents a complete relay request
type Relay struct {
	Payload RelayPayload `json:"payload"`
	Meta    RelayMeta    `json:"meta"`
	Proof   RelayProof   `json:"proof"`
}

// RelayResponse represents the response from a relay
type RelayResponse struct {
	Signature string     `json:"signature"`
	Response  string     `json:"response"`
	Proof     RelayProof `json:"proof"`
}

// Servicer represents a node in the viper network
type Servicer struct {
	Address       string            `json:"address"`
	Chains        []string          `json:"chains"`
	GeoZone       []string          `json:"geo_zone"`
	Jailed        bool              `json:"jailed"`
	Paused        bool              `json:"paused"`
	PublicKey     string            `json:"public_key"`
	NodeURL       string            `json:"node_url"`
	Status        int               `json:"status"`
	Tokens        string            `json:"tokens"`
	UnstakingTime time.Time         `json:"unstaking_time"`
	OutputAddress string            `json:"output_address"`
	Delegators    map[string]uint32 `json:"delegators"`
}

// Session contains information about a dispatch session
type Session struct {
	Header             SessionHeader `json:"header"`
	Key                string        `json:"key"`
	FishermenTriggered bool          `json:"fishermen_triggered"`
	Servicers          []Servicer    `json:"servicers"`
	Fishermen          []Servicer    `json:"fishermen"`
}

// Header contains session header information
type SessionHeader struct {
	RequestorPublicKey string `json:"requestor_public_key"`
	Chain              string `json:"chain"`
	GeoZone            string `json:"zone"`
	NumServicers       int64  `json:"num_servicers"`
	SessionHeight      int64  `json:"session_height"`
}

// DispatchResponse represents the response from a dispatch request
type DispatchResponse struct {
	Session     *Session `json:"session"`
	BlockHeight int      `json:"block_height"`
}

type RelayProofForSignature struct {
	Entropy            int64  `json:"entropy"`
	SessionBlockHeight int64  `json:"session_block_height"`
	ServicerPubKey     string `json:"servicer_pub_key"`
	Blockchain         string `json:"blockchain"`
	Signature          string `json:"signature"`
	Token              string `json:"token"`
	RequestHash        string `json:"request_hash"`
	GeoZone            string `json:"zone"`
	NumServicers       int64  `json:"num_servicers"`
	RelayType          int64  `json:"relay_type"`
	Weight             int64  `json:"weight"`
}
