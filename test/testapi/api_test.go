package testapi

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	qt "github.com/frankban/quicktest"
	"go.vocdoni.io/api/config"
	"go.vocdoni.io/api/test/testcommon"
	"go.vocdoni.io/api/types"
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

	var csp testcommon.TestCSP
	csp.UrlPrefix = "https://csp-dev.vocdoni.net"
	var err error
	csp.CspPubKey, err = hex.DecodeString("0383f236571b7e35d2a7a1f1dc5a31bd483863800bff31e36486faccb8c0d68d9d")
	if err != nil {
		log.Fatal(err)
	}
	// var API testcommon.TestAPI
	apiPort := 9000
	apiAuthToken := "bb1a42df36d0cf3f4dd53d71dffa15780d44c54a5971792acd31974bc2cbceb6"
	apiGateway := "https://api-dev.vocdoni.net/dvote"
	if err := API.Start(db, "/api", apiAuthToken, apiGateway, apiPort, csp); err != nil {
		log.Printf("SKIPPING: could not start the API: %v", err)
		return
	}

	os.Exit(m.Run())
	if err := API.DB.Ping(); err != nil {
		log.Printf("SKIPPING: could not connect to DB: %v", err)
		return
	}
	os.Exit(m.Run())
}

func DoRequest(t *testing.T, url, authToken, method string, request types.APIRequest) types.APIResponse {
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
	var response types.APIResponse
	err = json.Unmarshal([]byte(respBody), &response)
	qt.Check(t, err, qt.IsNil)
	return response
}
