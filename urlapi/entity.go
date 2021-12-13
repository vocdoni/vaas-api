package urlapi

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	dvoteutil "go.vocdoni.io/dvote/util"
	"go.vocdoni.io/dvote/vochain/scrutinizer/indexertypes"
	"go.vocdoni.io/proto/build/go/models"
)

func (u *URLAPI) enableEntityHandlers() error {
	if err := u.api.RegisterMethod(
		"/priv/account/organizations",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.createOrganizationHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/organizations/{organizationId}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.getOrganizationPrivateHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/organizations/{organizationId}",
		"DELETE",
		bearerstdapi.MethodAccessTypePrivate,
		u.deleteOrganizationHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/organizations/{organizationId}/key",
		"PATCH",
		bearerstdapi.MethodAccessTypePrivate,
		u.resetOrganizationKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/organizations/{organizationId}/metadata",
		"PUT",
		bearerstdapi.MethodAccessTypePrivate,
		u.setOrganizationMetadataHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/organizations/{organizationId}/elections/{type}",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.createProcessHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/organizations/{entityId}/elections/{type}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.listProcessesPrivateHandler,
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
		"/priv/elections/{electionId}/status",
		"PUT",
		bearerstdapi.MethodAccessTypePrivate,
		u.setProcessStatusHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/elections/{electionId}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.getProcessHandler,
	); err != nil {
		return err
	}
	return nil
}

// POST https://server/v1/priv/account/organizations
// createOrganizationHandler creates a new entity
func (u *URLAPI) createOrganizationHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var req types.APIRequest
	var entityPrivKey []byte
	var integratorPrivKey []byte
	var orgApiToken string
	var encryptedPrivKey []byte
	var metaURI string
	// var organizationMetadataKey []byte
	if req, err = util.UnmarshalRequest(msg); err != nil {
		return fmt.Errorf("createOrganizationHandler: %w", err)
	}
	if integratorPrivKey, err = util.GetAuthToken(msg); err != nil {
		return fmt.Errorf("createOrganizationHandler: %w", err)
	}
	orgApiToken = util.GenerateBearerToken()

	ethSignKeys := ethereum.NewSignKeys()
	if err = ethSignKeys.Generate(); err != nil {
		return fmt.Errorf("createOrganizationHandler: could not generate ethereum keys: %w", err)
	}

	// Encrypt private key to store in db
	_, priv := ethSignKeys.HexString()
	if entityPrivKey, err = hex.DecodeString(priv); err != nil {
		return fmt.Errorf("createOrganizationHandler: could not decode entity private key: %w", err)
	}

	if encryptedPrivKey, err = util.EncryptSymmetric(entityPrivKey, integratorPrivKey); err != nil {
		return fmt.Errorf("createOrganizationHandler: could not encrypt entity private key: %w", err)
	}

	// Post metadata to ipfs
	if metaURI, err = u.vocClient.SetEntityMetadata(types.EntityMetadata{
		Version:     "1.0",
		Languages:   []string{},
		Name:        map[string]string{"default": req.Name},
		Description: map[string]string{"default": req.Description},
		NewsFeed:    map[string]string{},
		Media: types.EntityMedia{
			Avatar: req.Avatar,
			Header: req.Header,
		},
	}, ethSignKeys.Address().Bytes()); err != nil {
		return fmt.Errorf("createOrganizationHandler: could not set entity metadata: %w", err)
	}

	// Register organization to database
	if _, err = u.db.CreateOrganization(integratorPrivKey, ethSignKeys.Address().Bytes(),
		encryptedPrivKey, uuid.NullUUID{}, 0, orgApiToken, req.Header, req.Avatar); err != nil {
		return fmt.Errorf("createOrganizationHandler: could not create organization: %w", err)
	}

	// Create the new account on the Vochain
	if err = u.vocClient.SetAccountInfo(ethSignKeys, metaURI); err != nil {
		return fmt.Errorf("createOrganizationHandler: could not create account on the vochain: %w", err)
	}

	resp.APIToken = orgApiToken
	resp.OrganizationID = ethSignKeys.Address().Bytes()

	return sendResponse(resp, ctx)
}

