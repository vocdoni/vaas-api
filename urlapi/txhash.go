package urlapi

import (
	"time"

	"github.com/google/uuid"
)

type txMap map[string]dbQuery

type dbQuery struct {
	createOrganization *createOrganizationQuery
	updateOrganization *updateOrganizationQuery
	createElection     *createElectionQuery
}

type createOrganizationQuery struct {
	integratorPrivKey []byte
	ethAddress        []byte
	ethPrivKeyCipher  []byte
	planID            uuid.NullUUID
	publicApiQuota    int
	publicApiToken    string
	headerUri         string
	avatarUri         string
}

type updateOrganizationQuery struct {
	integratorPrivKey []byte
	ethAddress        []byte
	headerUri         string
	avatarUri         string
}

type createElectionQuery struct {
	integratorPrivKey []byte
	ethAddress        []byte
	electionID        []byte
	title             string
	startDate         time.Time
	endDate           time.Time
	censusID          uuid.NullUUID
	startBlock        int
	endBlock          int
	confidential      bool
	hiddenResults     bool
}
