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
	"go.vocdoni.io/dvote/log"
)

var API testcommon.TestAPI

func TestMain(m *testing.M) {
	storage := os.TempDir()
	apiPort := 9000
	testApiAuthToken := "bb1a42df36d0cf3f4dd53d71dffa15780d44c54a5971792acd31974bc2cbceb6"
	db := &config.DB{
		Dbname:   "postgres",
		Password: "postgres",
		Host:     "localhost",
		Port:     5432,
		Sslmode:  "disable",
		User:     "postgres",
	}
	if err := API.Start(db, "/api", testApiAuthToken, storage, apiPort); err != nil {
		log.Infof("SKIPPING: could not start the API: %v", err)
		return
	}
	if err := API.DB.Ping(); err != nil {
		log.Infof("SKIPPING: could not connect to DB: %v", err)
		return
	}
	os.Exit(m.Run())
}

func DoRequest(t *testing.T, url, authToken,
	method string, request interface{}) ([]byte, int) {
	data, err := json.Marshal(request)
	t.Logf("making request %s to %s with token %s, data %s", method, url, authToken, string(data))
	qt.Check(t, err, qt.IsNil)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	qt.Check(t, err, qt.IsNil)
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	}
	resp, err := http.DefaultClient.Do(req)
	qt.Assert(t, err, qt.IsNil)
	if resp == nil {
		return nil, resp.StatusCode
	}
	respBody, err := io.ReadAll(resp.Body)
	qt.Check(t, err, qt.IsNil)
	t.Log(string(respBody))
	return respBody, resp.StatusCode
}
