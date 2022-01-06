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
	"go.vocdoni.io/dvote/log"
)

func TestOrganization(t *testing.T) {
	integrators := testcommon.CreateIntegrators(2)

	// create two integrators to test with
	for _, integrator := range integrators {
		req := types.APIRequest{
			CspUrlPrefix: integrator.CspUrlPrefix,
			CspPubKey:    hex.EncodeToString(integrator.CspPubKey),
			Name:         integrator.Name,
			Email:        integrator.Email,
		}
		respBody, statusCode := DoRequest(t, API.URL+"/v1/admin/accounts", API.AuthToken, "POST", req)
		if statusCode != 200 {
			log.Errorf("error response %s", string(respBody))
			t.FailNow()
		}
		var resp types.APIResponse
		err := json.Unmarshal(respBody, &resp)
		qt.Assert(t, err, qt.IsNil)
		qt.Check(t, resp.ID, qt.Not(qt.Equals), 0)
		qt.Check(t, len(resp.APIKey) > 0, qt.IsTrue)
		integrator.ID = resp.ID
		if integrator.SecretApiKey, err = hex.DecodeString(resp.APIKey); err != nil {
			log.Fatal(err)
		}
		log.Infof("%s", respBody)
	}

	// test create organization
	organization := testcommon.CreateOrganizations(1)[0]
	req := types.APIRequest{
		Name:        organization.Name,
		Description: organization.Description,
		Header:      organization.HeaderURI,
		Avatar:      organization.AvatarURI,
	}
	respBody, statusCode := DoRequest(t, API.URL+"/v1/priv/account/organizations",
		hex.EncodeToString(integrators[0].SecretApiKey), "POST", req)
	log.Infof("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)
	var resp types.APIResponse
	err := json.Unmarshal(respBody, &resp)
	qt.Assert(t, err, qt.IsNil)
	qt.Check(t, len(resp.APIToken) > 0, qt.IsTrue)
	qt.Check(t, len(resp.TxHash) > 0, qt.IsTrue)
	organization.ID = resp.ID
	organization.EthAddress = resp.OrganizationID
	organization.APIToken = resp.APIToken
	// save the txHash so we can run other tests and come back to organization creation
	organization.CreationTxHash = resp.TxHash

	// create organization failure: missing integrator token
	req = types.APIRequest{
		Name:        organization.Name,
		Description: organization.Description,
		Header:      organization.HeaderURI,
		Avatar:      organization.AvatarURI,
	}
	respBody, statusCode = DoRequest(t, API.URL+"/v1/priv/account/organizations",
		"1234", "POST", req)
	log.Infof("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 401)

	// create organization failure: empty name
	req = types.APIRequest{
		Description: organization.Description,
		Header:      organization.HeaderURI,
		Avatar:      organization.AvatarURI,
	}
	respBody, statusCode = DoRequest(t, API.URL+"/v1/priv/account/organizations",
		hex.EncodeToString(integrators[0].SecretApiKey), "POST", req)
	log.Infof("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 400)

	// create organization: check txHash has been mined
	var respMined urlapi.APIMined
	for numTries := 5; numTries > 0; numTries-- {
		if numTries != 5 {
			time.Sleep(time.Second * 4)
		}
		req = types.APIRequest{}
		respBody, statusCode = DoRequest(t, API.URL+
			"/v1/priv/transactions/"+organization.CreationTxHash,
			hex.EncodeToString(integrators[0].SecretApiKey), "GET", req)
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

	// now fetch the organization we created
	respBody, statusCode = DoRequest(t, API.URL+
		"/v1/priv/account/organizations/"+hex.EncodeToString(organization.EthAddress),
		hex.EncodeToString(integrators[0].SecretApiKey), "GET", types.APIRequest{})
	log.Infof("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)
	err = json.Unmarshal(respBody, &resp)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, len(resp.APIToken) > 0, qt.IsTrue)
	organization.APIToken = resp.APIToken
	qt.Assert(t, resp.Name, qt.Equals, organization.Name)
	qt.Assert(t, resp.Description, qt.Equals, organization.Description)
	qt.Assert(t, resp.Avatar, qt.Equals, organization.AvatarURI)
	qt.Assert(t, resp.Header, qt.Equals, organization.HeaderURI)

	// fail get organization: bad id
	respBody, statusCode = DoRequest(t, API.URL+
		"/v1/priv/account/organizations/"+"1234",
		hex.EncodeToString(integrators[0].SecretApiKey), "GET", types.APIRequest{})
	log.Infof("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 400)

	// fail get organization: bad id
	respBody, statusCode = DoRequest(t, API.URL+
		"/v1/priv/account/organizations/"+hex.EncodeToString(organization.EthAddress),
		hex.EncodeToString(integrators[1].SecretApiKey), "GET", types.APIRequest{})
	log.Infof("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 400)

	// reset the organization api token
	respBody, statusCode = DoRequest(t, API.URL+
		"/v1/priv/account/organizations/"+hex.EncodeToString(organization.EthAddress)+"/key",
		hex.EncodeToString(integrators[0].SecretApiKey), "PATCH", types.APIRequest{})
	log.Infof("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)
	err = json.Unmarshal(respBody, &resp)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, len(resp.APIToken) > 0, qt.IsTrue)
	qt.Assert(t, resp.APIToken, qt.Not(qt.Equals), organization.APIToken)

	// cleaning up
	respBody, statusCode = DoRequest(t, fmt.Sprintf("%s/v1/priv/account/organizations/"+
		hex.EncodeToString(organization.EthAddress), API.URL),
		hex.EncodeToString(integrators[0].SecretApiKey), "DELETE", types.APIRequest{})
	log.Infof("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)

	// fail get organization: should be deleted
	respBody, statusCode = DoRequest(t, API.URL+
		"/v1/priv/account/organizations/"+hex.EncodeToString(organization.EthAddress),
		hex.EncodeToString(integrators[0].SecretApiKey), "GET", types.APIRequest{})
	log.Infof("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 400)

	for _, integrator := range integrators {
		respBody, statusCode := DoRequest(t, fmt.Sprintf("%s/v1/admin/accounts/%d",
			API.URL, integrator.ID), API.AuthToken, "DELETE", types.APIRequest{})
		log.Infof("%s", respBody)
		qt.Assert(t, statusCode, qt.Equals, 200)
	}
}
