package types

import (
	"fmt"
	"time"

	"go.vocdoni.io/dvote/api"
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
	APIKey           string               `json:"apiKey,omitempty"`
	APIToken         string               `json:"apiToken,omitempty"`
	Avatar           string               `json:"avatar,omitempty"`
	CensusID         int                  `json:"censusId,omitempty"`
	ContentURI       string               `json:"contentUri,omitempty"`
	CspPubKey        types.HexBytes       `json:"cspPubKey,omitempty"`
	CspUrlPrefix     string               `json:"cspUrlPrefix,omitempty"`
	Description      string               `json:"description,omitempty"`
	ElectionID       types.HexBytes       `json:"electionId,omitempty"`
	ExplorerUrl      string               `json:"explorerUrl,omitempty"`
	Header           string               `json:"header,omitempty"`
	ID               int                  `json:"id,omitempty"`
	Message          string               `json:"message,omitempty"`
	Name             string               `json:"name,omitempty"`
	Nullifier        string               `json:"nullifier,omitempty"`
	OrganizationID   types.HexBytes       `json:"organizationId,omitempty"`
	PrivateProcesses []APIElectionSummary `json:"private,omitempty"`
	ProcessID        types.HexBytes       `json:"processId,omitempty"`
	PublicProcesses  []APIElectionSummary `json:"public,omitempty"`
	Registered       bool                 `json:"registered,omitempty"`
}

// APIElectionInfo is the response struct for a getElection request
//  including all election information
type APIElectionInfo struct {
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
	Type      string    `json:"type,omitempty"`
	VoteCount uint32    `json:"voteCount,omitempty"`
}

// APIElectionSummary is the struct for returning election info from the database
type APIElectionSummary struct {
	OrgEthAddress   types.HexBytes `json:"orgEthAddress,omitempty" db:"organizationEthAddress"`
	ElectionID      types.HexBytes `json:"electionId,omitempty" db:"processId"`
	Title           string         `json:"title,omitempty" db:"title"`
	CensusID        string         `json:"censusId,omitempty" db:"censusId"`
	StartDate       time.Time      `json:"startDate,omitempty" db:"startDate"`
	EndDate         time.Time      `json:"endDate,omitempty" db:"endDate"`
	Confidential    bool           `json:"confidential,omitempty" db:"confidential"`
	HiddenResults   bool           `json:"hiddenResults,omitempty" db:"hiddenResults"`
	MetadataPrivKey []byte         `json:"metadataPrivKey,omitempty" db:"metadataPrivKey"`
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
	StreamURI string `json:"streamUri,omitempty"`
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
	NewsFeed    LanguageString `json:"newsFeed,omitempty"`
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

// SetError sets the MetaResponse's Message to a string
// representation of v. Usually, v's type will be error or string.
func (r *APIResponse) SetError(v interface{}) {
	r.Message = fmt.Sprintf("%s", v)
}
