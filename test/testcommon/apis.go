package testcommon

import (
	"fmt"
	"path/filepath"
	"time"

	"go.vocdoni.io/api/config"
	"go.vocdoni.io/api/database"
	"go.vocdoni.io/api/database/pgsql"
	"go.vocdoni.io/api/urlapi"
	"go.vocdoni.io/api/vocclient"
	"go.vocdoni.io/dvote/crypto/ethereum"
	dvotedb "go.vocdoni.io/dvote/db"
	"go.vocdoni.io/dvote/db/metadb"
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
	UrlPrefix   string
	CspPubKey   dvoteTypes.HexBytes
	CspSignKeys *ethereum.SignKeys
}

// Start creates a new mock database and API endpoint for testing.
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
		if err := pgsql.Migrator("upSync", t.DB); err != nil {
			log.Warn(err)
		}
		// Start token notifier
	}
	integratorTokenNotifier, _ := pgsql.NewNotifier(dbc, "integrator_tokens_update")

	if route != "" {
		t.StorageDir = storageDir

		// create gateway/vocone
		t.startTestGateway()

		// create CSP service
		t.startTestCSP()

		// start API
		time.Sleep(time.Second * 5)
		client, err := vocclient.New(t.Gateway, t.Signer)
		if err != nil {
			log.Fatal(err)
		}

		var httpRouter httprouter.HTTProuter
		// set router prometheusID so it does not conflict with any other router services
		httpRouter.PrometheusID = "api-chi"
		if err = httpRouter.Init(TestHost, port); err != nil {
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

		kv, err := metadb.New(dvotedb.TypePebble, filepath.Join(t.StorageDir, "metadb"))
		if err != nil {
			log.Fatal(err)
		}

		// Vaas api
		log.Infof("enabling VaaS API methods")
		if err := urlApi.EnableVotingServiceHandlers(t.DB, kv, client); err != nil {
			log.Fatal(err)
		}
		go integratorTokenNotifier.FetchNewTokens(urlApi)
		t.URL = fmt.Sprintf("http://%s:%d%s", TestHost, port, route)
		t.AuthToken = authToken
		time.Sleep(time.Second)
	}
	return nil
}
