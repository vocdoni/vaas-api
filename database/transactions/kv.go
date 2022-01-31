package transactions

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	dvotedb "go.vocdoni.io/dvote/db"
)

const (
	TxPrefix        = "tx"
	TimestampPrefix = "tm"
)

// TxCacheDB is a wrapper for the dvote database type, plus a mutex.
// It provides methods for storing serializable sql transactions relevant to the VaaS
type TxCacheDB struct {
	sync.RWMutex
	DB dvotedb.Database
}

// NewTxKv creates a new TxCacheDB type to store database transactions
func NewTxKv(db dvotedb.Database) *TxCacheDB {
	return &TxCacheDB{DB: db}
}

// StoreTx marshals & stores a SerializableTx with the given hash
func (kv *TxCacheDB) StoreTx(hash []byte, query SerializableTx) error {
	queryBytes, err := json.Marshal(&query)
	if err != nil {
		return fmt.Errorf("could not marshal account database transaction: %w", err)
	}
	kvTransaction := kv.DB.WriteTx()
	if err := kvTransaction.Set(append([]byte(TxPrefix), hash...), queryBytes); err != nil {
		return fmt.Errorf("could not cache transaction to database: %w", err)
	}
	if err := kvTransaction.Commit(); err != nil {
		return fmt.Errorf("could not cache transaction to database: %w", err)
	}
	return nil
}

// GetTx retrieves and unmarshals a SerializableTx with the given hash.
// If the tx is not found but there is no error otherwise, no error or tx is returned.
func (kv *TxCacheDB) GetTx(hash []byte) (*SerializableTx, error) {
	kvTransaction := kv.DB.ReadTx()
	queryBytes, err := kvTransaction.Get(append([]byte(TxPrefix), hash...))
	kvTransaction.Discard()
	// If key not found, don't return an error
	if errors.Is(err, dvotedb.ErrKeyNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not get query from tx cache: %w", err)
	}
	var serializableTx SerializableTx
	if err := json.Unmarshal(queryBytes, &serializableTx); err != nil {
		return nil, fmt.Errorf("could not get query from tx cache: %w", err)
	}
	return &serializableTx, nil
}

// DeleteTx deletes a tx entry from the kv
func (kv *TxCacheDB) DeleteTx(hash []byte) error {
	// Delete the entry from the kv
	kvTransaction := kv.DB.WriteTx()
	if err := kvTransaction.Delete(append([]byte(TxPrefix), hash...)); err != nil {
		return fmt.Errorf("could not remove database tx: %w", err)
	}
	return kvTransaction.Commit()
}

// StoreTxTime stores the creation timestamp associated with a Tx hash to the kv
func (kv *TxCacheDB) StoreTxTime(hash []byte, timestamp time.Time) error {
	queryBytes, err := json.Marshal(timestamp)
	if err != nil {
		return fmt.Errorf("could not marshal transaction timestamp: %w", err)
	}
	kvTransaction := kv.DB.WriteTx()
	if err := kvTransaction.Set(append([]byte(TimestampPrefix), hash...), queryBytes); err != nil {
		return fmt.Errorf("could not cache timestamp to database: %w", err)
	}
	if err := kvTransaction.Commit(); err != nil {
		return fmt.Errorf("could not cache timestamp to database: %w", err)
	}
	return nil
}

// StoreTxTime gets the creation timestamp associated with a Tx hash from the kv
func (kv *TxCacheDB) GetTxTime(hash []byte) (*time.Time, error) {
	kvTransaction := kv.DB.ReadTx()
	queryBytes, err := kvTransaction.Get(append([]byte(TimestampPrefix), hash...))
	kvTransaction.Discard()
	// If key not found, don't return an error
	if err == dvotedb.ErrKeyNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not get query from tx cache: %w", err)
	}
	var timestamp time.Time
	if err := json.Unmarshal(queryBytes, &timestamp); err != nil {
		return nil, fmt.Errorf("could not get query from tx cache: %w", err)
	}
	return &timestamp, nil
}
