package types

import (
	"fmt"
	"time"

	"go.vocdoni.io/dvote/types"
)

// APIRequest contains all of the possible request fields.
// Fields must be in alphabetical order
// Those fields with valid zero-values (such as bool) must be pointers
type APIRequest struct {
	Avatar        string     `json:"avatar"`
	Census        string     `json:"census"`
	Confidential  bool       `json:"confidential"`
	CspPubKey     string     `json:"cspPubKey"`
	CspUrlPrefix  string     `json:"cspUrlPrefix"`
	Description   string     `json:"description"`
	Email         string     `json:"email"`
	EndDate       string     `json:"endDate"`
	Header        string     `json:"header"`
	HiddenResults bool       `json:"hiddenResults"`
	Vote          string     `json:"vote"`
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	Questions     []Question `json:"questions"`
	StartDate     string     `json:"startDate"`
	StreamURI     string     `json:"streamUri"`
	Title         string     `json:"title"`
}

// APIResponse contains all of the possible response fields.
// Fields must be in alphabetical order
// Those fields with valid zero-values (such as bool) must be pointers
type APIResponse struct {
	APIKey           string         `json:"apiKey,omitempty"`
	APIToken         string         `json:"apiToken,omitempty"`
	Avatar           string         `json:"avatar,omitempty"`
	CensusID         int            `json:"census_id,omitempty"`
	ContentURI       string         `json:"contentUri,omitempty"`
	CspPubKey        types.HexBytes `json:"cspPubKey,omitempty"`
	CspUrlPrefix     string         `json:"cspUrlPrefix,omitempty"`
	Description      string         `json:"description,omitempty"`
	ElectionID       types.HexBytes `json:"electionId,omitempty"`
	EndBlock         []byte         `json:"end_block,omitempty"`
	Header           string         `json:"header,omitempty"`
	ID               int            `json:"id,omitempty"`
	Message          string         `json:"message,omitempty"`
	Name             string         `json:"name,omitempty"`
	Nullifier        string         `json:"nullifier,omitempty"`
	Ok               bool           `json:"ok,omitempty"`
	OrganizationID   types.HexBytes `json:"organizationId,omitempty"`
	PrivateProcesses []APIElection  `json:"private,omitempty"`
	PublicProcesses  []APIElection  `json:"public,omitempty"`
}

// APIProcess is the response struct for a getProcess request
type APIProcess struct {
	Description        string         `json:"description,omitempty"`
	OrganizationID     types.HexBytes `json:"organizationId,omitempty"`
	Header             string         `json:"header,omitempty"`
	Ok                 bool           `json:"ok,omitempty"`
	ElectionID         types.HexBytes `json:"electionId,omitempty"`
	Questions          []Question     `json:"questions,omitempty"`
	Results            []Result       `json:"results,omitempty"`
	ResultsAggregation string         `json:"results_aggregation,omitempty"`
	ResultsDisplay     string         `json:"results_display,omitempty"`
	// Estimated start/end dates
	EndDate   time.Time `json:"end_date,omitempty"`
	StartDate time.Time `json:"start_date,omitempty"`
	// Start/end blocks are source of truth
	EndBlock   string `json:"end_block,omitempty"`
	StartBlock string `json:"start_block,omitempty"`
	Status     string `json:"status,omitempty"`
	StreamURI  string `json:"stream_uri,omitempty"`
	Title      string `json:"title,omitempty"`
	Type       string `json:"type,omitempty"`
	VoteCount  uint32 `json:"vote_count,omitempty"`
}

type APIElection struct {
	OrgEthAddress   types.HexBytes `json:"orgEthAddress,omitempty" db:"organization_eth_address"`
	ProcessID       types.HexBytes `json:"processId,omitempty" db:"process_id"`
	Title           string         `json:"title,omitempty" db:"title"`
	CensusID        string         `json:"censusId,omitempty" db:"census_id"`
	StartDate       time.Time      `json:"startDate,omitempty" db:"start_date"`
	EndDate         time.Time      `json:"endDate,omitempty" db:"end_date"`
	StartBlock      int            `json:"startBlock,omitempty" db:"start_block"`
	EndBlock        int            `json:"endBlock,omitempty" db:"end_block"`
	Confidential    bool           `json:"confidential,omitempty" db:"confidential"`
	HiddenResults   bool           `json:"hiddenResults,omitempty" db:"hidden_results"`
	MetadataPrivKey []byte         `json:"metadataPrivKey,omitempty" db:"metadata_priv_key"`
}

type ProcessMetadata struct {
	Description LanguageString        `json:"description,omitempty"`
	Media       ProcessMedia          `json:"media,omitempty"`
	Meta        interface{}           `json:"meta,omitempty"`
	Questions   []QuestionMeta        `json:"questions,omitempty"`
	Results     ProcessResultsDetails `json:"results,omitempty"`
	Title       LanguageString        `json:"title,omitempty"`
	Version     string                `json:"version,omitempty"`
}

type LanguageString map[string]string

type ProcessMedia struct {
	Header    string `json:"header,omitempty"`
	StreamURI string `json:"stream_uri,omitempty"`
}

type ProcessResultsDetails struct {
	Aggregation string `json:"aggregation,omitempty"`
	Display     string `json:"display,omitempty"`
}

type QuestionMeta struct {
	Choices     []Choice       `json:"choices"`
	Description LanguageString `json:"description"`
	Title       LanguageString `json:"title"`
}

type Choice struct {
	Title LanguageString `json:"title,omitempty"`
	Value uint32         `json:"value,omitempty"`
}

type EntityMetadata struct {
	Version     string         `json:"version,omitempty"`
	Languages   []string       `json:"languages,omitempty"`
	Name        LanguageString `json:"name,omitempty"`
	Description LanguageString `json:"description,omitempty"`
	NewsFeed    LanguageString `json:"news_feed,omitempty"`
	Media       EntityMedia    `json:"media,omitempty"`
	Meta        interface{}    `json:"meta,omitempty"`
	Actions     interface{}    `json:"actions,omitempty"`
}

type EntityMedia struct {
	Avatar string `json:"avatar,omitempty"`
	Header string `json:"header,omitempty"`
	Logo   string `json:"logo,omitempty"`
}

type VochainResults struct {
	Height  uint32     `json:"height,omitempty"`
	Results [][]string `json:"results,omitempty"`
	State   string     `json:"state,omitempty"`
	Type    string     `json:"type,omitempty"`
}

// SetError sets the MetaResponse's Ok field to false, and Message to a string
// representation of v. Usually, v's type will be error or string.
func (r *APIResponse) SetError(v interface{}) {
	r.Ok = false
	r.Message = fmt.Sprintf("%s", v)
}
