package transactions

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.vocdoni.io/api/database"
	"go.vocdoni.io/api/types"
)

// CreateElectionTx is the serializable transaction for creating an election
//  commit commits the tx to the sql database
type CreateElectionTx struct {
	TxBody
	IntegratorPrivKey []byte
	EthAddress        []byte
	ElectionID        []byte
	EncryptedMetaKey  []byte
	Title             string
	ProofType         types.ProofType
	StartDate         time.Time
	EndDate           time.Time
	CensusID          uuid.NullUUID
	StartBlock        uint32
	EndBlock          uint32
	Confidential      bool
	HiddenResults     bool
}

func (tx CreateElectionTx) commit(db database.Database) error {
	_, err := db.CreateElection(tx.IntegratorPrivKey,
		tx.EthAddress, tx.ElectionID, tx.EncryptedMetaKey,
		tx.Title, string(tx.ProofType), tx.StartDate, tx.EndDate, tx.CensusID, int(tx.StartBlock),
		int(tx.EndBlock), tx.Confidential, tx.HiddenResults)
	if err != nil {
		return fmt.Errorf("could not create election: %w", err)
	}
	return nil
}
