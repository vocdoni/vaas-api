package testcommon

import (
	"path"
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
	TEST_HOST     = "127.0.0.1"
	TEST_GW_PATH  = "/dvote"
	TEST_GW_PORT  = 9090
	TEST_CSP_PATH = "/v1/auth/elections"
	TEST_CSP_PORT = 5000
)

func (t *TestAPI) startTestGateway() {
	storageDir := filepath.Join(t.StorageDir, ".voconed")
	oracle := ethereum.SignKeys{}
	if err := oracle.Generate(); err != nil {
		log.Fatal(err)
	}

	vc, err := vocone.NewVocone(storageDir, &oracle)
	if err != nil {
		log.Fatal(err)
	}

	vc.SetBlockTimeTarget(time.Second)
	vc.SetBlockSize(500)
	go vc.Start()
	if err = vc.EnableAPI(TEST_HOST, TEST_GW_PORT, TEST_GW_PATH); err != nil {
		log.Fatal(err)
	}
	t.Gateway = "http://" + TEST_HOST + ":" + strconv.Itoa(TEST_GW_PORT) + TEST_GW_PATH

	select {}
}

func (t *TestAPI) startTestCSP() {
	dir := path.Join(t.StorageDir, "auth")
	t.CSP.CspSignKeys = &ethereum.SignKeys{}
	if err := t.CSP.CspSignKeys.Generate(); err != nil {
		log.Fatal(err)
	}
	_, privKey := t.CSP.CspSignKeys.HexString()
	log.Infof("new private key generated: %s", privKey)

	router := httprouter.HTTProuter{}
	authHandler := handlers.Handlers["dummy"]
	if err := authHandler.Init(dir); err != nil {
		log.Fatal(err)
	}
	log.Infof("using CSP handler %s", authHandler.GetName())
	// Start the router
	t.CSP.UrlPrefix = TEST_HOST
	t.CSP.CspPubKey = t.CSP.CspSignKeys.PublicKey()
	if err := router.Init(t.CSP.UrlPrefix, TEST_CSP_PORT); err != nil {
		log.Fatal(err)
	}
	// Create the blind CA API and assign the auth function
	pub, priv := t.CSP.CspSignKeys.HexString()
	log.Infof("CSP root public key: %s", pub)
	cs, err := csp.NewBlindCSP(priv, path.Join(dir, authHandler.GetName()), authHandler.Auth)
	if err != nil {
		log.Fatal(err)
	}
	if err := cs.ServeAPI(&router, TEST_CSP_PATH); err != nil {
		log.Fatal(err)
	}

	select {}
}
