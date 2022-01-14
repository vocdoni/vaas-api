package testapi

import (
	"encoding/hex"
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
	elections := testcommon.CreateElections(1, false, false, types.PROOF_TYPE_BLIND)
	elections = append(elections, testcommon.CreateElections(
		1, true, false, types.PROOF_TYPE_ECDSA)...)
	elections = append(elections, testcommon.CreateElections(1, true, true, types.PROOF_TYPE_ECDSA)...)

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
		statusCode := DoRequest(t,
			fmt.Sprintf("%s/v1/priv/organizations/%x/elections/%s",
				API.URL, testOrganizations[0].EthAddress, election.ProofType),
			hex.EncodeToString(testIntegrators[0].SecretApiKey), "POST", req, &resp)
		qt.Assert(t, statusCode, qt.Equals, 200)
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
			statusCode := DoRequest(t,
				fmt.Sprintf("%s/v1/priv/transactions/%s", API.URL, election.CreationTxHash),
				hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", req, &respMined)
			qt.Assert(t, statusCode, qt.Equals, 200)
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
			statusCode := DoRequest(t,
				fmt.Sprintf("%s/v1/priv/elections/%x", API.URL, election.ElectionID),
				hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{}, &electionResp)
			qt.Assert(t, statusCode, qt.Equals, 200)
			qt.Assert(t, electionResp.Description, qt.Equals, election.Description)
			qt.Assert(t, electionResp.Title, qt.Equals, election.Title)
			qt.Assert(t, electionResp.Header, qt.Equals, election.Header)
			qt.Assert(t, electionResp.StreamURI, qt.Equals, election.StreamURI)
			qt.Assert(t, electionResp.Questions, qt.HasLen, len(election.Questions))
			qt.Assert(t, electionResp.ProofType, qt.Equals, election.ProofType)
			qt.Assert(t, hex.EncodeToString(electionResp.OrganizationID),
				qt.Equals, hex.EncodeToString(election.OrganizationID))
			qt.Assert(t, electionResp.ElectionID, qt.Not(qt.HasLen), 0)
			for i, question := range electionResp.Questions {
				qt.Assert(t, question.Title, qt.Equals, election.Questions[i].Title)
				qt.Assert(t, question.Description, qt.Equals, election.Questions[i].Description)
				qt.Assert(t, question.Choices, qt.HasLen, len(election.Questions[i].Choices))
			}
			status = electionResp.Status
			numTries--
		}
		qt.Assert(t, electionResp.Status, qt.Equals, "ACTIVE")
	}
}

func TestElectionStatus(t *testing.T) {
	t.Parallel()
	// test set election status
	for _, election := range testElections {
		var resp *types.APIResponse
		statusCode := DoRequest(t,
			fmt.Sprintf("%s/v1/priv/elections/%x/CANCELED", API.URL, election.ElectionID),
			hex.EncodeToString(testIntegrators[0].SecretApiKey), "PUT", types.APIRequest{}, &resp)
		qt.Assert(t, statusCode, qt.Equals, 200)
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
			statusCode := DoRequest(t,
				fmt.Sprintf("%s/v1/priv/transactions/%s", API.URL, election.CreationTxHash),
				hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", req, &respMined)
			qt.Assert(t, statusCode, qt.Equals, 200)
			// if mined, break loop
			if respMined.Mined != nil && *respMined.Mined {
				break
			}
		}
		qt.Assert(t, respMined.Mined, qt.Not(qt.IsNil))
		qt.Assert(t, *respMined.Mined, qt.IsTrue)
	}

	// get canceled election list
	var canceledElectionList []types.APIElectionSummary
	statusCode := DoRequest(t,
		fmt.Sprintf("%s/v1/priv/organizations/%x/elections/canceled",
			API.URL, testOrganizations[0].EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey),
		"GET", types.APIRequest{}, &canceledElectionList)
	qt.Assert(t, statusCode, qt.Equals, 200)
	qt.Assert(t, canceledElectionList, qt.HasLen, len(testElections))

	// test get election statuses
	for _, election := range testElections {
		var electionResp types.APIElectionInfo
		statusCode := DoRequest(t,
			fmt.Sprintf("%s/v1/priv/elections/%x", API.URL, election.ElectionID),
			hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{}, &electionResp)
		qt.Assert(t, statusCode, qt.Equals, 200)
		qt.Assert(t, electionResp.Status, qt.Equals, "CANCELED")
	}
}

func TestElectionList(t *testing.T) {
	t.Parallel()
	// get election list
	var electionList []types.APIElectionSummary
	statusCode := DoRequest(t,
		fmt.Sprintf("%s/v1/priv/organizations/%x/elections", API.URL, testOrganizations[1].EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{}, &electionList)
	qt.Assert(t, statusCode, qt.Equals, 200)

	// get active election list
	var activeElectionList []types.APIElectionSummary
	statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/organizations/%x/elections/active",
			API.URL, testOrganizations[1].EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{}, &activeElectionList)
	qt.Assert(t, statusCode, qt.Equals, 200)

	// get election lists with empty filters
	var emptyElectionList []types.APIElectionSummary
	statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/organizations/%x/elections/upcoming",
			API.URL, testOrganizations[1].EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{}, &emptyElectionList)
	qt.Assert(t, statusCode, qt.Equals, 200)
	qt.Assert(t, emptyElectionList, qt.HasLen, 0)

	statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/organizations/%x/elections/ended",
			API.URL, testOrganizations[1].EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{}, &emptyElectionList)
	qt.Assert(t, statusCode, qt.Equals, 200)
	qt.Assert(t, emptyElectionList, qt.HasLen, 0)

	statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/organizations/%x/elections/canceled",
			API.URL, testOrganizations[1].EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{}, &emptyElectionList)
	qt.Assert(t, statusCode, qt.Equals, 200)
	qt.Assert(t, emptyElectionList, qt.HasLen, 0)

	statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/organizations/%x/elections/paused",
			API.URL, testOrganizations[1].EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{}, &emptyElectionList)
	qt.Assert(t, statusCode, qt.Equals, 200)
	qt.Assert(t, emptyElectionList, qt.HasLen, 0)

	// get blind election list
	var blindElectionList []types.APIElectionSummary
	statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/organizations/%x/elections/blind",
			API.URL, testOrganizations[1].EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey),
		"GET", types.APIRequest{}, &blindElectionList)
	qt.Assert(t, statusCode, qt.Equals, 200)
	qt.Assert(t, blindElectionList, qt.HasLen, 1)

	// get signed election list
	var signedElectionList []types.APIElectionSummary
	statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/organizations/%x/elections/signed",
			API.URL, testOrganizations[1].EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey),
		"GET", types.APIRequest{}, &signedElectionList)
	qt.Assert(t, statusCode, qt.Equals, 200)
	qt.Assert(t, signedElectionList, qt.HasLen, 2)

	qt.Assert(t, electionList, qt.HasLen, len(testActiveElections))
	qt.Assert(t, electionList, qt.HasLen, len(activeElectionList))
}
