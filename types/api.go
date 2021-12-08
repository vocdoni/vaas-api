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
	Avatar        string         `json:"avatar"`
	CspPubKey     types.HexBytes `json:"cspPubKey"`
	CspUrlPrefix  string         `json:"cspUrlPrefix"`
	Description   string         `json:"description"`
	Header        string         `json:"header"`
	Name          string         `json:"name"`
	Email         string         `json:"email"`
	ID            int            `json:"id"`
	Title         string         `json:"title"`
	StreamURI     string         `json:"streamUri"`
	StartDate     string         `json:"startDate"`
	EndDate       string         `json:"endDate"`
	Questions     []Question     `json:"questions"`
	Confidential  bool           `json:"confidential"`
	HiddenResults bool           `json:"hiddenResults"`
	Census        int            `json:"census"`
}

// APIResponse contains all of the possible response fields.
// Fields must be in alphabetical order
// Those fields with valid zero-values (such as bool) must be pointers
type APIResponse struct {
	APIKey       string         `json:"apiKey,omitempty"`
	Avatar       string         `json:"avatar,omitempty"`
	CensusID     int            `json:"census_id,omitempty"`
	ContentURI   string         `json:"contentUri,omitempty"`
	CspPubKey    types.HexBytes `json:"cspPubKey,omitempty"`
	CspUrlPrefix string         `json:"cspUrlPrefix,omitempty"`
	Description  string         `json:"description,omitempty"`
	EndBlock     []byte         `json:"end_block,omitempty"`
	EntityID     types.HexBytes `json:"entityId,omitempty"`
	Header       string         `json:"header,omitempty"`
	ID           int            `json:"id,omitempty"`
	Message      string         `json:"message,omitempty"`
	Name         string         `json:"name,omitempty"`
	Ok           bool           `json:"ok,omitempty"`
	ProcessID    types.HexBytes `json:"processId,omitempty"`
}

// APIProcess is the response struct for a getProcess request
type APIProcess struct {
	Description        string         `json:"description,omitempty"`
	EntityID           types.HexBytes `json:"entityId,omitempty"`
	Header             string         `json:"header,omitempty"`
	Ok                 bool           `json:"ok,omitempty"`
	ProcessID          types.HexBytes `json:"processId,omitempty"`
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

type ProcessSummary struct {
	ProcessID   types.HexBytes `json:"processId,omitempty"`
	Title       string
	Description string
	Header      string
	Status      string    `json:"status,omitempty"`
	StartDate   time.Time `json:"startDate,omitempty"`
	EndDate     time.Time `json:"endDate,omitempty"`
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
