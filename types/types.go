package types

import (
	"time"

	"github.com/google/uuid"
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
	Email        string `json:"email" db:"email"`
	CspUrlPrefix string `json:"cspUrlPrefix" db:"csp_url_prefix"`
	CspPubKey    []byte `json:"cspPubKey" db:"csp_pub_key"` // CSP compressed eth public key
}

type QuotaPlan struct {
	CreatedUpdated
	ID              uuid.UUID `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	MaxCensusSize   int       `json:"maxCensusSize" db:"max_census_size"`
	MaxProcessCount int       `json:"maxProcessCount" db:"max_process_count"`
}
type Organization struct {
	CreatedUpdated
	ID               int           `json:"id" db:"id"`
	IntegratorID     int           `json:"integratorId" db:"integrator_id"`
	IntegratorApiKey []byte        `json:"integratorApiKey" db:"integrator_api_key"`
	EthAddress       []byte        `json:"ethAddress" db:"eth_address"`
	EthPrivKeyCipher []byte        `json:"ethPrivKeyCipher" db:"eth_priv_key_cipher"` // encrypted priv key for metadata
	HeaderURI        string        `json:"headerUri" db:"header_uri"`                 // cURI
	AvatarURI        string        `json:"avatarUri" db:"avatar_uri"`                 // cURI
	PublicAPIToken   string        `json:"publicApiToken" db:"public_api_token"`      // Public API token
	QuotaPlanID      uuid.NullUUID `json:"quotaPlanId" db:"quota_plan_id"`            // Billing plan ID
	PublicAPIQuota   int           `json:"publicApiQuota" db:"public_api_quota"`
}

type Election struct {
	CreatedUpdated
	ID               int           `json:"id,omitempty" db:"id"`
	OrgEthAddress    []byte        `json:"orgEthAddress,omitempty" db:"organization_eth_address"`
	IntegratorApiKey []byte        `json:"integratorApiKey,omitempty" db:"integrator_api_key"`
	ProcessID        []byte        `json:"processId,omitempty" db:"process_id"`
	Title            string        `json:"title,omitempty" db:"title"`
	CensusID         uuid.NullUUID `json:"censusId,omitempty" db:"census_id"`
	StartDate        time.Time     `json:"startDate,omitempty" db:"start_date"`
	EndDate          time.Time     `json:"endDate,omitempty" db:"end_date"`
	StartBlock       int           `json:"startBlock,omitempty" db:"start_block"`
	EndBlock         int           `json:"endBlock,omitempty" db:"end_block"`
	Confidential     bool          `json:"confidential,omitempty" db:"confidential"`
	HiddenResults    bool          `json:"hiddenResults,omitempty" db:"hidden_results"`
	MetadataPrivKey  []byte        `json:"metadataPrivKey,omitempty" db:"metadata_priv_key"`
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
