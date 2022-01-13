package testapi

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"go.vocdoni.io/api/config"
	"go.vocdoni.io/api/test/testcommon"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/urlapi"
	"go.vocdoni.io/dvote/log"
)

var API testcommon.TestAPI
var testIntegrators []*types.Integrator
var testOrganizations []*testcommon.TestOrganization
var testElections []*testcommon.TestElection
var testActiveElections []*testcommon.TestElection

func TestMain(m *testing.M) {
	storage := os.TempDir()
	apiPort := 9000
	testApiAuthToken := "bb1a42df36d0cf3f4dd53d71dffa15780d44c54a5971792acd31974bc2cbceb6"
	db := &config.DB{
		Dbname:   "postgres",
		Password: "postgres",
		Host:     "postgres",
		Port:     5432,
		Sslmode:  "disable",
		User:     "postgres",
	}
	if err := API.Start(db, "/api", testApiAuthToken, storage, apiPort); err != nil {
		log.Fatalf("SKIPPING: could not start the API: %v", err)
		return
	}
	if err := API.DB.Ping(); err != nil {
		log.Infof("SKIPPING: could not connect to DB: %v", err)
		return
	}
	setupTestIntegrators()
	setupTestOrganizations()
	setupTestElections()
	os.Exit(m.Run())
}

