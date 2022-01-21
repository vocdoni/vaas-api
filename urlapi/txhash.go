package urlapi

import (
	"fmt"
	"time"

	"go.vocdoni.io/api/database/transactions"
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
	txTime, err := transactions.GetTxTime(u.kv, txHash)
	if err != nil {
		return fmt.Errorf("transaction %x not found: %w", txHash, err)
	}
	if txTime == nil {
		return fmt.Errorf("transaction %x has no record", txHash)
	}
	mined := txTime.Add(15 * time.Second).Before(time.Now())
	// TODO make vocclient api request to get tx status
	// If tx has been mined, check dbTransactions map for pending db queries
	if !mined {
		return sendResponse(APIMined{Mined: &mined}, ctx)
	}

	// Lock KvMutex so we don't get a tx as it's deleted
	transactions.KvMutex.Lock()
	defer transactions.KvMutex.Unlock()

	queryTx, err := transactions.GetTx(u.kv, txHash)
	if err != nil {
		return err
	}
	// If no queryTx found, there's no db transaction to execute.
	if queryTx == nil {
		return sendResponse(APIMined{Mined: &mined}, ctx)
	}

	// Else, commit the queryTx to the database
	if err = queryTx.Commit(&u.db); err != nil {
		return err
	}
	// If query has been committed, delete from kv
	if err = transactions.DeleteTx(u.kv, txHash); err != nil {
		return err
	}
	return sendResponse(APIMined{Mined: &mined}, ctx)
}