// GET https://server/v1/priv/account/organizations/<organizationId>
// getOrganizationPrivateHandler fetches an entity
func (u *URLAPI) getOrganizationPrivateHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var organization *types.Organization
	var organizationMetadata *types.EntityMetadata
	var metaUri string
	// authenticate integrator has permission to edit this entity
	if _, _, organization, err = u.authEntityPermissions(msg, ctx); err != nil {
		return fmt.Errorf("getOrganizationPrivateHandler: %w", err)
	}

	// Fetch process from vochain
	if metaUri, _, _, err = u.vocClient.GetAccount(organization.EthAddress); err != nil {
		return fmt.Errorf("getOrganizationPrivateHandler: %w", err)
	}

	// Fetch metadata
	if organizationMetadata, err = u.vocClient.FetchOrganizationMetadata(metaUri); err != nil {
		return fmt.Errorf("getOrganizationPrivateHandler: could not get organization metadata with URI\"%s\": %w", metaUri, err)
	}

	resp.APIToken = organization.PublicAPIToken
	resp.Name = organizationMetadata.Name["default"]
	resp.Description = organizationMetadata.Description["default"]
	resp.Avatar = organizationMetadata.Media.Avatar
	resp.Header = organizationMetadata.Media.Header
	return sendResponse(resp, ctx)
}

// DELETE https://server/v1/priv/account/organizations/<organizationId>
// deleteOrganizationHandler deletes an entity
func (u *URLAPI) deleteOrganizationHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var integratorPrivKey []byte
	var entityID []byte
	// authenticate integrator has permission to edit this entity
	if integratorPrivKey, entityID, _, err = u.authEntityPermissions(msg, ctx); err != nil {
		log.Warn(err)
		return sendResponse(resp, ctx)
	}

	if err = u.db.DeleteOrganization(integratorPrivKey, entityID); err != nil {
		log.Warn(err)
		return sendResponse(resp, ctx)
	}
	return sendResponse(resp, ctx)
}

// PATCH https://server/v1/account/organizations/<id>/key
// resetOrganizationKeyHandler resets an entity's api key
func (u *URLAPI) resetOrganizationKeyHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var integratorPrivKey []byte
	var entityID []byte
	// authenticate integrator has permission to edit this entity
	if integratorPrivKey, entityID, _,
		err = u.authEntityPermissions(msg, ctx); err != nil {
		return fmt.Errorf("resetOrganizationKeyHandler: %w", err)
	}

	// Now generate a new api key & update integrator
	resp.APIToken = util.GenerateBearerToken()
	if _, err = u.db.UpdateOrganizationPublicAPIToken(
		integratorPrivKey, entityID, resp.APIToken); err != nil {
		return fmt.Errorf("resetOrganizationKeyHandler: could not update public api token %w", err)
	}
	return sendResponse(resp, ctx)
}

// PUT https://server/v1/priv/organizations/<organizationId>/metadata
// setOrganizationMetadataHandler sets an entity's metadata
func (u *URLAPI) setOrganizationMetadataHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var req types.APIRequest
	var organization *types.Organization
	var entityID []byte
	var metaURI string

	// authenticate integrator has permission to edit this entity
	if _, entityID, organization, err = u.authEntityPermissions(msg, ctx); err != nil {
		return fmt.Errorf("setOrganizationMetadataHandler: %w", err)
	}

	// Post metadata to ipfs
	if metaURI, err = u.vocClient.SetEntityMetadata(types.EntityMetadata{
		Version:     "1.0",
		Languages:   []string{},
		Name:        map[string]string{"default": req.Name},
		Description: map[string]string{"default": req.Description},
		NewsFeed:    map[string]string{},
		Media: types.EntityMedia{
			Avatar: req.Avatar,
			Header: req.Header,
		},
	}, entityID); err != nil {
		return fmt.Errorf("setOrganizationMetadataHandler: could not set entity metadata: %w", err)
	}

	// Update organization in the db to make sure it matches the metadata
	if _, err = u.db.UpdateOrganization(organization.IntegratorApiKey, organization.EthAddress,
		req.Header, req.Avatar); err != nil {
		return fmt.Errorf("setOrganizationMetadataHandler: could not update organization: %w", err)
	}

	resp.OrganizationID = entityID
	resp.ContentURI = metaURI
	return sendResponse(resp, ctx)
}

