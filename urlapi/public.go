package urlapi

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
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
		"/pub/organizations/{organizationId}/elections",
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
func (u *URLAPI) registerPublicKeyHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
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
	entityId, err := util.GetBytesID(ctx, "organizationId")
	if err != nil {
		return err
	}
	list, err :=
		u.getProcessList(ctx.URLParam("type"), []byte{}, entityId, false)
	if err != nil {
		return err
	}
	return sendResponse(list, ctx)
}

// GET https://server/v1/pub/elections/<processId>
// getProcessInfoPublicHandler gets public process info
func (u *URLAPI) getProcessInfoPublicHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	processId, err := util.GetBytesID(ctx, "electionId")
	if err != nil {
		return err
	}

	// Fetch process from vochain
	vochainProcess, err := u.vocClient.GetProcess(processId)
	if err != nil {
		return fmt.Errorf("unable to get process: %w", err)
	}

	// Fetch results
	var results *types.VochainResults
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			return fmt.Errorf("unable to get results %w", err)
		}
	}

	dbElection, err := u.db.GetElectionPublic(vochainProcess.EntityID, processId)
	if err != nil {
		return fmt.Errorf("could not fetch election %x from db: %w", processId, err)
	}

	if dbElection.Confidential {
		return fmt.Errorf("process %x is confidential, use authenticated API", processId)
	}

	// Fetch metadata
	processMetadata, err := u.vocClient.FetchProcessMetadata(vochainProcess.Metadata)
	if err != nil {
		return fmt.Errorf("unable to get metadata: %w", err)
	}

	// Parse all the information
	resp, err := u.parseProcessInfo(vochainProcess, results, processMetadata)
	if err != nil {
		return fmt.Errorf("could not parse information for process %x: %w", processId, err)
	}

	return sendResponse(resp, ctx)
}

// GET https://server/v1/pub/elections/<processId>/auth/<signature>
// getProcessInfoConfidentialHandler gets process info, including private metadata,
//  checking the voter's signature for inclusion in the census
func (u *URLAPI) getProcessInfoConfidentialHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	processId, err := util.GetBytesID(ctx, "electionId")
	if err != nil {
		return err
	}
	cspSignature, err := util.GetBytesID(ctx, "signature")
	if err != nil {
		return err
	}

	// Fetch process from vochain
	vochainProcess, err := u.vocClient.GetProcess(processId)
	if err != nil {
		return fmt.Errorf("unable to get process: %w", err)
	}

	// Fetch results
	var results *types.VochainResults
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			return fmt.Errorf("unable to get results %w", err)
		}
	}

	dbElection, err := u.db.GetElectionPublic(vochainProcess.EntityID, processId)
	if err != nil {
		return fmt.Errorf("could not fetch election %x from db: %w", processId, err)
	}
	integrator, err := u.db.GetIntegratorByKey(dbElection.IntegratorApiKey)
	if err != nil {
		return fmt.Errorf("could not fetch election's integrator from db: %w", err)
	}
	cspPubKey, err := ethereum.PubKeyFromSignature(processId, cspSignature)
	if err != nil {
		return fmt.Errorf("could not extract csp pubKey from signature: %w", err)
	}
	if !bytes.Equal(cspPubKey, integrator.CspPubKey) {
		return fmt.Errorf("signature pubKey %x does not match integrator's csp pubKey %x",
			cspPubKey, integrator.CspPubKey)
	}

	var processMetadata *types.ProcessMetadata
	if dbElection.Confidential {
		metaKey := dbElection.MetadataPrivKey
		// If globalMetadataKey exists, try to decrypt metadata private key
		if len(u.globalMetadataKey) > 0 {
			var ok bool
			metaKey, ok = util.DecryptSymmetric(dbElection.MetadataPrivKey, u.globalMetadataKey)
			if !ok {
				return fmt.Errorf("could not decrypt election private metadata key")
			}
		}
		if processMetadata, err = u.vocClient.FetchProcessMetadataConfidential(
			vochainProcess.Metadata, metaKey); err != nil {
			return fmt.Errorf("could not get process metadata: %w", err)
		}
	} else {
		if processMetadata, err = u.vocClient.FetchProcessMetadata(vochainProcess.Metadata); err != nil {
			return fmt.Errorf("could not get process metadata: %w", err)
		}
	}

	// Parse all the information
	resp, err := u.parseProcessInfo(vochainProcess, results, processMetadata)
	if err != nil {
		return fmt.Errorf("could not parse information for process %x: %w", processId, err)
	}

	return sendResponse(resp, ctx)
}

// GET https://server/v1/pub/account/organizations/<organizationId>
// getOrganizationHandler fetches an entity
func (u *URLAPI) getOrganizationHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	// authenticate integrator has permission to edit this entity
	orgInfo, err := u.authEntityPermissions(msg, ctx)
	if err != nil {
		return err
	}
	// Fetch process from vochain
	metaUri, _, _, err := u.vocClient.GetAccount(orgInfo.organization.EthAddress)
	if err != nil {
		return fmt.Errorf("unable to get account: %w", err)
	}

	// Fetch metadata
	organizationMetadata, err := u.vocClient.FetchOrganizationMetadata(metaUri)
	if err != nil {
		return fmt.Errorf("could not get organization metadata with URI\"%s\": %w", metaUri, err)
	}
	resp := types.APIResponse{
		Name:        organizationMetadata.Name["default"],
		Description: organizationMetadata.Description["default"],
		Avatar:      organizationMetadata.Media.Avatar,
		Header:      organizationMetadata.Media.Header,
	}
	return sendResponse(resp, ctx)
}

// POST https://server/v1/pub/elections/<processId>/vote
func (u *URLAPI) submitVotePublicHandler(
	msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	log.Debugf("query to submit vote for process %s", ctx.URLParam("electionId"))
	req, err := util.UnmarshalRequest(msg)
	if err != nil {
		return err
	}
	var votePkg []byte
	if votePkg, err = base64.StdEncoding.DecodeString(req.Vote); err != nil {
		return fmt.Errorf("could not decode vote pkg to base64: %w", err)
	}
	var resp types.APIResponse
	if resp.Nullifier, err = u.vocClient.RelayVote(votePkg); err != nil {
		return fmt.Errorf("could not submit vote tx: %w", err)
	}

	return sendResponse(resp, ctx)
}

// GET https://server/v1//pub/nullifiers/{nullifier}
// getVoteHandler returns a single vote envelope for the given nullifier
func (u *URLAPI) getVoteHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	nullifier, err := util.GetBytesID(ctx, "nullifier")
	if err != nil {
		return err
	}
	var resp types.APIResponse
	resp.Registered = new(bool)
	if resp.ProcessID, *resp.Registered, err = u.vocClient.GetVoteStatus(nullifier); err != nil {
		return fmt.Errorf("could not get envelope status for vote with nullifier %x: %w", nullifier, err)
	}
	if *resp.Registered {
		resp.ExplorerUrl = fmt.Sprintf("%s%x", u.config.ExplorerVoteUrl, nullifier)
	}
	return sendResponse(resp, ctx)
}
