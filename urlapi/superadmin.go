package urlapi

import (
	"encoding/hex"
	"fmt"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	dvoteTypes "go.vocdoni.io/dvote/types"
	dvoteUtil "go.vocdoni.io/dvote/util"
)

const (
	// TODO determine corred max requests for integrators
	INTEGRATOR_MAX_REQUESTS = 2 << 16
)

func (u *URLAPI) enableSuperadminHandlers(adminToken string) error {
	u.api.SetAdminToken(adminToken)
	log.Infof("admin token;: %s", adminToken)
	if err := u.api.RegisterMethod(
		"/admin/accounts",
		"POST",
		bearerstdapi.MethodAccessTypeAdmin,
		u.createIntegratorAccountHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/admin/accounts/{id}",
		"PUT",
		bearerstdapi.MethodAccessTypeAdmin,
		u.updateIntegratorAccountHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/admin/accounts/{id}/key",
		"PATCH",
		bearerstdapi.MethodAccessTypeAdmin,
		u.resetIntegratorKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/admin/accounts/{id}",
		"GET",
		bearerstdapi.MethodAccessTypeAdmin,
		u.getIntegratorAccountHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/admin/accounts/{id}",
		"DELETE",
		bearerstdapi.MethodAccessTypeAdmin,
		u.deleteIntegratorAccountHandler,
	); err != nil {
		return err
	}
	return nil
}

// POST https://server/v1/admin/accounts
// createIntegratorAccountHandler creates a new integrator account
func (u *URLAPI) createIntegratorAccountHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var apiKey dvoteTypes.HexBytes
	var req types.APIRequest
	var resp types.APIResponse
	if req, err = util.UnmarshalRequest(msg); err != nil {
		log.Error(err)
		return err
	}
	resp.APIKey = util.GenerateBearerToken()
	if apiKey, err = hex.DecodeString(resp.APIKey); err != nil {
		log.Errorf("error generating private key: %v", err)
		return fmt.Errorf("error generating private key: %v", err)
	}

	var cspPubKey dvoteTypes.HexBytes
	cspPubKey, err = hex.DecodeString(dvoteUtil.TrimHex(req.CspPubKey))
	if err != nil {
		log.Errorf("error devocding csp pub key: %v", err)
		return fmt.Errorf("error devocding csp pub key")
	}
	if resp.ID, err = u.db.CreateIntegrator(apiKey,
		cspPubKey, req.CspUrlPrefix, req.Name, req.Email); err != nil {
		log.Error(err)
		return err
	}
	resp.Ok = true
	return sendResponse(resp, ctx)
}

// PUT https://server/v1/admin/accounts/<id>
// updateIntegratorAccountHandler updates an existing integrator account
func (u *URLAPI) updateIntegratorAccountHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var req types.APIRequest
	var resp types.APIResponse
	var id int
	if id, err = util.GetIntID(ctx, "id"); err != nil {
		log.Error(err)
		return err
	}
	if req, err = util.UnmarshalRequest(msg); err != nil {
		log.Error(err)
		return err
	}

	var cspPubKey dvoteTypes.HexBytes
	cspPubKey, err = hex.DecodeString(dvoteUtil.TrimHex(req.CspPubKey))
	if err != nil {
		log.Errorf("error devocding csp pub key: %v", err)
		return fmt.Errorf("error devocding csp pub key")
	}
	if _, err = u.db.UpdateIntegrator(id, cspPubKey, req.Name, req.CspUrlPrefix); err != nil {
		log.Error(err)
		return err
	}
	resp.Ok = true
	return sendResponse(resp, ctx)
}

// PATCH https://server/v1/admin/accounts/<id>/key
// resetIntegratorKeyHandler resets an integrator api key
func (u *URLAPI) resetIntegratorKeyHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var apiKey dvoteTypes.HexBytes
	var resp types.APIResponse
	var id int
	if id, err = util.GetIntID(ctx, "id"); err != nil {
		log.Error(err)
		return err
	}

	// Now generate a new api key & update integrator
	resp.APIKey = util.GenerateBearerToken()
	if apiKey, err = hex.DecodeString(resp.APIKey); err != nil {
		log.Errorf("error generating private key: %v", err)
		return fmt.Errorf("error generating private key: %v", err)
	}
	if _, err = u.db.UpdateIntegratorApiKey(id, apiKey); err != nil {
		log.Error(err)
		return err
	}
	resp.Ok = true
	return sendResponse(resp, ctx)
}

// GET https://server/v1/admin/accounts/<id>
// getIntegratorAccountHandler fetches an integrator account
func (u *URLAPI) getIntegratorAccountHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var integrator *types.Integrator
	var id int
	if id, err = util.GetIntID(ctx, "id"); err != nil {
		log.Error(err)
		return err
	}
	if integrator, err = u.db.GetIntegrator(id); err != nil {
		log.Error(err)
		return err
	}
	resp.Name = integrator.Name
	resp.CspPubKey = integrator.CspPubKey
	resp.CspUrlPrefix = integrator.CspUrlPrefix
	resp.Ok = true
	return sendResponse(resp, ctx)
}

// DELETE https://server/v1/admin/accounts/<id>
// deleteIntegratorAccountHandler deletes an integrator account
func (u *URLAPI) deleteIntegratorAccountHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var id int
	if id, err = util.GetIntID(ctx, "id"); err != nil {
		log.Error(err)
		return err
	}
	if err = u.db.DeleteIntegrator(id); err != nil {
		log.Error(err)
		return err
	}
	resp.Ok = true
	return sendResponse(resp, ctx)
}