// POST https://server/v1/priv/organizations/<organizationId>/elections/signed
// POST https://server/v1/priv/organizations/<organizationId>/elections/blind
// createProcessHandler creates a process with the given metadata, either with signed or blind signature voting
func (u *URLAPI) createProcessHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIResponse
	var req types.APIRequest
	var organization *types.Organization
	// var blind bool
	var entityID []byte
	var processID []byte
	var integratorPrivKey []byte
	var metaUri string

	// TODO use blind/signed

	// ctx.URLParam("type")

	// authenticate integrator has permission to edit this entity
	if integratorPrivKey, entityID, organization, err = u.authEntityPermissions(msg, ctx); err != nil {
		return fmt.Errorf("createProcessHandler: %w", err)
	}

	if req, err = util.UnmarshalRequest(msg); err != nil {
		return fmt.Errorf("createProcessHandler: %w", err)
	}

	if req.Confidential {
		return fmt.Errorf("createProcessHandler: confidential processes are not yet supported")
	}

	pid := dvoteutil.RandomHex(32)
	if processID, err = hex.DecodeString(pid); err != nil {
		return fmt.Errorf("createProcessHandler: could not decode process ID: %w", err)
	}
	entityPrivKey, ok := util.DecryptSymmetric(organization.EthPrivKeyCicpher, integratorPrivKey)
	if !ok {
		return fmt.Errorf("createProcessHandler: could not decrypt entity private key")
	}
	entitySignKeys := ethereum.NewSignKeys()
	if err = entitySignKeys.AddHexKey(hex.EncodeToString(entityPrivKey)); err != nil {
		return fmt.Errorf("createProcessHandler: could not decode entity private key: %w", err)
	}

	startDate, err := time.Parse("2006-01-02T15:04:05.000Z", req.StartDate)
	if err != nil {
		return fmt.Errorf("createProcessHandler: could not parse startDate: %w", err)
	}
	endDate, err := time.Parse("2006-01-02T15:04:05.000Z", req.EndDate)
	if err != nil {
		return fmt.Errorf("createProcessHandler: could not parse startDate: %w", err)
	}

	now := time.Now()
	if startDate.Before(now) || endDate.Before(now) {
		return fmt.Errorf("createProcessHandler: election start and end date cannot be in the past")
	}
	if endDate.Before(startDate) {
		return fmt.Errorf("createProcessHandler: end date must be after start date")
	}
	startBlock, err := u.estimateBlockHeight(startDate)
	if err != nil {
		return fmt.Errorf("createProcessHandler: unable to estimate startDate block height: %w", err)
	}
	endBlock, err := u.estimateBlockHeight(endDate)
	if err != nil {
		return fmt.Errorf("createProcessHandler: unable to estimate endDate block height: %w", err)
	}
	log.Debugf("start block %d end block %d", startBlock, endBlock)

	metadata := types.ProcessMetadata{
		Description: map[string]string{"default": req.Description},
		Media: types.ProcessMedia{
			Header:    req.Header,
			StreamURI: req.StreamURI,
		},
		Meta:      metaUri,
		Questions: []types.QuestionMeta{},
		Results: types.ProcessResultsDetails{
			Aggregation: "discrete-values",
			Display:     "multiple-choice",
		},
		Title:   map[string]string{"default": req.Title},
		Version: "1.0",
	}

	envelopeType := &models.EnvelopeType{
		Serial:         false,
		Anonymous:      false,
		EncryptedVotes: req.HiddenResults,
		UniqueValues:   false,
		CostFromWeight: false,
	}
	processMode := &models.ProcessMode{
		AutoStart:         false,
		Interruptible:     true,
		DynamicCensus:     false,
		EncryptedMetaData: req.Confidential,
		PreRegister:       false,
	}

	maxChoiceValue := 0
	for _, question := range req.Questions {
		if len(question.Choices) > maxChoiceValue {
			maxChoiceValue = len(question.Choices)
		}
		metaQuestion := types.QuestionMeta{
			Choices:     []types.Choice{},
			Description: map[string]string{"default": question.Description},
			Title:       map[string]string{"default": question.Title},
		}
		for i, choice := range question.Choices {
			metaQuestion.Choices = append(metaQuestion.Choices, types.Choice{
				Title: map[string]string{"default": choice},
				Value: uint32(i),
			})
		}
		metadata.Questions = append(metadata.Questions, metaQuestion)
	}
	log.Debugf("req questions: %v", req.Questions)
	log.Debugf("meta qustions: %v", metadata.Questions)

	voteOptions := &models.ProcessVoteOptions{
		MaxCount:          uint32(len(req.Questions)),
		MaxValue:          uint32(maxChoiceValue),
		MaxVoteOverwrites: 0,
		MaxTotalCost:      uint32(len(req.Questions) * maxChoiceValue),
		CostExponent:      1,
	}

	if metaUri, err = u.vocClient.SetProcessMetadata(metadata, processID); err != nil {
		return fmt.Errorf("createProcessHandler: could not set process metadata: %w", err)
	}

	// TODO use encryption priv/pub keys if process is encrypted
	if startBlock, err = u.vocClient.CreateProcess(processID, entityID, startBlock,
		endBlock-startBlock, []byte{}, "", envelopeType, processMode,
		voteOptions, models.CensusOrigin_OFF_CHAIN_CA, metaUri, entitySignKeys); err != nil {
		return fmt.Errorf("createProcessHandler: could not create process on the vochain: %w", err)
	}

	if _, err = u.db.CreateElection(integratorPrivKey, entityID, processID, req.Title, startDate,
		endDate, uuid.NullUUID{}, int(startBlock), int(endBlock), req.Confidential, req.HiddenResults); err != nil {
		return fmt.Errorf("createProcessHandler: could not create election: %w", err)
	}
	resp.ElectionID = processID
	return sendResponse(resp, ctx)
}

