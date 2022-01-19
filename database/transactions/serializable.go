package transactions

import (
	"go.vocdoni.io/api/database"
)

// SerializableTx is a database transaction that can be serialized and saved for later.
// SerializableTx.Commit() attempts to commit this query to the database, and returns
//  the id of the new database entry, if one exists.
type SerializableTx interface {
	Commit(db *database.Database) (int, error)
}
