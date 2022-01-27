package transactions

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	dvotedb "go.vocdoni.io/dvote/db"
)

const (
	TxPrefix        = "tx"
	TimestampPrefix = "tm"
)

type TxCacheDb struct {
	Db  dvotedb.Database
	Mtx *sync.RWMutex
}

func NewTxKv(db dvotedb.Database, mutex *sync.RWMutex) TxCacheDb {
	return TxCacheDb{Db: db, Mtx: mutex}
}

func (kv *TxCacheDb) StoreTx(hash []byte, query SerializableTx) error {
	hash = append([]byte(TxPrefix), hash...)
	queryBytes, err := json.Marshal(&query)
	if err != nil {
		return fmt.Errorf("could not marshal account database transaction: %w", err)
	}
	tx := kv.Db.WriteTx()
	if err = tx.Set(hash, queryBytes); err != nil {
		return fmt.Errorf("could not cache transaction to database: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("could not cache transaction to database: %w", err)
	}
	return nil
}

func (kv *TxCacheDb) GetTx(hash []byte) (*SerializableTx, error) {
	hash = append([]byte(TxPrefix), hash...)
	tx := kv.Db.ReadTx()
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

func (kv *TxCacheDb) DeleteTx(hash []byte) error {
	hash = append([]byte(TxPrefix), hash...)
	// Delete the entry from the kv
	tx := kv.Db.WriteTx()
	if err := tx.Delete(hash); err != nil {
		return fmt.Errorf("could not remove database tx: %w", err)
	}
	return tx.Commit()
}

func (kv *TxCacheDb) StoreTxTime(hash []byte, timestamp time.Time) error {
	hash = append([]byte(TimestampPrefix), hash...)
	queryBytes, err := json.Marshal(timestamp)
	if err != nil {
		return fmt.Errorf("could not marshal transaction timestamp: %w", err)
	}
	tx := kv.Db.WriteTx()
	if err = tx.Set(hash, queryBytes); err != nil {
		return fmt.Errorf("could not cache timestamp to database: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("could not cache timestamp to database: %w", err)
	}
	return nil
}

func (kv *TxCacheDb) GetTxTime(hash []byte) (*time.Time, error) {
	hash = append([]byte(TimestampPrefix), hash...)
	tx := kv.Db.ReadTx()
	queryBytes, err := tx.Get(hash)
	tx.Discard()
	// If key not found, don't return an error
	if err == dvotedb.ErrKeyNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not get query from tx cache: %w", err)
	}
	var timestamp time.Time
	if err = json.Unmarshal(queryBytes, &timestamp); err != nil {
		return nil, fmt.Errorf("could not get query from tx cache: %w", err)
	}
	return &timestamp, nil
}
