package types

import (
	"fmt"

	"github.com/google/uuid"
)

// APIRequest contains all of the possible request fields.
// Fields must be in alphabetical order
// Those fields with valid zero-values (such as bool) must be pointers
type APIRequest struct {
	// TODO add necessary request fields for APIv1 definition
}

// APIResponse contains all of the possible response fields.
// Fields must be in alphabetical order
// Those fields with valid zero-values (such as bool) must be pointers
type APIResponse struct {
	// TODO add necessary response fields for APIv1 definition
	Message string `json:"message,omitempty"`
	Ok      bool   `json:"ok"`
}

// SetError sets the MetaResponse's Ok field to false, and Message to a string
// representation of v. Usually, v's type will be error or string.
func (r *APIResponse) SetError(v interface{}) {
	r.Ok = false
	r.Message = fmt.Sprintf("%s", v)
}

type TokenEmail struct {
	Token uuid.UUID `json:"tokens"`
	Email string    `json:"emails"`
}

type Status struct {
	Registered  bool `json:"registered"`
	NeedsUpdate bool `json:"needsUpdate"`
}
