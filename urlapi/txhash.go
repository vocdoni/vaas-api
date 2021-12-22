package urlapi

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
)

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

type APIMined struct {
	Mined bool `json:"mined,omitempty"`
}

// GET https://server/v1/priv/transactions/<transactionHash>
func (u *URLAPI) getTxStatusHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	txHash, err := util.GetBytesID(ctx, "transactionHash")
	if err != nil {
		return sendResponse(APIMined{Mined: false}, ctx)
	}
	val, ok := u.txWaitMap.Load(hex.EncodeToString(txHash))
	var txTime time.Time
	if ok {
		txTime, ok = val.(time.Time)
	}
	mined := ok && txTime.Add(15*time.Second).Before(time.Now())
	// TODO make vocclient api request to get tx status
	// If tx has been mined, check dbTransactions map for pending db queries
	if !mined {
		return sendResponse(APIMined{Mined: mined}, ctx)
	}
	tx, ok := u.dbTransactions.LoadAndDelete(hex.EncodeToString(txHash))
	// If transaction not in map, it is a transaction
	//  not associated with a db query (setProcessStatus)
	if !ok {
		return sendResponse(APIMined{Mined: mined}, ctx)
	}
	switch queryTx := tx.(type) {
	// Make the db request depending on query type
	case createOrganizationQuery:
		if _, err = u.db.CreateOrganization(queryTx.integratorPrivKey, queryTx.ethAddress,
			queryTx.ethPrivKeyCipher, queryTx.planID, queryTx.publicApiQuota,
			queryTx.publicApiToken, queryTx.headerUri, queryTx.avatarUri); err != nil {
			return fmt.Errorf("could not create organization: %w", err)
		}
	case updateOrganizationQuery:
		if _, err = u.db.UpdateOrganization(queryTx.integratorPrivKey, queryTx.ethAddress,
			queryTx.headerUri, queryTx.avatarUri); err != nil {
			return fmt.Errorf("could not update organization: %w", err)
		}
	case createElectionQuery:
		if _, err = u.db.CreateElection(queryTx.integratorPrivKey, queryTx.ethAddress,
			queryTx.electionID, queryTx.title, queryTx.startDate, queryTx.endDate,
			queryTx.censusID, queryTx.startBlock, queryTx.endBlock, queryTx.confidential,
			queryTx.hiddenResults); err != nil {
			return fmt.Errorf("could not create election: %w", err)
		}
	}
	return sendResponse(APIMined{Mined: mined}, ctx)
}
