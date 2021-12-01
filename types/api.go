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
	Plan         string         `json:"plan"`
	ID           int            `json:"id"`
}

// APIResponse contains all of the possible response fields.
// Fields must be in alphabetical order
// Those fields with valid zero-values (such as bool) must be pointers
type APIResponse struct {
	APIKey  string `json:"apiKey"`
	ID      int    `json:"id"`
	Message string `json:"message,omitempty"`
	Ok      bool   `json:"ok"`
}

// SetError sets the MetaResponse's Ok field to false, and Message to a string
// representation of v. Usually, v's type will be error or string.
func (r *APIResponse) SetError(v interface{}) {
	r.Ok = false
	r.Message = fmt.Sprintf("%s", v)
}
