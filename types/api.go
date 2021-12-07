package types

import (
	"fmt"

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
	Type               string         `json:"type,omitempty"`
	Title              string         `json:"title,omitempty"`
	Description        string         `json:"description,omitempty"`
	Header             string         `json:"header,omitempty"`
	StreamURI          string         `json:"stream_uri,omitempty"`
	Questions          []Question     `json:"questions,omitempty"`
	Status             string         `json:"status,omitempty"`
	VoteCount          uint32         `json:"vote_count,omitempty"`
	Results            []Result       `json:"results,omitempty"`
	EntityID           types.HexBytes `json:"entityId,omitempty"`
	Ok                 bool           `json:"ok,omitempty"`
	ProcessID          types.HexBytes `json:"processId,omitempty"`
	StartBlock         string         `json:"start_block,omitempty"`
	EndBlock           string         `json:"end_block,omitempty"`
	ResultsAggregation string
	ResultsDisplay     string
}

type ProcessMetadata struct {
	Version     string                `json:"version,omitempty"`
	Title       LanguageString        `json:"title,omitempty"`
	Description LanguageString        `json:"description,omitempty"`
	Media       ProcessMedia          `json:"media,omitempty"`
	Meta        interface{}           `json:"meta,omitempty"`
	Results     ProcessResultsDetails `json:"results,omitempty"`
	Questions   []QuestionMeta        `json:"questions,omitempty"`
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
	Title       LanguageString `json:"title"`
	Description LanguageString `json:"description"`
	Choices     []Choice       `json:"choices"`
}

type Choice struct {
	Title LanguageString
	Value uint32
}

type EntityMetadata struct {
	Avatar      string `json:"avatar"`
	Description string `json:"description"`
	Header      string `json:"header"`
	Name        string `json:"name"`
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