// GET https://server/v1/priv/organizations/<organizationId>/elections/signed
// GET https://server/v1/priv/organizations/<organizationId>/elections/blind
// GET https://server/v1/priv/organizations/<organizationId>/elections/active
// GET https://server/v1/pprivub/organizations/<organizationId>/elections/ended
// GET https://server/v1/priv/organizations/<organizationId>/elections/upcoming
// listProcessesPrivateHandler' lists signed, blind, active, ended, or upcoming processes
func (u *URLAPI) listProcessesPrivateHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var entityId []byte
	var integratorPrivKey []byte
	var err error
	var resp []types.APIElection

	if integratorPrivKey, entityId, _, err = u.authEntityPermissions(msg, ctx); err != nil {
		return fmt.Errorf("listProcessesPrivateHandler: %w", err)
	}

	filter := ctx.URLParam("type")
	switch filter {
	case "active", "ended", "upcoming":
		var tempProcessList []string
		var totalProcs int
		var currentHeight uint32
		if currentHeight, err = u.vocClient.GetCurrentBlock(); err != nil {
			return fmt.Errorf("listProcessesPrivateHandler: could not get current block height: %w", err)
		}
		cont := true
		for cont {
			if tempProcessList, err = u.vocClient.GetProcessList(entityId,
				"", "", "", 0, false, totalProcs, 64); err != nil {
				return fmt.Errorf("listProcessesPrivateHandler: %s not a valid request path", ctx.Request.URL.Path)
			}
			if len(tempProcessList) < 64 {
				cont = false
			}
			totalProcs += len(tempProcessList)
			for _, processID := range tempProcessList {
				var processIDBytes []byte
				var newProcess *types.Election
				if processIDBytes, err = hex.DecodeString(processID); err != nil {
					log.Error(err)
					continue
				}
				if newProcess, err = u.db.GetElection(integratorPrivKey, entityId, processIDBytes); err != nil {
					log.Warn(fmt.Errorf("could not get election,"+
						" process %x may no be in db: %w", processIDBytes, err))
					continue
				}
				newProcess.OrgEthAddress = entityId
				newProcess.ProcessID = processIDBytes

				switch filter {
				case "active":
					if newProcess.StartBlock < int(currentHeight) && newProcess.EndBlock > int(currentHeight) {
						resp = append(resp, reflectElectionPrivate(*newProcess))
					}
				case "upcoming":
					if newProcess.StartBlock > int(currentHeight) {
						resp = append(resp, reflectElectionPrivate(*newProcess))
					}
				case "ended":
					if newProcess.EndBlock < int(currentHeight) {
						resp = append(resp, reflectElectionPrivate(*newProcess))
					}
				}
			}
		}
	case "blind", "signed":
		return fmt.Errorf("listProcessesPrivateHandler: filter %s unimplemented", filter)
	default:
		return fmt.Errorf("listProcessesPrivateHandler: %s not a valid filter", filter)

	}
	return sendResponse(resp, ctx)
}

