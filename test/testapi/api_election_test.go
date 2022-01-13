package testapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"go.vocdoni.io/api/test/testcommon"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/urlapi"
)

func TestElection(t *testing.T) {
	t.Parallel()

	// test create different kinds of elections
	elections := testcommon.CreateElections(1, false, false)
	elections = append(elections, testcommon.CreateElections(1, true, false)...)
	elections = append(elections, testcommon.CreateElections(1, true, true)...)

	for _, election := range elections {
		var resp *types.APIResponse
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
		respBody, statusCode := DoRequest(t,
			fmt.Sprintf("%s/v1/priv/organizations/%x/elections/blind",
				API.URL, testOrganizations[0].EthAddress),
			hex.EncodeToString(testIntegrators[0].SecretApiKey), "POST", req)
		t.Logf("%s", respBody)
		qt.Assert(t, statusCode, qt.Equals, 200)
		err := json.Unmarshal(respBody, &resp)
		qt.Assert(t, err, qt.IsNil)
		election.ElectionID = resp.ElectionID
		election.OrganizationID = testOrganizations[0].EthAddress
		election.CreationTxHash = resp.TxHash
	}

	// create election: check txHash has been mined
	var respMined urlapi.APIMined
	for _, election := range elections {
		for numTries := 10; numTries > 0; numTries-- {
			if numTries != 10 {
				time.Sleep(time.Second * 4)
			}
			req := types.APIRequest{}
			respBody, statusCode := DoRequest(t,
				fmt.Sprintf("%s/v1/priv/transactions/%s", API.URL, election.CreationTxHash),
				hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", req)
			t.Logf("%s", respBody)
			qt.Assert(t, statusCode, qt.Equals, 200)
			err := json.Unmarshal(respBody, &respMined)
			qt.Assert(t, err, qt.IsNil)
			// if mined, break loop
			if respMined.Mined != nil && *respMined.Mined {
				break
			}
		}
		qt.Assert(t, *respMined.Mined, qt.IsTrue)
	}

	// test get elections
	for _, election := range elections {
		var status string
		numTries := 10
		var electionResp types.APIElectionInfo
		for status != "ACTIVE" && numTries > 0 {
			if status != "" {
				time.Sleep(2 * time.Second)
			}
			respBody, statusCode := DoRequest(t,
				fmt.Sprintf("%s/v1/priv/elections/%x", API.URL, election.ElectionID),
				hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{})
			t.Logf("%s", respBody)
			qt.Assert(t, statusCode, qt.Equals, 200)
			err := json.Unmarshal(respBody, &electionResp)
			qt.Assert(t, err, qt.IsNil)
			qt.Assert(t, electionResp.Description, qt.Equals, election.Description)
			qt.Assert(t, electionResp.Title, qt.Equals, election.Title)
			qt.Assert(t, electionResp.Header, qt.Equals, election.Header)
			qt.Assert(t, electionResp.StreamURI, qt.Equals, election.StreamURI)
			qt.Assert(t, len(electionResp.Questions), qt.Equals, len(election.Questions))
			qt.Assert(t, hex.EncodeToString(electionResp.OrganizationID),
				qt.Equals, hex.EncodeToString(election.OrganizationID))
			qt.Assert(t, len(electionResp.ElectionID) > 0, qt.IsTrue)
			election.ElectionID = electionResp.ElectionID
			for i, question := range electionResp.Questions {
				qt.Assert(t, question.Title, qt.Equals, election.Questions[i].Title)
				qt.Assert(t, question.Description, qt.Equals, election.Questions[i].Description)
				qt.Assert(t, len(question.Choices), qt.Equals, len(election.Questions[i].Choices))
			}
			status = electionResp.Status
			numTries--
		}
		qt.Assert(t, electionResp.Status, qt.Equals, "ACTIVE")
	}
}

func TestElectionStatus(t *testing.T) {
	// test set election status
	for _, election := range testElections {
		var resp *types.APIResponse
		respBody, statusCode := DoRequest(t,
			fmt.Sprintf("%s/v1/priv/elections/%x/CANCELED", API.URL, election.ElectionID),
			hex.EncodeToString(testIntegrators[0].SecretApiKey), "PUT", types.APIRequest{})
		t.Logf("%s", respBody)
		qt.Assert(t, statusCode, qt.Equals, 200)
		err := json.Unmarshal(respBody, &resp)
		qt.Assert(t, err, qt.IsNil)
		election.CreationTxHash = resp.TxHash
	}

	// set election status: check txHash has been mined
	var respMined urlapi.APIMined
	for _, election := range testElections {
		for numTries := 10; numTries > 0; numTries-- {
			if numTries != 10 {
				time.Sleep(time.Second * 4)
			}
			req := types.APIRequest{}
			respBody, statusCode := DoRequest(t,
				fmt.Sprintf("%s/v1/priv/transactions/%s", API.URL, election.CreationTxHash),
				hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", req)
			t.Logf("%s", respBody)
			qt.Assert(t, statusCode, qt.Equals, 200)
			err := json.Unmarshal(respBody, &respMined)
			qt.Assert(t, err, qt.IsNil)
			// if mined, break loop
			if respMined.Mined != nil && *respMined.Mined {
				break
			}
		}
		qt.Assert(t, respMined.Mined, qt.Not(qt.IsNil))
		qt.Assert(t, *respMined.Mined, qt.IsTrue)
	}

	// test get election statuses
	for _, election := range testElections {
		var electionResp types.APIElectionInfo
		respBody, statusCode := DoRequest(t,
			fmt.Sprintf("%s/v1/priv/elections/%x", API.URL, election.ElectionID),
			hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{})
		t.Logf("%s", respBody)
		qt.Assert(t, statusCode, qt.Equals, 200)
		err := json.Unmarshal(respBody, &electionResp)
		qt.Assert(t, err, qt.IsNil)
		qt.Assert(t, electionResp.Status, qt.Equals, "CANCELED")
	}
}

func TestElectionList(t *testing.T) {
	// get election list
	respBody, statusCode := DoRequest(t,
		fmt.Sprintf("%s/v1/priv/organizations/%x/elections", API.URL, testOrganizations[1].EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{})
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)
	var electionList []types.APIElectionSummary
	err := json.Unmarshal(respBody, &electionList)
	qt.Assert(t, err, qt.IsNil)

	// get active election list
	respBody, statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/organizations/%x/elections", API.URL, testOrganizations[1].EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{})
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)
	var activeElectionList []types.APIElectionSummary
	err = json.Unmarshal(respBody, &activeElectionList)
	qt.Assert(t, err, qt.IsNil)

	qt.Assert(t, len(electionList), qt.Equals, len(testActiveElections))
	qt.Assert(t, len(electionList), qt.Equals, len(activeElectionList))
}
