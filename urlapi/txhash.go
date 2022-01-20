package urlapi

import (
	"encoding/hex"
	"time"

	"go.vocdoni.io/api/database/transactions"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
)

type APIMined struct {
	Mined *bool `json:"mined,omitempty"`
	ID    int   `json:"id,omitempty"`
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
		txTime, _ = val.(time.Time)
	} else {
		txTime = time.Now().Add(-15 * time.Second)
	}
	mined := txTime.Add(15 * time.Second).Before(time.Now())
	// TODO make vocclient api request to get tx status
	// If tx has been mined, check dbTransactions map for pending db queries
	if !mined {
		return sendResponse(APIMined{Mined: &mined}, ctx)
	}
	queryTx, err := transactions.GetTx(u.kv, txHash)
	if err != nil {
		return err
	}
	// If no queryTx found, there's no db transaction to execute.
	if queryTx == nil {
		return sendResponse(APIMined{Mined: &mined}, ctx)
	}

	// Else, commit the queryTx to the database
	id, err := queryTx.Commit(&u.db)
	if err != nil {
		return err
	}
	// If query has been committed, delete from kv
	if err = transactions.DeleteTx(u.kv, txHash); err != nil {
		return err
	}
	return sendResponse(APIMined{Mined: &mined, ID: id}, ctx)
}
