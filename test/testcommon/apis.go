package testcommon

import (
	"go.vocdoni.io/api/config"
	"go.vocdoni.io/api/database"
	"go.vocdoni.io/api/database/pgsql"
	"go.vocdoni.io/api/database/testdb"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
)

type TestAPI struct {
	DB     database.Database
	Port   int
	Signer *ethereum.SignKeys
}

// Start creates a new database connection and API endpoint for testing.
// If dbc is nill the testdb will be used.
// If route is nill, then the websockets API won't be initialized
func (t *TestAPI) Start(dbc *config.DB, route string) error {
	log.Init("info", "stdout")
	var err error
	if route != "" {
		// Signer
		t.Signer = ethereum.NewSignKeys()
		t.Signer.Generate()
	}
	if dbc != nil {
		// Postgres with sqlx
		if t.DB, err = pgsql.New(dbc); err != nil {
			return err
		}
	} else {
		// Mock database
		if t.DB, err = testdb.New(); err != nil {
			return err
		}
	}

	if route != "" {
		log.Infof("enabling API methods")
	}
	return nil
}
