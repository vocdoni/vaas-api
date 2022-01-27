package urlapi

import (
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

	txTime, err := u.kv.GetTxTime(txHash)
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
	u.kv.Mtx.RLock()
	defer u.kv.Mtx.RUnlock()

	// ONLY if the tx has been mined, try to get the "queryTx" from the map/kv
	queryTx, err := u.kv.GetTx(txHash)
	if err != nil {
		return err
	}

	// If queryTx exists on the kv, return false. The query still needs to be committed to the db
	if queryTx != nil {
		mined = false
	}
	return sendResponse(APIMined{Mined: &mined}, ctx)
}
