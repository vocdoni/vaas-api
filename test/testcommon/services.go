package testcommon

import (
	"path/filepath"
	"strconv"
	"time"

	"github.com/vocdoni/blind-csp/csp"
	"github.com/vocdoni/blind-csp/handlers"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/vocone"
)

const (
	TestHost    = "127.0.0.1"
	TestGWPath  = "/dvote"
	TestGWPort  = 9090
	TestCSPPath = "/v1/auth/elections"
	TestCSPPort = 5000
)

func (t *TestAPI) startTestGateway() {
	storageDir := filepath.Join(t.StorageDir, ".voconed")
	oracle := ethereum.SignKeys{}
	if err := oracle.Generate(); err != nil {
		log.Fatal(err)
	}

	var err error
	if t.VC, err = vocone.NewVocone(storageDir, &oracle); err != nil {
		log.Fatal(err)
	}
	t.VC.SetBlockTimeTarget(time.Second)
	t.VC.SetBlockSize(500)
	// Set treasurer so we can mint tokens
	treasurer := ethereum.NewSignKeys()
	if err := treasurer.Generate(); err != nil {
		log.Fatal(err)
	}
	if err := t.VC.SetTreasurer(treasurer.Address()); err != nil {
		log.Fatal(err)
	}
	// Set transaction costs
	if err := t.VC.SetBulkTxCosts(10); err != nil {
		log.Fatal(err)
	}
	go t.VC.Start()
	if err := t.VC.EnableAPI(TestHost, TestGWPort, TestGWPath); err != nil {
		log.Fatal(err)
	}
	t.Gateway = "http://" + TestHost + ":" + strconv.Itoa(TestGWPort) + TestGWPath
}

func (t *TestAPI) startTestCSP() {
	dir := filepath.Join(t.StorageDir, "auth")
	t.CSP.CspSignKeys = &ethereum.SignKeys{}
	if err := t.CSP.CspSignKeys.Generate(); err != nil {
		log.Fatal(err)
	}
	_, privKey := t.CSP.CspSignKeys.HexString()
	log.Infof("new private key generated: %s", privKey)

	router := httprouter.HTTProuter{}
	// set router prometheusID so it does not conflict with any other router services
	router.PrometheusID = "csp-chi"
	authHandler := handlers.Handlers["dummy"]
	if err := authHandler.Init(dir); err != nil {
		log.Fatal(err)
	}
	log.Infof("using CSP handler %s", authHandler.GetName())
	// Start the router
	t.CSP.UrlPrefix = TestHost
	t.CSP.CspPubKey = t.CSP.CspSignKeys.PublicKey()
	if err := router.Init(t.CSP.UrlPrefix, TestCSPPort); err != nil {
		log.Fatal(err)
	}
	// Create the blind CA API and assign the auth function
	pub, priv := t.CSP.CspSignKeys.HexString()
	log.Infof("CSP root public key: %s", pub)
	cs, err := csp.NewBlindCSP(priv, filepath.Join(dir, authHandler.GetName()), authHandler.Auth)
	if err != nil {
		log.Fatal(err)
	}
	if err := cs.ServeAPI(&router, TestCSPPath); err != nil {
		log.Fatal(err)
	}
}
