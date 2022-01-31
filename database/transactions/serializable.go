package transactions

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.vocdoni.io/api/database"
)

// SerializableTxType describes the type of transaction to serialize
type SerializableTxType string

const (
	// Transaction type to create an election in the database
	CreateElection SerializableTxType = "createElection"
	// Transaction type to create an organization in the database
	CreateOrganization SerializableTxType = "createOrganization"
	// Transaction type to update an organization in the database
	UpdateOrganization SerializableTxType = "updateOrganization"
)

// SerializableTx is a database transaction that can be serialized and stored for use later.
type SerializableTx struct {
	Type         SerializableTxType `json:"type"`
	Body         TxBody             `json:"body"`
	CreationTime time.Time          `json:"creationTime"`
}

// Commit commits the serializableTx to the database, using the txBody implementer's commit method
func (tx *SerializableTx) Commit(db database.Database) error {
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
	if objMap["body"] == nil {
		return fmt.Errorf("cannot unmarshal serializableTx: body is empty")
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
	commit(db database.Database) error
}
