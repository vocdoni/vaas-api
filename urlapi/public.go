package urlapi

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/vochain/scrutinizer/indexertypes"
	"go.vocdoni.io/proto/build/go/models"
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
		return fmt.Errorf("registerPublicKeyHandler: %w", err)
	}

	filter := ctx.URLParam("type")
	switch filter {
	case "active", "ended", "upcoming":
		var tempProcessList []string
		var totalProcs int
		var currentHeight uint32
		if currentHeight, err = u.vocClient.GetCurrentBlock(); err != nil {
			return fmt.Errorf("registerPublicKeyHandler: could not get current block height: %w", err)
		}
		cont := true
		for cont {
			if tempProcessList, err = u.vocClient.GetProcessList(entityId,
				"", "", "", 0, false, totalProcs, 64); err != nil {
				return fmt.Errorf("registerPublicKeyHandler: %s not a valid filter", filter)
			}
			if len(tempProcessList) < 64 {
				cont = false
			}
			totalProcs += len(tempProcessList)
			for _, processID := range tempProcessList {
				var processIDBytes []byte
				var newProcess *types.Election
				if processIDBytes, err = hex.DecodeString(processID); err != nil {
					log.Errorf("registerPublicKeyHandler: %w", err)
					continue
				}
				if newProcess, err = u.db.GetElectionPublic(entityId, processIDBytes); err != nil {
					log.Warn(fmt.Errorf("registerPublicKeyHandler: could not get public election,"+
						" process %x may not be in db: %w", processIDBytes, err))
					continue
				}
				newProcess.OrgEthAddress = entityId
				newProcess.ProcessID = processIDBytes

				switch filter {
				case "active":
					if newProcess.StartBlock < int(currentHeight) && newProcess.EndBlock > int(currentHeight) {
						if newProcess.Confidential {
							resp.PrivateProcesses = append(resp.PrivateProcesses, reflectElectionPublic(*newProcess))
						} else {
							resp.PublicProcesses = append(resp.PublicProcesses, reflectElectionPublic(*newProcess))
						}
					}
				case "upcoming":
					if newProcess.StartBlock > int(currentHeight) {
						if newProcess.Confidential {
							resp.PrivateProcesses = append(resp.PrivateProcesses, reflectElectionPublic(*newProcess))
						} else {
							resp.PublicProcesses = append(resp.PublicProcesses, reflectElectionPublic(*newProcess))
						}
					}
				case "ended":
					if newProcess.EndBlock < int(currentHeight) {
						if newProcess.Confidential {
							resp.PrivateProcesses = append(resp.PrivateProcesses, reflectElectionPublic(*newProcess))
						} else {
							resp.PublicProcesses = append(resp.PublicProcesses, reflectElectionPublic(*newProcess))
						}
					}
				}
			}
		}
	case "blind", "signed":
		return fmt.Errorf("registerPublicKeyHandler: filter %s unimplemented", filter)
	default:
		return fmt.Errorf("registerPublicKeyHandler: %s not a valid filter type", filter)

	}
	return sendResponse(resp, ctx)
}

