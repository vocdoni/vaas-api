package testcommon

import (
	"fmt"
	"time"

	"go.vocdoni.io/api/config"
	"go.vocdoni.io/api/database"
	"go.vocdoni.io/api/database/pgsql"
	"go.vocdoni.io/api/urlapi"
	"go.vocdoni.io/api/vocclient"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/log"
	dvoteTypes "go.vocdoni.io/dvote/types"
)

type TestAPI struct {
	DB         database.Database
	Port       int
	Signer     *ethereum.SignKeys
	URL        string
	AuthToken  string
	CSP        TestCSP
	Gateway    string
	StorageDir string
}

type TestCSP struct {
	UrlPrefix string
	CspPubKey dvoteTypes.HexBytes
}

// Start creates a new database connection and API endpoint for testing.
// If dbc is nil the testdb will be used.
// If route is nil, then the websockets API, CSP, and Vocone won't be initialized
// If route is nil, storageDir is not needed
func (t *TestAPI) Start(dbc *config.DB, route, authToken, storageDir string, port int) error {
	log.Init("info", "stdout")
	var err error
	if route != "" {
		// Signer
		t.Signer = ethereum.NewSignKeys()
		if err = t.Signer.Generate(); err != nil {
			log.Fatal(err)
		}
	}
	if dbc != nil {
		// Postgres with sqlx
		if t.DB, err = pgsql.New(dbc); err != nil {
			return err
		}
	}
	if err := pgsql.Migrator("upSync", t.DB); err != nil {
		log.Fatal(err)
	}

	if route != "" {
		t.StorageDir = storageDir
		// create gateway/vocone
		go t.startTestGateway()
		// start testing CSP
		go t.startTestCSP()

		// start API
		time.Sleep(time.Second * 5)
		client, err := vocclient.New(t.Gateway, t.Signer)
		if err != nil {
			log.Fatal(err)
		}

		var httpRouter httprouter.HTTProuter
		if err = httpRouter.Init(TEST_HOST, port); err != nil {
			log.Fatal(err)
		}
		// Rest api
		urlApi, err := urlapi.NewURLAPI(&httpRouter, &config.API{
			Route:      route,
			ListenPort: port,
			AdminToken: authToken,
			GatewayUrl: t.Gateway,
		}, nil)
		if err != nil {
			log.Fatal(err)
		}

		// Vaas api
		log.Infof("enabling VaaS API methods")
		if err := urlApi.EnableVotingServiceHandlers(t.DB, client); err != nil {
			log.Fatal(err)
		}
		t.URL = fmt.Sprintf("http://%s:%d%s", TEST_HOST, port, route)
		t.AuthToken = authToken
		time.Sleep(time.Second * 10)
	}
	return nil
}
