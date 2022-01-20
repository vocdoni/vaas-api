package transactions

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.vocdoni.io/api/database"
)

type CreateElectionTx struct {
	TxBody
	IntegratorPrivKey []byte
	EthAddress        []byte
	ElectionID        []byte
	EncryptedMetaKey  []byte
	Title             string
	StartDate         time.Time
	EndDate           time.Time
	CensusID          uuid.NullUUID
	StartBlock        int
	EndBlock          int
	Confidential      bool
	HiddenResults     bool
}

func (tx CreateElectionTx) commit(db *database.Database) (int, error) {
	id, err := (*db).CreateElection(tx.IntegratorPrivKey,
		tx.EthAddress, tx.ElectionID, tx.EncryptedMetaKey,
		tx.Title, tx.StartDate, tx.EndDate, tx.CensusID, tx.StartBlock, tx.EndBlock,
		tx.Confidential, tx.HiddenResults)
	if err != nil {
		return 0, fmt.Errorf("could not create election: %w", err)
	}
	return id, nil
}