// GET https://server/v1/pub/elections/<processId>
// getProcessInfoPublicHandler gets public process info
func (u *URLAPI) getProcessInfoPublicHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIProcess
	var processId []byte
	var vochainProcess *indexertypes.Process
	var results *types.VochainResults
	var processMetadata *types.ProcessMetadata
	if processId, err = util.GetBytesID(ctx, "electionId"); err != nil {
		return fmt.Errorf("getProcessInfoPublicHandler: %w", err)
	}

	// Fetch process from vochain
	if vochainProcess, err = u.vocClient.GetProcess(processId); err != nil {
		return fmt.Errorf("getProcessInfoPublicHandler: unable to get process: %w", err)
	}

	// Fetch results
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			return fmt.Errorf("getProcessInfoPublicHandler: unable to get results %w", err)
		}
	}

	// Fetch metadata
	metadataUri := vochainProcess.Metadata
	if processMetadata, err = u.vocClient.FetchProcessMetadata(metadataUri); err != nil {
		return fmt.Errorf("getProcessInfoPublicHandler: unable to get metadata: %w", err)
	}

	// Parse all the information
	resp = u.parseProcessInfo(vochainProcess, results, processMetadata)

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
	var organization *types.Organization
	var organizationMetadata *types.EntityMetadata
	var metaUri string
	// authenticate integrator has permission to edit this entity
	if _, _, organization, err = u.authEntityPermissions(msg, ctx); err != nil {
		return fmt.Errorf("getOrganizationHandler: %w", err)
	}
	// Fetch process from vochain
	if metaUri, _, _, err = u.vocClient.GetAccount(organization.EthAddress); err != nil {
		return fmt.Errorf("getOrganizationHandler: unable to get account: %w", err)
	}

	// Fetch metadata
	if organizationMetadata, err = u.vocClient.FetchOrganizationMetadata(metaUri); err != nil {
		return fmt.Errorf("getOrganizationHandler: could not get organization metadata with URI\"%s\": %w", metaUri, err)
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
		return fmt.Errorf("submitVotePublicHandler: %w", err)
	}
	var votePkg []byte
	if votePkg, err = base64.StdEncoding.DecodeString(req.Vote); err != nil {
		return fmt.Errorf("submitVotePublicHandler: could not decode vote pkg to base64: %w", err)
	}
	if resp.Nullifier, err = u.vocClient.RelayTx(votePkg); err != nil {
		return fmt.Errorf("submitVotePublicHandler: could not submit vote tx: %w", err)
	}

	return sendResponse(resp, ctx)
}

// TODO add listProcessesInfoHandler

func (u *URLAPI) parseProcessInfo(vc *indexertypes.Process,
	results *types.VochainResults, meta *types.ProcessMetadata) (process types.APIProcess) {
	var err error

	// TODO update when blind is added to election
	// if db.Blind {
	process.Type = "blind-"
	// 	} else {
	// 	resp.Type = "signed-"
	// }
	if vc.Mode.EncryptedMetaData {
		process.Type += "confidential-"
	} else {
		process.Type += "plain-"
	}
	if vc.Envelope.EncryptedVotes {
		process.Type += "hidden-results"
	} else {
		process.Type += "rolling-results"
	}

	process.Title = meta.Title["default"]
	process.Description = meta.Description["default"]
	process.Header = meta.Media.Header
	process.StreamURI = meta.Media.StreamURI

	for _, question := range meta.Questions {
		newQuestion := types.Question{
			Title:       question.Title["default"],
			Description: question.Description["default"],
		}
		for _, choice := range question.Choices {
			newQuestion.Choices = append(newQuestion.Choices, choice.Title["default"])
		}
		process.Questions = append(process.Questions, newQuestion)
	}
	process.Status = strings.ToTitle(models.ProcessStatus_name[vc.Status])[0:1] +
		strings.ToLower(models.ProcessStatus_name[vc.Status])[1:]

	if results != nil {
		process.VoteCount = results.Height
		if process.Results, err = aggregateResults(meta, results); err != nil {
			log.Errorf("could not aggregate results: %w", err)
		}
	}
	process.OrganizationID = vc.EntityID
	process.ElectionID = vc.ID

	process.StartBlock = vc.StartBlock
	process.EndBlock = vc.EndBlock

	if process.StartDate, err = u.estimateBlockTime(vc.StartBlock); err != nil {
		log.Warnf("could not estimate startDate at %d: %w", vc.StartBlock, err)
	}

	if process.EndDate, err = u.estimateBlockTime(vc.EndBlock); err != nil {
		log.Warnf("could not estimate endDate at %d: %w", vc.EndBlock, err)
	}

	process.ResultsAggregation = meta.Results.Aggregation
	process.ResultsDisplay = meta.Results.Display

	process.Ok = true

	return process
}

