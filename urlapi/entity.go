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
		"/priv/account/organizations/{entityId}",
		"GET",
		bearerstdapi.MethodAccessTypePrivate,
		u.getOrganizationPrivateHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/organizations/{entityId}",
		"DELETE",
		bearerstdapi.MethodAccessTypePrivate,
		u.deleteOrganizationHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/account/organizations/{entityId}/key",
		"PATCH",
		bearerstdapi.MethodAccessTypePrivate,
		u.resetOrganizationKeyHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/organizations/{entityId}/metadata",
		"PUT",
		bearerstdapi.MethodAccessTypePrivate,
		u.setOrganizationMetadataHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/priv/organizations/{entityId}/processes/*",
		"POST",
		bearerstdapi.MethodAccessTypePrivate,
		u.createProcessHandler,
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
	if err := u.api.RegisterMethod(
		"/priv/processes/{processId}",
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
		log.Error(err)
		return err
	}
	if integratorPrivKey, err = util.GetAuthToken(msg); err != nil {
		log.Error(err)
		return err
	}
	orgApiToken = util.GenerateBearerToken()

	ethSignKeys := ethereum.NewSignKeys()
	if err = ethSignKeys.Generate(); err != nil {
		log.Errorf("could not generate ethereum keys: %v", err)
		return fmt.Errorf("could not generate ethereum keys: %v", err)
	}

	// Encrypt private key to store in db
	_, priv := ethSignKeys.HexString()
	if entityPrivKey, err = hex.DecodeString(priv); err != nil {
		log.Errorf("could not decode entity private key: %v", err)
		return fmt.Errorf("could not decode entity private key: %v", err)
	}

	if encryptedPrivKey, err = util.EncryptSymmetric(entityPrivKey, integratorPrivKey); err != nil {
		log.Errorf("could not encrypt organization private key: %v", err)
		return fmt.Errorf("could not encrypt organization private key: %v", err)
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
		log.Error(err)
		return err
	}

	// Register organization to database
	if _, err = u.db.CreateOrganization(integratorPrivKey, ethSignKeys.Address().Bytes(),
		encryptedPrivKey, uuid.NullUUID{}, 0, orgApiToken, req.Header, req.Avatar); err != nil {
		log.Errorf("could not create organization: %v", err)
		return fmt.Errorf("could not create organization: %v", err)
	}

	// Create the new account on the Vochain
	if err = u.vocClient.SetAccountInfo(ethSignKeys, metaURI); err != nil {
		log.Errorf("could not create account on the vochain: %v", err)
		return fmt.Errorf("could not create account on the vochain: %v", err)
	}

	resp.APIToken = orgApiToken
	resp.OrganizationID = ethSignKeys.Address().Bytes()

	return sendResponse(resp, ctx)
}

// GET https://server/v1/priv/account/organizations/<entityId>
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
		log.Error(err)
		return err
	}

	// Fetch process from vochain
	if metaUri, _, _, err = u.vocClient.GetAccount(organization.EthAddress); err != nil {
		log.Error(err)
		return err
	}

	// Fetch metadata
	if organizationMetadata, err = u.vocClient.FetchOrganizationMetadata(metaUri); err != nil {
		log.Errorf("could not get organization metadata with URI\"%s\": %v", metaUri, err)
		return fmt.Errorf("could not get organization metadata with URI\"%s\": %v", metaUri, err)
	}

	resp.APIToken = organization.PublicAPIToken
	resp.Name = organizationMetadata.Name["default"]
	resp.Description = organizationMetadata.Description["default"]
	resp.Avatar = organizationMetadata.Media.Avatar
	resp.Header = organizationMetadata.Media.Header
	return sendResponse(resp, ctx)
}

// DELETE https://server/v1/priv/account/organizations/<entityId>
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
		log.Error(err)
		return err
	}

	// Now generate a new api key & update integrator
	resp.APIToken = util.GenerateBearerToken()
	if _, err = u.db.UpdateOrganizationPublicAPIToken(
		integratorPrivKey, entityID, resp.APIToken); err != nil {
		log.Error(err)
		return err
	}
	return sendResponse(resp, ctx)
}

// PUT https://server/v1/priv/organizations/<entityId>/metadata
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
		log.Error(err)
		return err
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
		log.Error(err)
		return err
	}

	// Update organization in the db to make sure it matches the metadata
	if _, err = u.db.UpdateOrganization(organization.IntegratorApiKey, organization.EthAddress,
		req.Header, req.Avatar); err != nil {
		log.Error(err)
		return err
	}

	resp.OrganizationID = entityID
	resp.ContentURI = metaURI
	return sendResponse(resp, ctx)
}

// POST https://server/v1/priv/organizations/<entityId>/processes/signed
// POST https://server/v1/priv/organizations/<entityId>/processes/blind
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

	// if strings.HasSuffix(ctx.Request.URL.Path, "signed") {
	// 	blind = false
	// } else if strings.HasSuffix(ctx.Request.URL.Path, "blind") {
	// 	blind = true
	// } else {
	// 	log.Errorf("%s not a valid request path", ctx.Request.URL.Path)
	// 	return fmt.Errorf("%s not a valid request path", ctx.Request.URL.Path)
	// }

	// authenticate integrator has permission to edit this entity
	if integratorPrivKey, entityID, organization, err = u.authEntityPermissions(msg, ctx); err != nil {
		log.Error(err)
		return err
	}

	if req, err = util.UnmarshalRequest(msg); err != nil {
		log.Error(err)
		return err
	}

	if req.Confidential {
		log.Errorf("confidential processes are not yet supported")
		return fmt.Errorf("confidential processes are not yet supported")
	}

	pid := dvoteutil.RandomHex(32)
	if processID, err = hex.DecodeString(pid); err != nil {
		log.Error(err)
		return err
	}
	entityPrivKey, ok := util.DecryptSymmetric(organization.EthPrivKeyCicpher, integratorPrivKey)
	if !ok {
		log.Errorf("could not decrypt entity private key")
		return fmt.Errorf("could not decrypt entity private key")
	}
	entitySignKeys := ethereum.NewSignKeys()
	if err = entitySignKeys.AddHexKey(hex.EncodeToString(entityPrivKey)); err != nil {
		log.Errorf("could not decode entity private key: %v", err)
		return fmt.Errorf("could not decode entity private key: %v", err)
	}

	startDate, err := time.Parse("2006-01-02T15:04:05.000Z", req.StartDate)
	if err != nil {
		log.Error(err)
		return err
	}
	endDate, err := time.Parse("2006-01-02T15:04:05.000Z", req.EndDate)
	if err != nil {
		log.Error(err)
		return err
	}

	now := time.Now()
	if startDate.Before(now) || endDate.Before(now) {
		log.Errorf("election start and end date cannot be in the past")
		return fmt.Errorf("election start and end date cannot be in the past")
	}
	if endDate.Before(startDate) {
		log.Errorf("end date must be after start date")
		return fmt.Errorf("end date must be after start date")
	}
	startBlock, err := u.estimateBlockHeight(startDate)
	if err != nil {
		log.Error(err)
		return err
	}
	endBlock, err := u.estimateBlockHeight(endDate)
	if err != nil {
		log.Error(err)
		return err
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
	}

	voteOptions := &models.ProcessVoteOptions{
		MaxCount:          uint32(len(req.Questions)),
		MaxValue:          uint32(maxChoiceValue),
		MaxVoteOverwrites: 0,
		MaxTotalCost:      uint32(len(req.Questions) * maxChoiceValue),
		CostExponent:      1,
	}

	if metaUri, err = u.vocClient.SetProcessMetadata(metadata, processID); err != nil {
		log.Error(err)
		return err
	}

	// TODO use encryption priv/pub keys if process is encrypted
	if startBlock, err = u.vocClient.CreateProcess(processID, entityID, startBlock,
		endBlock-startBlock, []byte{}, "", envelopeType, processMode,
		voteOptions, models.CensusOrigin_OFF_CHAIN_CA, metaUri, entitySignKeys); err != nil {
		log.Error(err)
		return err
	}

	if _, err = u.db.CreateElection(integratorPrivKey, entityID, processID, req.Title, startDate,
		endDate, uuid.NullUUID{}, int(startBlock), int(endBlock), req.Confidential, req.HiddenResults); err != nil {
		log.Error(err)
		return err
	}
	resp.ProcessID = processID
	return sendResponse(resp, ctx)
}

// GET https://server/v1/priv/processes/<processId>
// getProcessHandler gets the entirety of a process, including metadata
// confidential processes need no extra step, only the api key
func (u *URLAPI) getProcessHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIProcess
	var processId []byte
	var vochainProcess *indexertypes.Process
	var results *types.VochainResults
	var processMetadata *types.ProcessMetadata
	if processId, err = util.GetBytesID(ctx, "processId"); err != nil {
		log.Error(err)
		return err
	}

	// Fetch process from vochain
	if vochainProcess, err = u.vocClient.GetProcess(processId); err != nil {
		log.Error(err)
		return err
	}

	// Fetch results
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			log.Error(err)
			return err
		}
	}

	// Fetch metadata
	metadataUri := vochainProcess.Metadata
	if processMetadata, err = u.vocClient.FetchProcessMetadata(metadataUri); err != nil {
		log.Error(err)
		return err
	}

	// Parse all the information
	resp = u.parseProcessInfo(vochainProcess, results, processMetadata)

	sendResponse(resp, ctx)
	return nil
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
	if entityID, err = util.GetBytesID(ctx, "entityId"); err != nil {
		return nil, nil, nil, err
	}
	if organization, err = u.db.GetOrganization(integratorPrivKey, entityID); err != nil {
		return nil, nil, nil, fmt.Errorf("entity %X could not be fetched from the db: %v", entityID, err)
	}
	if !bytes.Equal(organization.IntegratorApiKey, integratorPrivKey) {
		return nil, nil, nil, fmt.Errorf("entity %X does not belong to this integrator", entityID)
	}
	return integratorPrivKey, entityID, organization, nil
}
