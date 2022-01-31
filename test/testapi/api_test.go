package testapi

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
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
	storage, err := ioutil.TempDir("/tmp", ".vaas-test")
	if err != nil {
		log.Fatal(err)
	}
	apiPort := 9000
	testApiAuthToken := "bb1a42df36d0cf3f4dd53d71dffa15780d44c54a5971792acd31974bc2cbceb6"
	// check for TEST_DB_HOST env var. If not exist, don't run the db tests
	dbHost := os.Getenv("TEST_DB_HOST")
	if dbHost == "" {
		log.Infof("SKIPPING: database host not set")
		return
	}
	dbPort, err := strconv.Atoi(os.Getenv("TEST_DB_PORT"))
	if err != nil {
		dbPort = 5432
	}
	db := &config.DB{
		Dbname:   "postgres",
		Password: "postgres",
		Host:     dbHost,
		Port:     dbPort,
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
	os.RemoveAll("/tmp/.vaas-test")
	os.Exit(m.Run())
}

func DoRequest(t *testing.T, url, authToken,
	method string, request interface{}, response interface{}) int {
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
		return resp.StatusCode
	}
	respBody, err := io.ReadAll(resp.Body)
	if t != nil {
		t.Log(string(respBody))
		qt.Assert(t, err, qt.IsNil)
	} else {
		log.Infof(string(respBody))
		if err != nil {
			log.Fatal(err)
		}
	}
	if resp.StatusCode != 200 {
		return resp.StatusCode
	}
	err = json.Unmarshal(respBody, response)
	if t != nil {
		qt.Assert(t, err, qt.IsNil)
	} else {
		if err != nil {
			log.Fatal(err)
		}
	}
	return resp.StatusCode
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
		var resp types.APIResponse
		statusCode := DoRequest(nil,
			fmt.Sprintf("%s/v1/admin/accounts", API.URL),
			API.AuthToken, "POST", req, &resp)
		if statusCode != 200 {
			log.Fatalf("could not create testing integrator")
		}
		integrator.ID = resp.ID
		var err error
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
		var resp types.APIResponse
		statusCode := DoRequest(nil,
			fmt.Sprintf("%s/v1/priv/account/organizations", API.URL),
			hex.EncodeToString(testIntegrators[0].SecretApiKey), "POST", req, &resp)
		if statusCode != 200 {
			log.Fatalf("could not create testing organization")
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
			statusCode = DoRequest(nil,
				fmt.Sprintf("%s/v1/priv/transactions/%s", API.URL, organization.CreationTxHash),
				hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", req, &respMined)
			if statusCode != 200 {
				log.Fatalf("could not create testing organization")
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
	elections := testcommon.CreateElections(1, false, false, types.PROOF_TYPE_BLIND)
	elections = append(elections, testcommon.CreateElections(1, true, false, types.PROOF_TYPE_ECDSA)...)
	elections = append(elections, testcommon.CreateElections(1, true, true, types.PROOF_TYPE_ECDSA)...)
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
		statusCode := DoRequest(nil,
			fmt.Sprintf("%s/v1/priv/organizations/%x/elections/%s", API.URL, organization.EthAddress, election.ProofType),
			hex.EncodeToString(testIntegrators[0].SecretApiKey), "POST", req, &resp)
		if statusCode != 200 {
			log.Fatalf("could not create testing organization")
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
			statusCode := DoRequest(nil,
				fmt.Sprintf("%s/v1/priv/transactions/%s", API.URL, election.CreationTxHash),
				hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", req, &respMined)
			if statusCode != 200 {
				log.Fatalf("could not create testing organization")
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