// GET https://server/v1/priv/elections/<processId>
// getProcessHandler gets the entirety of a process, including metadata
// confidential processes need no extra step, only the api key
func (u *URLAPI) getProcessHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIProcess
	var processId []byte
	var vochainProcess *indexertypes.Process
	var results *types.VochainResults
	var processMetadata *types.ProcessMetadata
	if processId, err = util.GetBytesID(ctx, "electionId"); err != nil {
		return fmt.Errorf("getProcessHandler: %w", err)
	}

	// Fetch process from vochain
	if vochainProcess, err = u.vocClient.GetProcess(processId); err != nil {
		return fmt.Errorf("getProcessHandler: unable to fetch process from the vochain: %w", err)
	}

	// Fetch results
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			return fmt.Errorf("getProcessHandler: could not get results: %w", err)
		}
	}

	// Fetch metadata
	metadataUri := vochainProcess.Metadata
	if processMetadata, err = u.vocClient.FetchProcessMetadata(metadataUri); err != nil {
		return fmt.Errorf("getProcessHandler: could not get process metadata: %w", err)
	}

	// Parse all the information
	resp = u.parseProcessInfo(vochainProcess, results, processMetadata)

	return sendResponse(resp, ctx)
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

// PUT https://server/v1/priv/elections/<processId>/status
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
	if entityID, err = util.GetBytesID(ctx, "organizationId"); err != nil {
		return nil, nil, nil, err
	}
	if organization, err = u.db.GetOrganization(integratorPrivKey, entityID); err != nil {
		return nil, nil, nil, fmt.Errorf("entity %X could not be fetched from the db: %w", entityID, err)
	}
	if !bytes.Equal(organization.IntegratorApiKey, integratorPrivKey) {
		return nil, nil, nil, fmt.Errorf("entity %X does not belong to this integrator", entityID)
	}
	return integratorPrivKey, entityID, organization, nil
}

func reflectElectionPrivate(election types.Election) types.APIElection {
	newElection := types.APIElection{
		OrgEthAddress:   election.OrgEthAddress,
		ElectionID:      election.ProcessID,
		Title:           election.Title,
		CensusID:        election.CensusID.UUID.String(),
		StartDate:       election.StartDate,
		EndDate:         election.EndDate,
		StartBlock:      uint32(election.StartBlock),
		EndBlock:        uint32(election.EndBlock),
		Confidential:    election.Confidential,
		HiddenResults:   election.HiddenResults,
		MetadataPrivKey: election.MetadataPrivKey,
	}
	if election.CensusID.UUID == uuid.Nil {
		newElection.CensusID = ""
	}
	return newElection
}
