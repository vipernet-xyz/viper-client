package models

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
	Version            string `json:"version"`
	RequestorPublicKey string `json:"requestor_public_key"`
	ClientPublicKey    string `json:"client_public_key"`
	Signature          string `json:"signature"`
}

// RelayProof contains proof information for a relay request
type RelayProof struct {
	RequestHash        string   `json:"request_hash"`
	Entropy            int64    `json:"entropy"`
	SessionBlockHeight int64    `json:"session_block_height"`
	ServicerPubKey     string   `json:"servicer_pub_key"`
	Blockchain         string   `json:"blockchain"`
	Token              ViperAAT `json:"aat"`
	GeoZone            string   `json:"geo_zone"`
	NumServicers       int64    `json:"num_servicers"`
	RelayType          int      `json:"relay_type"` // Regular (1) or Subscription (2)
	Weight             int      `json:"weight"`
	Signature          string   `json:"signature"`
	Address            string   `json:"address"`
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
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
	NodeURL   string `json:"node_url"`
}

// Session contains information about a dispatch session
type Session struct {
	Key       string     `json:"key"`
	Header    Header     `json:"header"`
	Servicers []Servicer `json:"servicers"`
}

// Header contains session header information
type Header struct {
	RequestorPubKey    string `json:"requestor_public_key"`
	Chain              string `json:"chain"`
	GeoZone            string `json:"geo_zone"`
	NumServicers       int64  `json:"num_servicers"`
	SessionBlockHeight int64  `json:"session_block_height"`
}

// DispatchResponse represents the response from a dispatch request
type DispatchResponse struct {
	Session     Session `json:"session"`
	BlockHeight int64   `json:"block_height"`
}
