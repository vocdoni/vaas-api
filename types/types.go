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
	ID           []byte `json:"id" db:"id"`
	IsAuthorized bool   `json:"isAuthorized" db:"is_authorized"`
	IntegratorInfo
}

type IntegratorInfo struct {
	Email string `json:"email,omitempty" db:"email"`
	Name  string `json:"name" db:"name"`
	Size  int    `json:"size" db:"size"`
}
type Entity struct {
	CreatedUpdated
	ID           []byte `json:"id" db:"id"`
	IntegratorID []byte `json:"integratorId" db:"integrator_id"`
	IsAuthorized bool   `json:"isAuthorized" db:"is_authorized"`
	EntityInfo
}

type EntityInfo struct {
	Email string `json:"email,omitempty" db:"email"`
	Name  string `json:"name" db:"name"`
	Size  int    `json:"size" db:"size"`
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