func (u *URLAPI) estimateBlockTime(height uint32) (time.Time, error) {
	currentHeight, err := u.vocClient.GetCurrentBlock()
	if err != nil {
		return time.Time{}, err
	}
	diffHeight := int64(height) - int64(currentHeight)

	if diffHeight < 0 {
		blk, err := u.vocClient.GetBlock(height)
		if err != nil {
			return time.Time{}, err
		}
		if blk == nil {
			return time.Time{}, fmt.Errorf("cannot get block height %d", height)
		}
		return blk.Timestamp, nil
	}

	times, err := u.vocClient.GetBlockTimes()
	if err != nil {
		return time.Time{}, err
	}

	getMaxTimeFrom := func(i int) uint32 {
		for ; i >= 0; i-- {
			if times[i] != 0 {
				return uint32(times[i])
			}
		}
		return 10000 // fallback
	}

	t := uint32(0)
	switch {
	// if less than around 15 minutes missing
	case diffHeight < 100:
		t = getMaxTimeFrom(1)
	// if less than around 6 hours missing
	case diffHeight < 1000:
		t = getMaxTimeFrom(3)
	// if less than around 6 hours missing
	case diffHeight >= 1000:
		t = getMaxTimeFrom(4)
	}
	return time.Now().Add(time.Duration(diffHeight*int64(t)) * time.Millisecond), nil
}

func (u *URLAPI) estimateBlockHeight(target time.Time) (uint32, error) {
	currentHeight, err := u.vocClient.GetCurrentBlock()
	if err != nil {
		return 0, err
	}
	currentTime := time.Now()
	// diff time in seconds
	diffTime := target.Unix() - currentTime.Unix()

	times, err := u.vocClient.GetBlockTimes()
	if err != nil {
		return 0, err
	}

	// block time in ms
	getMaxTimeFrom := func(i int) uint32 {
		for ; i >= 0; i-- {
			if times[i] != 0 {
				return uint32(times[i])
			}
		}
		return 10000 // fallback
	}
	inPast := diffTime < 0
	absDiff := diffTime
	if inPast {
		absDiff = -absDiff
	}
	t := uint32(0)
	switch {
	// if less than around 15 minutes missing
	case absDiff < 900:
		t = getMaxTimeFrom(1)
	// if less than around 6 hours missing
	case absDiff < 21600:
		t = getMaxTimeFrom(3)
	// if less than around 6 hours missing
	case absDiff >= 21600:
		t = getMaxTimeFrom(4)
	}
	blockDiff := uint32(absDiff) / uint32((t / 1000))
	if inPast {
		if blockDiff > currentHeight {
			return 0, fmt.Errorf("target time %v is before Vochain origin", target)
		}
		return currentHeight - uint32(blockDiff), nil
	}
	return currentHeight + uint32(blockDiff), nil
}

func aggregateResults(meta *types.ProcessMetadata,
	results *types.VochainResults) ([]types.Result, error) {
	var aggregatedResults []types.Result
	if meta == nil {
		return nil, fmt.Errorf("no process metadata provided")
	}
	if meta.Questions == nil || len(meta.Questions) == 0 {
		return nil, fmt.Errorf("process meta has no questions")
	}
	if results == nil || len(results.Results) == 0 {
		return nil, fmt.Errorf("process results struct is empty")

	}
	if len(meta.Questions) != len(results.Results) {
		return nil, fmt.Errorf("number of results does not match number of questions")
	}
	if meta.Results.Aggregation != "discrete-counting" &&
		meta.Results.Aggregation != "index-weighted" {
		return nil, fmt.Errorf("process aggregation method %s not supported", meta.Results.Aggregation)
	}
	for i, question := range meta.Questions {
		var titles []string
		var values []string
		if len(question.Choices) > len(results.Results[i]) {
			return nil, fmt.Errorf("number of results does not match number of choices")
		}
		for _, choice := range question.Choices {
			titles = append(titles, choice.Title["default"])
			values = append(values, results.Results[i][choice.Value])
		}
		aggregatedResults = append(aggregatedResults, types.Result{
			Title: titles,
			Value: values,
		})
	}
	return aggregatedResults, nil
}

func reflectElectionPublic(election types.Election) types.APIElection {
	newElection := types.APIElection{
		OrgEthAddress: election.OrgEthAddress,
		ElectionID:    election.ProcessID,
		Title:         election.Title,
		CensusID:      election.CensusID.UUID.String(),
		StartDate:     election.StartDate,
		EndDate:       election.EndDate,
		StartBlock:    uint32(election.StartBlock),
		EndBlock:      uint32(election.EndBlock),
		Confidential:  election.Confidential,
		HiddenResults: election.HiddenResults,
	}
	if election.CensusID.UUID == uuid.Nil {
		newElection.CensusID = ""
	}
	return newElection
}
