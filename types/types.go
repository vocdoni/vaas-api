package types

import (
	"time"

	"go.vocdoni.io/dvote/types"
)

type CreatedUpdated struct {
	CreatedAt time.Time `json:"createdAt,omitempty" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" db:"updated_at"`
}

type Integrator struct {
	CreatedUpdated
	ID           int    `json:"id" db:"id"`
	SecretApiKey []byte `json:"secretApiKey" db:"secret_api_key"`
	Name         string `json:"name" db:"name"`
	CspUrlPrefix string `json:"cspUrlPrefix" db:"csp_url_prefix"`
	CspPubKey    []byte `json:"cspPubKey" db:"csp_pub_key"` // CSP compressed eth public key
}
type Organization struct {
	CreatedUpdated
	ID               int    `json:"id" db:"id"`
	IntegratorID     int    `json:"integratorId" db:"integrator_id"`
	IntegratorApiKey int    `json:"integratorApiKey" db:"integrator_api_key"`
	EthAddress       []byte `json:"ethAddress" db:"eth_address"`
	EncryptedPrivKey []byte `json:"encrypedPrivKey" db:"encrypted_priv_key"` // encrypted priv key for metadata
	Name             string `json:"name" db:"name"`
	HeaderURI        string `json:"headerUri" db:"header_uri"`            // cURI
	AvatarURI        string `json:"avatarUri" db:"avatar_uri"`            // cURI
	PublicAPIToken   string `json:"publicApiToken" db:"public_api_token"` // Public API token
	QuotaPlanID      int    `json:"quotaPlanId" db:"quota_plan_id"`       // Billing plan ID
	PublicAPIQuota   int    `json:"publicApiQuota" db:"public_api_quota"`
}

type ErrorMsg struct {
	Error string `json:"error"`
}

type EntitiesMsg struct {
	EntityID  types.HexBytes    `json:"entityID"`
	Processes []*ProcessSummary `json:"processes,omitempty"`
}

type ProcessSummary struct {
	ProcessID types.HexBytes `json:"processId,omitempty"`
	Status    string         `json:"status,omitempty"`
	StartDate time.Time      `json:"startDate,omitempty"`
	EndDate   time.Time      `json:"endDate,omitempty"`
}

type Process struct {
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Header      string     `json:"header"`
	StreamURI   string     `json:"streamUri"`
	Status      string     `json:"status"`
	VoteCount   uint64     `json:"voteCount"`
	Questions   []Question `json:"questions"`
	Results     []Result   `json:"result"`
}

type Result struct {
	Title []string `json:"title"`
	Value []string `json:"value"`
}

type Question struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Choices     []string `json:"choices"`
}

type ListOptions struct {
	Count  int    `json:"count,omitempty"`
	Order  string `json:"order,omitempty"`
	Skip   int    `json:"skip,omitempty"`
	SortBy string `json:"sortBy,omitempty"`
}
