package urlapi

import (
	"fmt"

	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
)

func (u *URLAPI) enableIntegratorHandlers() error {
	if err := u.api.RegisterMethod(
		"/priv/account/entities",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.createEntityHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/entities/{entityId}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.getEntityHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/entities/{entityId}",
		"DELETE",
		bearerstdapi.MethodAccessTypePrivate,
		u.deleteEntityHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/entities/{id}/key",
		"PATCH",
		bearerstdapi.MethodAccessTypePrivate,
		u.resetEntityKeyHandler,
	); err != nil {
		return err
	}
	return nil
}

// POST https://server/v1/priv/account/entities
// createEntityHandler creates a new entity
func (u *URLAPI) createEntityHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {

	// check if token exists as a valid integrator token

	req, err := util.UnmarshalRequest(ctx)
	if err != nil {
		return fmt.Errorf("could not decode request body: %v", err)
	}
	organizationID, err := util.GetOrganizationID(ctx)
	if err != nil {
		return fmt.Errorf("could not retrieve EntityID: %v", err)
	}
	log.Debugf("organization %x", organizationID)
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/priv/account/entities/<entityId>
// getEntityHandler fetches an entity
func (u *URLAPI) getEntityHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// DELETE https://server/v1/priv/account/entities/<entityId>
// deleteEntityHandler deletes an entity
func (u *URLAPI) deleteEntityHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// PATCH https://server/v1/account/entities/<id>/key
// resetEntityKeyHandler resets an entity's api key
func (u *URLAPI) resetEntityKeyHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}
