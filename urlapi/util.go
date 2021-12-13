package urlapi

import (
	"bytes"
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

type orgPermissionsInfo struct {
	integratorPrivKey []byte
	entityID          []byte
	organization      *types.Organization
}

func (u *URLAPI) authEntityPermissions(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) (orgPermissionsInfo, error) {
	var err error
	var entityID []byte
	var integratorPrivKey []byte
	var organization *types.Organization

	if integratorPrivKey, err = util.GetAuthToken(msg); err != nil {
		return orgPermissionsInfo{}, err
	}
	if entityID, err = util.GetBytesID(ctx, "organizationId"); err != nil {
		return orgPermissionsInfo{}, err
	}
	if organization, err = u.db.GetOrganization(integratorPrivKey, entityID); err != nil {
		return orgPermissionsInfo{}, fmt.Errorf("entity %X could not be fetched from the db: %w", entityID, err)
	}
	if !bytes.Equal(organization.IntegratorApiKey, integratorPrivKey) {
		return orgPermissionsInfo{}, fmt.Errorf("entity %X does not belong to this integrator", entityID)
	}
	return orgPermissionsInfo{
		integratorPrivKey: integratorPrivKey,
		entityID:          entityID,
		organization:      organization,
	}, nil
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
func (u *URLAPI) parseProcessInfo(vc *indexertypes.Process,
	results *types.VochainResults, meta *types.ProcessMetadata) (process types.APIElectionInfo) {
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
			log.Error(fmt.Errorf("could not aggregate results: %w", err))
		}
	}
	process.OrganizationID = vc.EntityID
	process.ElectionID = vc.ID

	process.StartBlock = vc.StartBlock
	process.EndBlock = vc.EndBlock

	if process.StartDate, err = u.estimateBlockTime(vc.StartBlock); err != nil {
		log.Warnf("could not estimate startDate at %d: %s", vc.StartBlock, err.Error())
	}

	if process.EndDate, err = u.estimateBlockTime(vc.EndBlock); err != nil {
		log.Warnf("could not estimate endDate at %d: %s", vc.EndBlock, err.Error())
	}

	process.ResultsAggregation = meta.Results.Aggregation
	process.ResultsDisplay = meta.Results.Display

	process.Ok = true

	return process
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

func (u *URLAPI) getProcessList(filter string, integratorPrivKey, entityId []byte,
	private bool) (pub []types.APIElectionSummary, priv []types.APIElectionSummary, err error) {
	switch filter {
	case "active", "ended", "upcoming":
		var tempProcessList []string
		var fullProcessList []string
		var currentHeight uint32
		if currentHeight, err = u.vocClient.GetCurrentBlock(); err != nil {
			return nil, nil, fmt.Errorf("could not get current block height: %w", err)
		}
		// loop to fetch all processes
		for {
			if tempProcessList, err = u.vocClient.GetProcessList(entityId,
				"", "", "", 0, false, len(fullProcessList), 64); err != nil {
				return nil, nil, fmt.Errorf("%s not a valid filter", filter)
			}
			fullProcessList = append(fullProcessList, tempProcessList...)
			if len(tempProcessList) < 64 {
				break
			}
		}
		// loop to fetch processes from db, filter by date
		for _, processID := range fullProcessList {
			var processIDBytes []byte
			var newProcess *types.Election
			if processIDBytes, err = hex.DecodeString(processID); err != nil {
				log.Error(err)
				continue
			}
			if private {
				newProcess, err = u.db.GetElection(integratorPrivKey,
					entityId, processIDBytes)
			} else {
				newProcess, err = u.db.GetElectionPublic(entityId, processIDBytes)
			}
			if err != nil {
				log.Warn(fmt.Errorf("could not get election,"+
					" process %x may no be in db: %w", processIDBytes, err))
				continue
			}
			newProcess.OrgEthAddress = entityId
			newProcess.ProcessID = processIDBytes

			// filter processes by date
			switch filter {
			case "active":
				if newProcess.StartBlock < int(currentHeight) && newProcess.EndBlock > int(currentHeight) {
					priv, pub = appendProcess(priv, pub, newProcess, private)
				}
			case "upcoming":
				if newProcess.StartBlock > int(currentHeight) {
					priv, pub = appendProcess(priv, pub, newProcess, private)
				}
			case "ended":
				if newProcess.EndBlock < int(currentHeight) {
					priv, pub = appendProcess(priv, pub, newProcess, private)
				}
			}
		}
	case "blind", "signed":
		return nil, nil, fmt.Errorf("filter %s unimplemented", filter)
	default:
		return nil, nil, fmt.Errorf("%s not a valid filter", filter)

	}
	return priv, pub, nil
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
func appendProcess(priv, pub []types.APIElectionSummary, newProcess *types.Election,
	private bool) (privateElections []types.APIElectionSummary,
	publicElections []types.APIElectionSummary) {
	if private {
		priv = append(priv, reflectElectionPrivate(*newProcess))
	} else {
		if newProcess.Confidential {
			priv = append(priv, reflectElectionPublic(*newProcess))
		} else {
			pub = append(pub, reflectElectionPublic(*newProcess))
		}
	}
	return priv, pub
}

func reflectElectionPrivate(election types.Election) types.APIElectionSummary {
	newElection := types.APIElectionSummary{
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
	// uuid.Nil returns a full zero-value uuid string. if there is no census uuid,
	// set the censusID string to empty so it is left out of the json response.
	if election.CensusID.UUID == uuid.Nil {
		newElection.CensusID = ""
	}
	return newElection
}

func reflectElectionPublic(election types.Election) types.APIElectionSummary {
	newElection := types.APIElectionSummary{
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
	// uuid.Nil returns a full zero-value uuid string. if there is no census uuid,
	// set the censusID string to empty so it is left out of the json response.
	if election.CensusID.UUID == uuid.Nil {
		newElection.CensusID = ""
	}
	return newElection
}
