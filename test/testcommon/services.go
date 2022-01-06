package testcommon

import (
	"path/filepath"
	"strconv"
	"time"

	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/vocone"
)

const (
	TEST_HOST    = "127.0.0.1"
	TEST_GW_PATH = "/dvote"
	TEST_GW_PORT = 9090
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
