package testapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	qt "github.com/frankban/quicktest"
	"go.vocdoni.io/api/config"
	"go.vocdoni.io/api/test/testcommon"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/log"
)

var API testcommon.TestAPI

func TestMain(m *testing.M) {
	db := &config.DB{
		Dbname:   "vaas",
		Password: "vocdoni",
		Host:     "127.0.0.1",
		Port:     5432,
		Sslmode:  "disable",
		User:     "vocdoni",
	}
	storage := os.TempDir()
	apiPort := 9000
	apiAuthToken := "bb1a42df36d0cf3f4dd53d71dffa15780d44c54a5971792acd31974bc2cbceb6"
	if err := API.Start(db, "/api", apiAuthToken, storage, apiPort); err != nil {
		log.Fatalf("SKIPPING: could not start the API: %v", err)
		return
	}
	if err := API.DB.Ping(); err != nil {
		log.Infof("SKIPPING: could not connect to DB: %v", err)
		return
	}
	os.Exit(m.Run())
}

func DoRequest(t *testing.T, url, authToken, method string, request types.APIRequest) ([]byte, int) {
	log.Infof("making request %s to %s", method, url)
	data, err := json.Marshal(request)
	qt.Check(t, err, qt.IsNil)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	qt.Check(t, err, qt.IsNil)
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	}
	resp, err := http.DefaultClient.Do(req)
	qt.Check(t, err, qt.IsNil)
	respBody, err := io.ReadAll(resp.Body)
	qt.Check(t, err, qt.IsNil)
	return respBody, resp.StatusCode
}
