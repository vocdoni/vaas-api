package urlapi

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
)

type APIMined struct {
	Mined *bool `json:"mined,omitempty"`
	Id    int   `json:"id,omitempty"`
}

type queryTx struct {
	tx *sqlx.Tx
	id int
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

	var id int
	// Make the db request
	switch queryTx := tx.(type) {
	case *queryTx:
		if err = queryTx.tx.Commit(); err != nil {
			err2 := queryTx.tx.Rollback()
			if err2 != nil {
				log.Warnf(err2.Error())
			}
			id = queryTx.id
			return fmt.Errorf("could not execute database transaction: %w", err)
		}
	default:
		return fmt.Errorf("could not execute database transaction: wrong query type")
	}
	return sendResponse(APIMined{Mined: &mined, Id: id}, ctx)
}
