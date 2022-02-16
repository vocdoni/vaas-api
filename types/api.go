package types

import (
	"fmt"
	"time"

	"go.vocdoni.io/dvote/api"
	"go.vocdoni.io/dvote/types"
)

const (
	PROOF_TYPE_ECDSA ProofType = "ecdsa"
	PROOF_TYPE_BLIND ProofType = "blind"
)

type ProofType string

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
	APIKey         string                `json:"apiKey,omitempty"`
	APIToken       string                `json:"apiToken,omitempty"`
	Avatar         string                `json:"avatar,omitempty"`
	CensusID       int                   `json:"censusId,omitempty"`
	ContentURI     string                `json:"contentUri,omitempty"`
	CspPubKey      types.HexBytes        `json:"cspPubKey,omitempty"`
	CspUrlPrefix   string                `json:"cspUrlPrefix,omitempty"`
	Description    string                `json:"description,omitempty"`
	ElectionID     types.HexBytes        `json:"electionId,omitempty"`
	ExplorerUrl    string                `json:"explorerUrl,omitempty"`
	Header         string                `json:"header,omitempty"`
	ID             int                   `json:"id,omitempty"`
	Message        string                `json:"message,omitempty"`
	Name           string                `json:"name,omitempty"`
	Nullifier      string                `json:"nullifier,omitempty"`
	OrganizationID types.HexBytes        `json:"organizationId,omitempty"`
	Organizations  []APIOrganizationInfo `json:"organizations,omitempty"`
	Registered     *bool                 `json:"registered,omitempty"`
	TxHash         types.HexBytes        `json:"txHash,omitempty"`
}

// APIOrganizationInfo is the organization summary for the getOrganizationList call
type APIOrganizationInfo struct {
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
	ID          string    `json:"id"`
	APIToken    string    `json:"api_token,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Avatar      string    `json:"avatar,omitempty"`
	Header      string    `json:"header,omitempty"`
}

// APIElectionInfo is the response struct for a getElection request
//  including all election information
type APIElectionInfo struct {
	ChainID            string         `json:"chainId,omitempty"`
	Description        string         `json:"description,omitempty"`
	OrganizationID     types.HexBytes `json:"organizationId,omitempty"`
	Header             string         `json:"header,omitempty"`
	ElectionID         types.HexBytes `json:"electionId,omitempty"`
	EncryptionPubKeys  []api.Key      `json:"encryptionPubKeys,omitempty"`
	Questions          []Question     `json:"questions,omitempty"`
	Results            []Result       `json:"results,omitempty"`
	ResultsAggregation string         `json:"aggregation,omitempty"`
	ResultsDisplay     string         `json:"display,omitempty"`
	// Estimated start/end dates
	EndDate   time.Time `json:"endDate,omitempty"`
	StartDate time.Time `json:"startDate,omitempty"`
	Status    string    `json:"status,omitempty"`
	StreamURI string    `json:"streamUri,omitempty"`
	Title     string    `json:"title,omitempty"`
	ProofType ProofType `json:"proofType,omitempty"`
	Type      string    `json:"type,omitempty"`
	VoteCount uint32    `json:"voteCount,omitempty"`
}

// APIElectionSummary is the struct for returning election info from the database
type APIElectionSummary struct {
	CensusID        string         `json:"censusId,omitempty"`
	Confidential    *bool          `json:"confidential,omitempty"`
	ElectionID      types.HexBytes `json:"electionId,omitempty"`
	EndDate         time.Time      `json:"endDate,omitempty"`
	HiddenResults   *bool          `json:"hiddenResults,omitempty"`
	MetadataPrivKey []byte         `json:"metadataPrivKey,omitempty"`
	OrgEthAddress   types.HexBytes `json:"orgEthAddress,omitempty"`
	StartDate       time.Time      `json:"startDate,omitempty"`
	Status          string         `json:"status,omitempty"`
	Title           string         `json:"title,omitempty"`
	ProofType       ProofType      `json:"proofType,omitempty"`
}

// ProcessMetadata contains the process metadata fields as stored on ipfs
type ProcessMetadata struct {
	Description LanguageString        `json:"description,omitempty"`
	Media       ProcessMedia          `json:"media,omitempty"`
	Meta        interface{}           `json:"meta,omitempty"`
	Questions   []QuestionMeta        `json:"questions,omitempty"`
	Results     ProcessResultsDetails `json:"results,omitempty"`
	Title       LanguageString        `json:"title,omitempty"`
	Version     string                `json:"version,omitempty"`
}

// Result is a single election result for the API response
type Result struct {
	Title []string `json:"title"`
	Value []string `json:"value"`
}

// Question is a single election question for the API response
type Question struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Choices     []Choice `json:"choices"`
}

// Choice is a sigle question choice for the API response
type Choice struct {
	Title string `json:"title"`
	Value uint32 `json:"value"`
}

// LanguageString is a wrapper for multi-language strings, specified in metadata.
//  example {"default": "hello", "en": "hello", "es": "hola"}
type LanguageString map[string]string

// ProcessMedia holds the process metadata's header and streamURI
type ProcessMedia struct {
	Header    string `json:"header,omitempty"`
	StreamURI string `json:"streamUri,omitempty"`
}

// ProcessResultsDetails describes how a process results should be displayed and aggregated
type ProcessResultsDetails struct {
	Aggregation string `json:"aggregation,omitempty"`
	Display     string `json:"display,omitempty"`
}

// QuestionMeta contains metadata for one single question of a process
type QuestionMeta struct {
	Choices     []ChoiceMetadata `json:"choices"`
	Description LanguageString   `json:"description"`
	Title       LanguageString   `json:"title"`
}

// ChoiceMetadata contains metadata for one choice of a question
type ChoiceMetadata struct {
	Title LanguageString `json:"title,omitempty"`
	Value uint32         `json:"value,omitempty"`
}

// EntityMetadata is the metadata for an organization
type EntityMetadata struct {
	Version     string         `json:"version,omitempty"`
	Languages   []string       `json:"languages,omitempty"`
	Name        LanguageString `json:"name,omitempty"`
	Description LanguageString `json:"description,omitempty"`
	NewsFeed    LanguageString `json:"newsFeed,omitempty"`
	Media       EntityMedia    `json:"media,omitempty"`
	Meta        interface{}    `json:"meta,omitempty"`
	Actions     interface{}    `json:"actions,omitempty"`
}

// EntityMedia stores the avatar, header, and logo for an entity metadata
type EntityMedia struct {
	Avatar string `json:"avatar,omitempty"`
	Header string `json:"header,omitempty"`
	Logo   string `json:"logo,omitempty"`
}

// VochainResults is the results of a single process, as returned by the vochain
type VochainResults struct {
	Height  uint32     `json:"height,omitempty"`
	Results [][]string `json:"results,omitempty"`
	State   string     `json:"state,omitempty"`
	Type    string     `json:"type,omitempty"`
}

// RawFile provides a json struct wrapper to a raw bytes payload, used for storing
//  encrypted metadata on ipfs. Version is "1.0"
type RawFile struct {
	Payload []byte `json:"payload,omitempty"`
	Version string `json:"version,omitempty"`
}

// SetError sets the MetaResponse's Message to a string
// representation of v. Usually, v's type will be error or string.
func (r *APIResponse) SetError(v interface{}) {
	r.Message = fmt.Sprintf("%s", v)
}
