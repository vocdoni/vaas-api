package types

import (
	"fmt"

	"go.vocdoni.io/dvote/types"
)

// APIRequest contains all of the possible request fields.
// Fields must be in alphabetical order
// Those fields with valid zero-values (such as bool) must be pointers
type APIRequest struct {
	Avatar       string         `json:"avatar"`
	CspPubKey    types.HexBytes `json:"cspPubKey"`
	CspUrlPrefix string         `json:"cspUrlPrefix"`
	Description  string         `json:"description"`
	Header       string         `json:"header"`
	Name         string         `json:"name"`
	ID           int            `json:"id"`
}

// APIResponse contains all of the possible response fields.
// Fields must be in alphabetical order
// Those fields with valid zero-values (such as bool) must be pointers
type APIResponse struct {
	Avatar       string         `json:"avatar"`
	APIKey       string         `json:"apiKey"`
	CspPubKey    types.HexBytes `json:"cspPubKey"`
	Header       string         `json:"header"`
	CspUrlPrefix string         `json:"cspUrlPrefix"`
	ContentURI   string         `json:"contentUri"`
	EntityID     types.HexBytes `json:"entityId"`
	ID           int32          `json:"id"`
	Message      string         `json:"message,omitempty"`
	Name         string         `json:"name"`
	Ok           bool           `json:"ok"`
}

type EntityMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Header      string `json:"header"`
	Avatar      string `json:"avatar"`
}

// SetError sets the MetaResponse's Ok field to false, and Message to a string
// representation of v. Usually, v's type will be error or string.
func (r *APIResponse) SetError(v interface{}) {
	r.Ok = false
	r.Message = fmt.Sprintf("%s", v)
}
