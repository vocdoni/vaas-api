package transactions

import (
	"encoding/json"
	"errors"
	"time"

	"go.vocdoni.io/api/database"
)

type SerializableTxType string

const (
	CreateElection     SerializableTxType = "createElection"
	CreateOrganization SerializableTxType = "createOrganization"
	UpdateOrganization SerializableTxType = "updateOrganization"
)

type SerializableTx struct {
	Type         SerializableTxType `json:"type"`
	Body         TxBody             `json:"body"`
	CreationTime time.Time          `json:"creationTime"`
}

func (tx *SerializableTx) Commit(db *database.Database) error {
	return tx.Body.commit(db)
}

func (tx *SerializableTx) UnmarshalJSON(b []byte) error {
	// First unmarshal entire struct
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(b, &objMap)
	if err != nil {
		return err
	}
	err = json.Unmarshal(*objMap["type"], &tx.Type)
	if err != nil {
		return err
	}

	switch tx.Type {
	case CreateElection:
		var body CreateElectionTx
		err = json.Unmarshal(*objMap["body"], &body)
		if err != nil {
			return err
		}
		tx.Body = body
	case CreateOrganization:
		var body CreateOrganizationTx
		err = json.Unmarshal(*objMap["body"], &body)
		if err != nil {
			return err
		}
		tx.Body = body
	case UpdateOrganization:
		var body UpdateOrganizationTx
		err = json.Unmarshal(*objMap["body"], &body)
		if err != nil {
			return err
		}
		tx.Body = body
	default:
		return errors.New("unknown transaction type")
	}
	return nil
}

// SerializableTx is a database transaction that can be serialized and saved for later.
// SerializableTx.Commit() attempts to commit this query to the database, and returns
//  the id of the new database entry, if one exists.
type TxBody interface {
	commit(db *database.Database) error
}