func DoRequest(t *testing.T, url, authToken,
	method string, request interface{}) ([]byte, int) {
	data, err := json.Marshal(request)
	if t != nil {
		t.Logf("making request %s to %s with token %s, data %s", method, url, authToken, string(data))
		qt.Assert(t, err, qt.IsNil)
	} else {
		if err != nil {
			log.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if t != nil {
		qt.Assert(t, err, qt.IsNil)
	} else {
		if err != nil {
			log.Fatal(err)
		}
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	}
	resp, err := http.DefaultClient.Do(req)
	if t != nil {
		qt.Assert(t, err, qt.IsNil)
	} else {
		if err != nil {
			log.Fatal(err)
		}
	}
	if resp == nil {
		return nil, resp.StatusCode
	}
	respBody, err := io.ReadAll(resp.Body)
	if t != nil {
		qt.Assert(t, err, qt.IsNil)
		t.Log(string(respBody))
	} else {
		if err != nil {
			log.Fatal(err)
		}
	}
	return respBody, resp.StatusCode
}

func setupTestIntegrators() {
	log.Infof("setting up test integrators")
	testIntegrators = testcommon.CreateIntegrators(2)
	cspPubKey := hex.EncodeToString(API.CSP.CspPubKey)
	// create two integrators to test with
	for _, integrator := range testIntegrators {
		req := types.APIRequest{
			CspUrlPrefix: integrator.CspUrlPrefix,
			CspPubKey:    cspPubKey,
			Name:         integrator.Name,
			Email:        integrator.Email,
		}
		respBody, statusCode := DoRequest(nil, API.URL+"/v1/admin/accounts", API.AuthToken, "POST", req)
		if statusCode != 200 {
			log.Fatalf("could not create testing integrator")
		}
		var resp types.APIResponse
		err := json.Unmarshal(respBody, &resp)
		if err != nil {
			log.Fatalf("could not create testing integrator: %v", err)
		}
		integrator.ID = resp.ID
		if integrator.SecretApiKey, err = hex.DecodeString(resp.APIKey); err != nil {
			log.Fatal(err)
		}
	}
}

func setupTestOrganizations() {
	log.Infof("setting up test organizations")
	testOrganizations = testcommon.CreateOrganizations(2)
	// create two integrators to test with
	for _, organization := range testOrganizations {
		req := types.APIRequest{
			Name:        organization.Name,
			Description: organization.Description,
			Header:      organization.HeaderURI,
			Avatar:      organization.AvatarURI,
		}
		respBody, statusCode := DoRequest(nil, API.URL+"/v1/priv/account/organizations",
			hex.EncodeToString(testIntegrators[0].SecretApiKey), "POST", req)
		if statusCode != 200 {
			log.Fatalf("could not create testing organization")
		}
		var resp types.APIResponse
		err := json.Unmarshal(respBody, &resp)
		if err != nil {
			log.Fatalf("could not create testing organization: %v", err)
		}
		organization.ID = resp.ID
		organization.EthAddress = resp.OrganizationID
		organization.APIToken = resp.APIToken
		organization.CreationTxHash = resp.TxHash

		// create organization: check txHash has been mined
		var respMined urlapi.APIMined
		for numTries := 5; numTries > 0; numTries-- {
			if numTries != 5 {
				time.Sleep(time.Second * 4)
			}
			req = types.APIRequest{}
			respBody, statusCode = DoRequest(nil, API.URL+
				"/v1/priv/transactions/"+organization.CreationTxHash,
				hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", req)
			if statusCode != 200 {
				log.Fatalf("could not create testing organization")
			}
			err := json.Unmarshal(respBody, &respMined)
			if err != nil {
				log.Fatalf("could not create testing organization: %v", err)
			}
			// if mined, break loop
			if respMined.Mined != nil && *respMined.Mined {
				break
			}
		}
		if respMined.Mined == nil || !*respMined.Mined {
			log.Fatalf("could not create testing organization: tx never mined")
		}
	}
}

func setupTestElections() {
	log.Infof("setting up test elections")

	// set up two sets of elections, one for changing the status, one for voting with
	testElections = createElections(testOrganizations[0])
	testActiveElections = createElections(testOrganizations[1])

	// check that both sets of elections are mined
	checkElectionsMined(testElections)
	checkElectionsMined(testActiveElections)
}

func createElections(organization *testcommon.TestOrganization) []*testcommon.TestElection {
	elections := testcommon.CreateElections(1, false, false)
	elections = append(elections, testcommon.CreateElections(1, true, false)...)
	elections = append(elections, testcommon.CreateElections(1, true, true)...)
	for _, election := range elections {
		var resp types.APIResponse
		req := types.APIRequest{
			Title:         election.Title,
			Description:   election.Description,
			Header:        election.Header,
			StreamURI:     election.StreamURI,
			EndDate:       election.EndDate.Format("2006-01-02T15:04:05.000Z"),
			Confidential:  election.Confidential,
			HiddenResults: election.HiddenResults,
			Questions:     election.Questions,
		}
		respBody, statusCode := DoRequest(nil, API.URL+"/v1/priv/organizations/"+
			hex.EncodeToString(organization.EthAddress)+"/elections/blind",
			hex.EncodeToString(testIntegrators[0].SecretApiKey), "POST", req)
		if statusCode != 200 {
			log.Fatalf("could not create testing organization")
		}
		err := json.Unmarshal(respBody, &resp)
		if err != nil {
			log.Fatalf("could not create testing organization: %v", err)
		}
		election.ElectionID = resp.ElectionID
		election.OrganizationID = organization.EthAddress
		election.CreationTxHash = resp.TxHash
	}
	return elections
}

func checkElectionsMined(elections []*testcommon.TestElection) {
	var respMined urlapi.APIMined
	for _, election := range elections {
		for numTries := 10; numTries > 0; numTries-- {
			if numTries != 10 {
				time.Sleep(time.Second * 4)
			}
			req := types.APIRequest{}
			respBody, statusCode := DoRequest(nil, API.URL+
				"/v1/priv/transactions/"+election.CreationTxHash,
				hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", req)
			// if mined, break loop
			if statusCode != 200 {
				log.Fatalf("could not create testing organization")
			}
			err := json.Unmarshal(respBody, &respMined)
			if err != nil {
				log.Fatalf("could not create testing organization: %v", err)
			}
			if respMined.Mined != nil && *respMined.Mined {
				break
			}
		}
		if respMined.Mined == nil || !*respMined.Mined {
			log.Fatalf("could not create testing organization: tx never mined")
		}
	}
}
