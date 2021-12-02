package urlapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
)

func (u *URLAPI) enableEntityHandlers() error {
	if err := u.api.RegisterMethod(
		"/priv/account/entities",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.createOrganizationHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/entities/{entityId}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.getOrganizationHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/entities/{entityId}",
		"DELETE",
		bearerstdapi.MethodAccessTypePrivate,
		u.deleteOrganizationHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/entities/{id}/key",
		"PATCH",
		bearerstdapi.MethodAccessTypePrivate,
		u.resetOrganizationKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/entities/{entityId}/metadata",
		"PUT",
		bearerstdapi.MethodAccessTypePrivate,
		u.setEntityMetadataHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/entities/{entityId}/processes/*",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.createProcessHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/entities/{entityId}/processes/*",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.listProcessesHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/processes/{processId}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.getProcessHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.createCensusHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/{censusId}/tokens/*",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.addCensusTokensHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/{censusId}/tokens/{tokenId}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.getCensusTokenHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/{censusId}/tokens/{tokenId}",
		"DELETE",
		bearerstdapi.MethodAccessTypePrivate,
		u.deleteCensusTokenHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/{censusId}/tokens/{tokenId}",
		"DELETE",
		bearerstdapi.MethodAccessTypePrivate,
		u.deleteCensusTokenHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/{censusId}/keys/{publicKey}",
		"DELETE",
		bearerstdapi.MethodAccessTypePrivate,
		u.deletePublicKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/censuses/{censusId}/import/*",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.importPublicKeysHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/processes/{processId}/status",
		"PUT",
		bearerstdapi.MethodAccessTypePrivate,
		u.setProcessStatusHandler,
	); err != nil {
		return err
	}
	return nil
}

// POST https://server/v1/priv/account/entities
// createOrganizationHandler creates a new entity
func (u *URLAPI) createOrganizationHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var req types.APIRequest
	var integratorPrivKey []byte
	var orgApiToken string
	// var organizationMetadataKey []byte
	if req, err = util.UnmarshalRequest(ctx); err != nil {
		return err
	}
	if integratorPrivKey, err = util.GetAuthToken(msg); err != nil {
		return err
	}
	orgApiToken = util.GenerateBearerToken()
	// TODO create Vochain account once gateway API is available
	// TODO encrypt private key
	ethSignKeys := ethereum.NewSignKeys()
	if _, err = u.db.CreateOrganization(integratorPrivKey, ethSignKeys.Address().Bytes(),
		[]byte{}, 0, 0, orgApiToken, req.Header, req.Avatar); err != nil {
		return fmt.Errorf("could not create organization: %v", err)
	}
	// TODO use correct apiQuota to register token
	u.registerToken(orgApiToken, 0)

	resp.APIKey = orgApiToken
	resp.EntityID = ethSignKeys.Address().Bytes()
	return sendResponse(resp, ctx)
}

// GET https://server/v1/priv/account/entities/<entityId>
// getOrganizationHandler fetches an entity
func (u *URLAPI) getOrganizationHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var organization *types.Organization
	// authenticate integrator has permission to edit this entity
	if _, _, organization, err = u.authEntityPermissions(msg, ctx); err != nil {
		return err
	}

	// TODO get metadata if needed

	resp.APIKey = organization.PublicAPIToken
	// resp.Name = organization.Name
	resp.Avatar = organization.AvatarURI
	resp.Header = organization.HeaderURI
	return sendResponse(resp, ctx)
}

// DELETE https://server/v1/priv/account/entities/<entityId>
// deleteOrganizationHandler deletes an entity
func (u *URLAPI) deleteOrganizationHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var organization *types.Organization
	var integratorPrivKey []byte
	var entityID []byte
	// authenticate integrator has permission to edit this entity
	if integratorPrivKey, entityID, organization, err = u.authEntityPermissions(msg, ctx); err != nil {
		return err
	}

	if err = u.db.DeleteOrganization(integratorPrivKey, entityID); err != nil {
		return err
	}
	u.revokeToken(organization.PublicAPIToken)
	return sendResponse(resp, ctx)
}

// PATCH https://server/v1/account/entities/<id>/key
// resetOrganizationKeyHandler resets an entity's api key
func (u *URLAPI) resetOrganizationKeyHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var integratorPrivKey []byte
	var entityID []byte
	var oldOrganization *types.Organization
	// authenticate integrator has permission to edit this entity
	if integratorPrivKey, entityID, oldOrganization,
		err = u.authEntityPermissions(msg, ctx); err != nil {
		return err
	}
	u.revokeToken(oldOrganization.PublicAPIToken)

	// Now generate a new api key & update integrator
	resp.APIKey = util.GenerateBearerToken()
	if _, err = u.db.UpdateOrganizationPublicAPIToken(
		integratorPrivKey, entityID, resp.APIKey); err != nil {
		return err
	}
	u.registerToken(resp.APIKey, int64(oldOrganization.PublicAPIQuota))
	return sendResponse(resp, ctx)
}

// PUT https://server/v1/priv/entities/<entityId>/metadata
// setEntityMetadataHandler sets an entity's metadata
func (u *URLAPI) setEntityMetadataHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var req types.APIRequest
	var organization *types.Organization
	var entityID []byte
	var metaBytes []byte
	var metaURI string
	var entityMetadata types.EntityMetadata

	// authenticate integrator has permission to edit this entity
	if _, entityID, organization, err = u.authEntityPermissions(msg, ctx); err != nil {
		return err
	}

	entityMetadata.Avatar = req.Avatar
	entityMetadata.Description = req.Description
	entityMetadata.Header = req.Header
	entityMetadata.Name = req.Name

	if metaBytes, err = json.Marshal(entityMetadata); err != nil {
		return fmt.Errorf("could not marshal entity metadata: %v", err)
	}
	if metaURI, err = u.vocClient.AddFile(metaBytes, "ipfs",
		fmt.Sprintf("%X entity metadata", entityID)); err != nil {
		return fmt.Errorf("could not post metadata to ipfs: %v", err)
	}

	// Update organization in the db to make sure it matches the metadata
	u.db.UpdateOrganization(organization.IntegratorApiKey, organization.EthAddress,
		organization.QuotaPlanID, organization.PublicAPIQuota, req.Header, req.Avatar)

	// TODO update the entity on the Vochain to reflect the new IPFS uri

	resp.EntityID = entityID
	resp.ContentURI = metaURI
	return sendResponse(resp, ctx)
}

// POST https://server/v1/priv/entities/<entityId>/processes/signed
// POST https://server/v1/priv/entities/<entityId>/processes/blind
// createProcessHandler creates a process with the given metadata, either with signed or blind signature voting
func (u *URLAPI) createProcessHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var req types.APIRequest
	var blind bool
	var entityID []byte
	var processID []byte
	var integratorPrivKey []byte

	if strings.HasSuffix(ctx.Request.URL.Path, "signed") {
		blind = false
	} else if strings.HasSuffix(ctx.Request.URL.Path, "blind") {
		blind = true
	} else {
		return fmt.Errorf("%s not a valid request path", ctx.Request.URL.Path)
	}

	// authenticate integrator has permission to edit this entity
	if integratorPrivKey, entityID, _, err = u.authEntityPermissions(msg, ctx); err != nil {
		return err
	}

	if req, err = util.UnmarshalRequest(ctx); err != nil {
		return err
	}

	// TODO create election on the vochain

	u.db.CreateElection(integratorPrivKey, entityID, []byte{}, req.Title, req.Census, big.Int{}, big.Int{}, req.Confidential, req.HiddenResults)

	resp.ProcessID = processID
	return sendResponse(resp, ctx)
}

// GET https://server/v1/priv/entities/<entityId>/processes/signed
// GET https://server/v1/priv/entities/<entityId>/processes/blind
// GET https://server/v1/priv/entities/<entityId>/processes/active
// GET https://server/v1/priv/entities/<entityId>/processes/ended
// GET https://server/v1/priv/entities/<entityId>/processes/upcoming
// listProcessesHandler lists signed, blind, active, ended, or upcoming processes
func (u *URLAPI) listProcessesHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/priv/processes/<processId>
// getProcessHandler gets the entirety of a process, including metadata
// confidential processes need no extra step, only the api key
func (u *URLAPI) getProcessHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// POST https://server/v1/priv/censuses
// createCensusHandler creates a census where public keys or token slots (that will eventually contain a public key) are stored.
// A census can start with 0 items, and public keys can be imported later on.
// If census tokens are allocated, users will need to generate a wallet on the frontend and register the public key by themselves.
// This prevents both the API and the integrator from gaining access to the private key.
func (u *URLAPI) createCensusHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// POST https://server/v1/priv/censuses/<censusId>/tokens/flat
// POST https://server/v1/priv/censuses/<censusId>/tokens/weighted
// addCensusTokensHandler adds N (weight 1 or weighted) census tokens for voters to register their public keys
func (u *URLAPI) addCensusTokensHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/priv/censuses/<censusId>/tokens/<tokenId>
// getCensusTokenHandler gets the given census token with weight and assigned public key, if applicable
func (u *URLAPI) getCensusTokenHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// DELETE https://server/v1/priv/censuses/<censusId>/tokens/<tokenId>
// deleteCensusTokenHandler deletes the given token(s) from the given census
func (u *URLAPI) deleteCensusTokenHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// DELETE https://server/v1/priv/censuses/<censusId>/keys/<publicKey>
// deletePublicKeyHandler deletes the given public key(s) from the given census
func (u *URLAPI) deletePublicKeyHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// POST https://server/v1/priv/censuses/<censusId>/import/flat
// POST https://server/v1/priv/censuses/<censusId>/import/weighted
// importPublicKeysHandler imports a group of public keys into the existing census, weighted or weight 1
func (u *URLAPI) importPublicKeysHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// PUT https://server/v1/priv/processes/<processId>/status
// setProcessStatusHandler sets the process status (READY, PAUSED, ENDED, CANCELED)
func (u *URLAPI) setProcessStatusHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

func (u *URLAPI) authEntityPermissions(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) ([]byte, []byte, *types.Organization, error) {
	var err error
	var entityID []byte
	var integratorPrivKey []byte
	var organization *types.Organization

	if integratorPrivKey, err = util.GetAuthToken(msg); err != nil {
		return nil, nil, nil, err
	}
	if entityID, err = util.GetBytesID(ctx); err != nil {
		return nil, nil, nil, err
	}
	if organization, err = u.db.GetOrganization(integratorPrivKey, entityID); err != nil {
		return nil, nil, nil, fmt.Errorf("entity %X could not be fetched from the db", entityID)
	}
	if !bytes.Equal(organization.IntegratorApiKey, integratorPrivKey) {
		return nil, nil, nil, fmt.Errorf("entity %X does not belong to this integrator", entityID)
	}
	return integratorPrivKey, entityID, organization, nil
}
