package testcommon

import (
	"fmt"

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
	DB        database.Database
	Port      int
	Signer    *ethereum.SignKeys
	URL       string
	AuthToken string
	CSP       TestCSP
	Gateways  []string
}

type TestCSP struct {
	UrlPrefix string
	CspPubKey dvoteTypes.HexBytes
}

// Start creates a new database connection and API endpoint for testing.
// If dbc is nill the testdb will be used.
// If route is nill, then the websockets API won't be initialized
func (t *TestAPI) Start(dbc *config.DB, route, authToken string, gateway string, port int, csp TestCSP) error {
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
	}

	if route != "" {
		client, err := vocclient.New(gateway, t.Signer)
		if err != nil {
			log.Fatal(err)
		}

		var httpRouter httprouter.HTTProuter
		if err = httpRouter.Init("127.0.0.1", port); err != nil {
			log.Fatal(err)
		}
		// Rest api
		urlApi, err := urlapi.NewURLAPI(&httpRouter, &config.API{
			Route:      route,
			ListenPort: port,
			AdminToken: "test",
			GatewayUrl: gateway,
		}, nil)
		if err != nil {
			log.Fatal(err)
		}

		// Vaas api
		log.Infof("enabling VaaS API methods")
		if err := urlApi.EnableVotingServiceHandlers(t.DB, client); err != nil {
			log.Fatal(err)
		}
		t.URL = fmt.Sprintf("http://127.0.0.1:%d/api", port)
		t.AuthToken = authToken
		t.CSP = csp
	}
	return nil
}
