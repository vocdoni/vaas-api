package urlapi

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
)

type APIMined struct {
	Mined *bool `json:"mined,omitempty"`
}

// GET https://server/v1/priv/transactions/<transactionHash>
func (u *URLAPI) getTxStatusHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	txHash, err := util.GetBytesID(ctx, "transactionHash")
	if err != nil {
		mined := false
		return sendResponse(APIMined{Mined: &mined}, ctx)
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
		return sendResponse(APIMined{Mined: &mined}, ctx)
	}
	tx, ok := u.dbTransactions.LoadAndDelete(hex.EncodeToString(txHash))
	// If transaction not in map, it is a transaction
	//  not associated with a db query (setProcessStatus)
	if !ok {
		return sendResponse(APIMined{Mined: &mined}, ctx)
	}

	// Make the db request
	switch queryTx := tx.(type) {
	case *sql.Tx:
		if err = queryTx.Commit(); err != nil {
			return fmt.Errorf("could not execute database transaction: %w", err)
		}
	default:
		return fmt.Errorf("could not execute database transaction: wrong query type")
	}
	return sendResponse(APIMined{Mined: &mined}, ctx)
}
