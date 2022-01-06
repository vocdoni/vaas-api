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
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/log"
)

func TestPublic(t *testing.T) {
	integrator := testcommon.CreateIntegrators(1)[0]
	// generate new key pair to use as csp keys so we can test public methods
	cspSignKeys := ethereum.NewSignKeys()
	if err := cspSignKeys.Generate(); err != nil {
		t.Fatalf("could not generate sign keys: %v", err)
	}
	pub, _ := cspSignKeys.HexString()

	// create integrator to test with
	req := types.APIRequest{
		CspUrlPrefix: integrator.CspUrlPrefix,
		CspPubKey:    pub,
		Name:         integrator.Name,
		Email:        integrator.Email,
	}
	respBody, statusCode := DoRequest(t, API.URL+"/v1/admin/accounts", API.AuthToken, "POST", req)
	qt.Assert(t, statusCode, qt.Equals, 200)
	var resp types.APIResponse
	err := json.Unmarshal(respBody, &resp)
	qt.Assert(t, err, qt.IsNil)
	integrator.ID = resp.ID
	if integrator.SecretApiKey, err = hex.DecodeString(resp.APIKey); err != nil {
		log.Fatal(err)
	}

	// create organization to test with
	organization := testcommon.CreateOrganizations(1)[0]
	req = types.APIRequest{
		Name:        organization.Name,
		Description: organization.Description,
		Header:      organization.HeaderURI,
		Avatar:      organization.AvatarURI,
	}
	respBody, statusCode = DoRequest(t, API.URL+"/v1/priv/account/organizations",
		hex.EncodeToString(integrator.SecretApiKey), "POST", req)
	qt.Assert(t, statusCode, qt.Equals, 200)
	err = json.Unmarshal(respBody, &resp)
	qt.Assert(t, err, qt.IsNil)
	organization.ID = resp.ID
	organization.EthAddress = resp.OrganizationID
	organization.CreationTxHash = resp.TxHash

	// create organization: check txHash has been mined
	var respMined urlapi.APIMined
	for numTries := 5; numTries > 0; numTries-- {
		if numTries != 5 {
			time.Sleep(time.Second * 4)
		}
		req = types.APIRequest{}
		respBody, statusCode = DoRequest(t, API.URL+
			"/v1/priv/transactions/"+organization.CreationTxHash,
			hex.EncodeToString(integrator.SecretApiKey), "GET", req)
		log.Infof("%s", respBody)
		qt.Assert(t, statusCode, qt.Equals, 200)
		err := json.Unmarshal(respBody, &respMined)
		qt.Assert(t, err, qt.IsNil)
		// if mined, break loop
		if respMined.Mined != nil && *respMined.Mined {
			break
		}
	}
	qt.Assert(t, *respMined.Mined, qt.IsTrue)

	// test create different kinds of elections
	elections := testcommon.CreateElections(1, false, false)
	elections = append(elections, testcommon.CreateElections(1, true, false)...)
	elections = append(elections, testcommon.CreateElections(1, true, true)...)

	for _, election := range elections {
		var resp *types.APIResponse
		req = types.APIRequest{
			Title:         election.Title,
			Description:   election.Description,
			Header:        election.Header,
			StreamURI:     election.StreamURI,
			EndDate:       election.EndDate.Format("2006-01-02T15:04:05.000Z"),
			Confidential:  election.Confidential,
			HiddenResults: election.HiddenResults,
			Questions:     election.Questions,
		}
		respBody, statusCode = DoRequest(t, API.URL+"/v1/priv/organizations/"+
			hex.EncodeToString(organization.EthAddress)+"/elections/blind",
			hex.EncodeToString(integrator.SecretApiKey), "POST", req)
		log.Infof("%s", respBody)
		qt.Assert(t, statusCode, qt.Equals, 200)
		err = json.Unmarshal(respBody, &resp)
		qt.Assert(t, err, qt.IsNil)
		election.ElectionID = resp.ElectionID
		election.OrganizationID = organization.EthAddress
		election.CreationTxHash = resp.TxHash
	}

	// create election: check txHash has been mined
	for _, election := range elections {
		for numTries := 10; numTries > 0; numTries-- {
			if numTries != 10 {
				time.Sleep(time.Second * 4)
			}
			req = types.APIRequest{}
			respBody, statusCode = DoRequest(t, API.URL+
				"/v1/priv/transactions/"+election.CreationTxHash,
				hex.EncodeToString(integrator.SecretApiKey), "GET", req)
			log.Infof("%s", respBody)
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

	// get election list- public call (should exclude private elections)
	respBody, statusCode = DoRequest(t, API.URL+
		"/v1/pub/organizations/"+hex.EncodeToString(organization.EthAddress)+"/elections",
		organization.APIToken, "GET", types.APIRequest{})
	log.Infof("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)
	var pubElectionList []types.APIElectionSummary
	err = json.Unmarshal(respBody, &pubElectionList)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, len(pubElectionList), qt.Equals, 1)

	// test get elections (pub)
	for _, election := range elections {
		var electionResp types.APIElectionInfo
		respBody, statusCode = DoRequest(t, API.URL+
			"/v1/pub/elections/"+hex.EncodeToString(election.ElectionID),
			organization.APIToken, "GET", types.APIRequest{})
		log.Infof("%s", respBody)
		if election.Confidential {
			qt.Assert(t, statusCode, qt.Equals, 400)
			break
		}
		qt.Assert(t, statusCode, qt.Equals, 200)
		err = json.Unmarshal(respBody, &electionResp)
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
	}

	// test get elections (priv)
	for _, election := range elections {
		cspSignature := testcommon.GetCSPSignature(election.ElectionID, cspSignKeys)
		var electionResp types.APIElectionInfo
		respBody, statusCode = DoRequest(t, API.URL+
			"/v1/pub/elections/"+hex.EncodeToString(election.ElectionID)+"/auth/"+cspSignature,
			organization.APIToken, "GET", types.APIRequest{})
		log.Infof("%s", respBody)
		qt.Assert(t, statusCode, qt.Equals, 200)
		err = json.Unmarshal(respBody, &electionResp)
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
	}

	// cleaning up
	respBody, statusCode = DoRequest(t, fmt.Sprintf("%s/v1/priv/account/organizations/"+
		hex.EncodeToString(organization.EthAddress), API.URL),
		hex.EncodeToString(integrator.SecretApiKey), "DELETE", types.APIRequest{})
	log.Infof("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)

	respBody, statusCode = DoRequest(t, fmt.Sprintf("%s/v1/admin/accounts/%d",
		API.URL, integrator.ID), API.AuthToken, "DELETE", types.APIRequest{})
	log.Infof("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)
}
