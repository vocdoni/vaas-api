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

func TestOrganization(t *testing.T) {
	t.Parallel()
	// test create organization
	organization := testcommon.CreateOrganizations(1)[0]
	req := types.APIRequest{
		Name:        organization.Name,
		Description: organization.Description,
		Header:      organization.HeaderURI,
		Avatar:      organization.AvatarURI,
	}
	respBody, statusCode := DoRequest(t,
		fmt.Sprintf("%s/v1/priv/account/organizations", API.URL),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "POST", req)
	t.Logf("%s", respBody)
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

	// create organization: check txHash has been mined
	var respMined urlapi.APIMined
	for numTries := 5; numTries > 0; numTries-- {
		if numTries != 5 {
			time.Sleep(time.Second * 4)
		}
		req = types.APIRequest{}
		respBody, statusCode = DoRequest(t,
			fmt.Sprintf("%s/v1/priv/transactions/%s", API.URL, organization.CreationTxHash),
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

	// now fetch the organization we created
	respBody, statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/account/organizations/%x", API.URL, organization.EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{})
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)
	err = json.Unmarshal(respBody, &resp)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, len(resp.APIToken) > 0, qt.IsTrue)
	organization.APIToken = resp.APIToken
	qt.Assert(t, resp.Name, qt.Equals, organization.Name)
	qt.Assert(t, resp.Description, qt.Equals, organization.Description)
	qt.Assert(t, resp.Avatar, qt.Equals, organization.AvatarURI)
	qt.Assert(t, resp.Header, qt.Equals, organization.HeaderURI)

	// cleaning up
	respBody, statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/account/organizations/%x", API.URL, organization.EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "DELETE", types.APIRequest{})
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)

	// fail get organization: should be deleted
	respBody, statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/account/organizations/%x", API.URL, organization.EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{})
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 400)
}

func TestCreateOrganizationFailure(t *testing.T) {
	t.Parallel()
	organization := testcommon.CreateOrganizations(1)[0]
	// create organization failure: missing integrator token
	req := types.APIRequest{
		Name:        organization.Name,
		Description: organization.Description,
		Header:      organization.HeaderURI,
		Avatar:      organization.AvatarURI,
	}
	respBody, statusCode := DoRequest(t,
		fmt.Sprintf("%s/v1/priv/account/organizations", API.URL),
		"1234", "POST", req)
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 401)

	// create organization failure: empty name
	req = types.APIRequest{
		Description: organization.Description,
		Header:      organization.HeaderURI,
		Avatar:      organization.AvatarURI,
	}
	respBody, statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/account/organizations", API.URL),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "POST", req)
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 400)
}

func TestGetOrganizationFailure(t *testing.T) {
	t.Parallel()
	// fail get organization: bad id
	respBody, statusCode := DoRequest(t,
		fmt.Sprintf("%s/v1/priv/account/organizations/%s", API.URL, "1234"),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "GET", types.APIRequest{})
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 400)

	// fail get organization: bad api key
	respBody, statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/priv/account/organizations/%x", API.URL, testOrganizations[0].EthAddress),
		hex.EncodeToString(testIntegrators[1].SecretApiKey), "GET", types.APIRequest{})
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 400)

}

func TestResetAPIToken(t *testing.T) {
	t.Parallel()
	// reset the organization api token
	respBody, statusCode := DoRequest(t,
		fmt.Sprintf("%s/v1/priv/account/organizations/%x/key", API.URL, testOrganizations[0].EthAddress),
		hex.EncodeToString(testIntegrators[0].SecretApiKey), "PATCH", types.APIRequest{})
	t.Logf("%s", respBody)
	qt.Assert(t, statusCode, qt.Equals, 200)
	var resp types.APIResponse
	err := json.Unmarshal(respBody, &resp)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, len(resp.APIToken) > 0, qt.IsTrue)
	qt.Assert(t, resp.APIToken, qt.Not(qt.Equals), testOrganizations[0].APIToken)
}
