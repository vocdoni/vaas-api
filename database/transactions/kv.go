package transactions

import (
	"encoding/json"
	"fmt"

	dvotedb "go.vocdoni.io/dvote/db"
)

func StoreTx(kv dvotedb.Database, hash []byte, query SerializableTx) error {
	queryBytes, err := json.Marshal(&query)
	if err != nil {
		return fmt.Errorf("could not marshal account database transaction: %w", err)
	}
	tx := kv.WriteTx()
	if err = tx.Set(hash, queryBytes); err != nil {
		return fmt.Errorf("could not cache transaction to database: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("could not cache transaction to database: %w", err)
	}
	return nil
}

func GetTx(kv dvotedb.Database, hash []byte) (*SerializableTx, error) {
	tx := kv.ReadTx()
	queryBytes, err := tx.Get(hash)
	tx.Discard()
	// If key not found, don't return an error
	if err == dvotedb.ErrKeyNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not get query from tx cache: %w", err)
	}
	var serializableTx SerializableTx
	if err = json.Unmarshal(queryBytes, &serializableTx); err != nil {
		return nil, fmt.Errorf("could not get query from tx cache: %w", err)
	}
	return &serializableTx, nil
}

func DeleteTx(kv dvotedb.Database, hash []byte) error {
	// Delete the entry from the kv
	tx := kv.WriteTx()
	if err := tx.Delete(hash); err != nil {
		return fmt.Errorf("could not remove database tx: %w", err)
	}
	return tx.Commit()
}
