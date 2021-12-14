package urlapi

import (
	"encoding/base64"
	"fmt"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/vochain/scrutinizer/indexertypes"
)

func (u *URLAPI) enablePublicHandlers() error {
	if err := u.api.RegisterMethod(
		"/pub/censuses/{censusId}/token",
		"POST",
		bearerstdapi.MethodAccessTypePublic,
		u.registerPublicKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/organizations/{organizationId}/elections/{type}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.listProcessesHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/elections/{electionId}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.getProcessInfoPublicHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/elections/{electionId}/vote",
		"POST",
		bearerstdapi.MethodAccessTypePublic,
		u.submitVotePublicHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/elections/{electionId}/auth/{signature}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.getProcessInfoConfidentialHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/account/organizations/{organizationId}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.getOrganizationHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/nullifiers/{nullifier}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.getVoteHandler,
	); err != nil {
		return err
	}
	return nil
}

// POST https://server/v1/pub/censuses/<censusId>/token
// registerPublicKeyHandler registers a voter's public key with a census token
func (u *URLAPI) registerPublicKeyHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/pub/organizations/<organizationId>/elections/signed
// GET https://server/v1/pub/organizations/<organizationId>/elections/blind
// GET https://server/v1/pub/organizations/<organizationId>/elections/active
// GET https://server/v1/pub/organizations/<organizationId>/elections/ended
// GET https://server/v1/pub/organizations/<organizationId>/elections/upcoming
// listProcessesHandler' lists signed, blind, active, ended, or upcoming processes
func (u *URLAPI) listProcessesHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var entityId []byte
	var err error
	var resp types.APIResponse

	if entityId, err = util.GetBytesID(ctx, "organizationId"); err != nil {
		return err
	}

	filter := ctx.URLParam("type")
	if resp.PrivateProcesses, resp.PublicProcesses, err = u.getProcessList(filter,
		[]byte{}, entityId, false); err != nil {
		return err
	}
	return sendResponse(resp, ctx)
}

// GET https://server/v1/pub/elections/<processId>
// getProcessInfoPublicHandler gets public process info
func (u *URLAPI) getProcessInfoPublicHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIElectionInfo
	var processId []byte
	var vochainProcess *indexertypes.Process
	var results *types.VochainResults
	var processMetadata *types.ProcessMetadata
	if processId, err = util.GetBytesID(ctx, "electionId"); err != nil {
		return err
	}

	// Fetch process from vochain
	if vochainProcess, err = u.vocClient.GetProcess(processId); err != nil {
		return fmt.Errorf("unable to get process: %w", err)
	}

	// Fetch results
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			return fmt.Errorf("unable to get results %w", err)
		}
	}

	// Fetch metadata
	metadataUri := vochainProcess.Metadata
	if processMetadata, err = u.vocClient.FetchProcessMetadata(metadataUri); err != nil {
		return fmt.Errorf("unable to get metadata: %w", err)
	}

	// Parse all the information
	if resp, err = u.parseProcessInfo(vochainProcess, results, processMetadata); err != nil {
		return fmt.Errorf("could not parse information for process %x: %w", processId, err)
	}

	return sendResponse(resp, ctx)
}

// GET https://server/v1/pub/elections/<processId>/auth/<signature>
// getProcessInfoConfidentialHandler gets process info, including private metadata,
//  checking the voter's signature for inclusion in the census
func (u *URLAPI) getProcessInfoConfidentialHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/pub/account/organizations/<organizationId>
// getOrganizationHandler fetches an entity
func (u *URLAPI) getOrganizationHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var orgInfo orgPermissionsInfo
	var organizationMetadata *types.EntityMetadata
	var metaUri string
	// authenticate integrator has permission to edit this entity
	if orgInfo, err = u.authEntityPermissions(msg, ctx); err != nil {
		return err
	}
	// Fetch process from vochain
	if metaUri, _, _, err = u.vocClient.GetAccount(orgInfo.organization.EthAddress); err != nil {
		return fmt.Errorf("unable to get account: %w", err)
	}

	// Fetch metadata
	if organizationMetadata, err = u.vocClient.FetchOrganizationMetadata(metaUri); err != nil {
		return fmt.Errorf("could not get organization metadata with URI\"%s\": %w", metaUri, err)
	}

	resp.Name = organizationMetadata.Name["default"]
	resp.Description = organizationMetadata.Description["default"]
	resp.Avatar = organizationMetadata.Media.Avatar
	resp.Header = organizationMetadata.Media.Header
	return sendResponse(resp, ctx)
}

// POST https://server/v1/pub/elections/<processId>/vote
func (u *URLAPI) submitVotePublicHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var req types.APIRequest
	log.Debugf("query to submit vote for process %s", ctx.URLParam("electionId"))
	if req, err = util.UnmarshalRequest(msg); err != nil {
		return err
	}
	var votePkg []byte
	if votePkg, err = base64.StdEncoding.DecodeString(req.Vote); err != nil {
		return fmt.Errorf("could not decode vote pkg to base64: %w", err)
	}
	if resp.Nullifier, err = u.vocClient.RelayVote(votePkg); err != nil {
		return fmt.Errorf("could not submit vote tx: %w", err)
	}

	return sendResponse(resp, ctx)
}

// GET https://server/v1//pub/nullifiers/{nullifier}
// getVoteHandler returns a single vote envelope for the given nullifier
func (u *URLAPI) getVoteHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var nullifier []byte
	var err error
	var resp types.APIResponse

	if nullifier, err = util.GetBytesID(ctx, "nullifier"); err != nil {
		return err
	}
	if resp.ProcessID, resp.Registered, err = u.vocClient.GetVoteStatus(nullifier); err != nil {
		return fmt.Errorf("could not get envelope status for vote with nullifier %x: %w", nullifier, err)
	}
	if resp.Registered {
		resp.ExplorerUrl = fmt.Sprintf("%s%x", u.config.ExplorerVoteUrl, nullifier)
	}
	return sendResponse(resp, ctx)
}
