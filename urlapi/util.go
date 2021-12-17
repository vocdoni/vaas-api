package urlapi

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	dvoteUtil "go.vocdoni.io/dvote/util"
	"go.vocdoni.io/dvote/vochain/scrutinizer/indexertypes"
	"go.vocdoni.io/proto/build/go/models"
)

const EXPLORER_NULLIFIER_URL = "https://vaas.explorer.vote/envelope/"

type orgPermissionsInfo struct {
	integratorPrivKey []byte
	entityID          []byte
	organization      *types.Organization
}

func (u *URLAPI) authEntityPermissions(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) (orgPermissionsInfo, error) {
	integratorPrivKey, err := util.GetAuthToken(msg)
	if err != nil {
		return orgPermissionsInfo{}, err
	}
	organizationID := common.HexToAddress(dvoteUtil.TrimHex(ctx.URLParam("organizationId")))
	organization, err := u.db.GetOrganization(integratorPrivKey, organizationID.Bytes())
	if err != nil {
		return orgPermissionsInfo{},
			fmt.Errorf("organization %s could not be fetched from the db: %w", organizationID.String(), err)
	}
	// if !bytes.Equal(organization.IntegratorApiKey, integratorPrivKey) {
	// 	return orgPermissionsInfo{}, fmt.Errorf("entity %s does not belong to this integrator", organizationID.String())
	// }
	return orgPermissionsInfo{
		integratorPrivKey: integratorPrivKey,
		entityID:          organizationID.Bytes(),
		organization:      organization,
	}, nil
}

func (u *URLAPI) parseProcessInfo(vc *indexertypes.Process,
	results *types.VochainResults, meta *types.ProcessMetadata) (types.APIElectionInfo, error) {
	process := types.APIElectionInfo{
		Description:        meta.Description["default"],
		OrganizationID:     vc.EntityID,
		Header:             meta.Media.Header,
		ElectionID:         vc.ID,
		ResultsAggregation: meta.Results.Aggregation,
		ResultsDisplay:     meta.Results.Display,
		EndBlock:           vc.EndBlock,
		StartBlock:         vc.StartBlock,
		StreamURI:          meta.Media.StreamURI,
		Title:              meta.Title["default"],
	}

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
	if len(models.ProcessStatus_name[vc.Status]) > 1 {
		process.Status = strings.ToTitle(models.ProcessStatus_name[vc.Status])[0:1] +
			strings.ToLower(models.ProcessStatus_name[vc.Status])[1:]
	}

	var err error
	if results != nil {
		process.VoteCount = results.Height
		if process.Results, err = aggregateResults(meta, results); err != nil {
			return process, fmt.Errorf("could not aggregate results: %v", err)
		}
	}

	if process.StartDate, err = u.estimateBlockTime(vc.StartBlock); err != nil {
		return process, fmt.Errorf("could not estimate startDate at %d: %w", vc.StartBlock, err)
	}

	if process.EndDate, err = u.estimateBlockTime(vc.EndBlock); err != nil {
		return process, fmt.Errorf("could not estimate endDate at %d: %w", vc.EndBlock, err)
	}
	return process, nil
}

func (u *URLAPI) estimateBlockTime(height uint32) (time.Time, error) {
	currentHeight, times, _, err := u.vocClient.GetBlockTimes()
	if err != nil {
		return time.Time{}, err
	}
	diffHeight := int64(height) - int64(currentHeight)
	inPast := diffHeight < 0
	absDiff := diffHeight
	if inPast {
		absDiff = -absDiff
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
	case absDiff < 100:
		t = getMaxTimeFrom(1)
	// if less than around 6 hours missing
	case absDiff < 1000:
		t = getMaxTimeFrom(3)
	// if less than around 6 hours missing
	case absDiff >= 1000:
		t = getMaxTimeFrom(4)
	}
	return time.Now().Add(time.Duration(diffHeight*int64(t)) * time.Millisecond), nil
}

func (u *URLAPI) estimateBlockHeight(target time.Time) (uint32, error) {
	currentHeight, times, _, err := u.vocClient.GetBlockTimes()
	if err != nil {
		return 0, err
	}
	currentTime := time.Now()
	// diff time in seconds
	diffTime := target.Unix() - currentTime.Unix()

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

// getProcessList gets a list of process summaries for given filters.
// if `private`, all processes are returned, including metadataPrivKeys, in the first return var.
// otherwise, confidential processes are returned first and public ones second.
func (u *URLAPI) getProcessList(filter string, integratorPrivKey, entityId []byte,
	private bool) ([]types.APIElectionSummary, []types.APIElectionSummary, error) {
	var priv []types.APIElectionSummary
	var pub []types.APIElectionSummary
	switch filter {
	case "active", "ended", "upcoming":
		var fullProcessList []string
		currentHeight, _, _, err := u.vocClient.GetBlockTimes()
		if err != nil {
			return nil, nil, fmt.Errorf("could not get current block height: %w", err)
		}
		// loop to fetch all processes
		for {
			tempProcessList, err := u.vocClient.GetProcessList(entityId,
				"", "", "", 0, false, len(fullProcessList), 64)
			if err != nil {
				return nil, nil, fmt.Errorf("unable to get process list from vochain: %w", err)
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
				log.Errorf("cannot decode process id %s: %v", processID, err)
				continue
			}
			if private {
				newProcess, err = u.db.GetElection(integratorPrivKey,
					entityId, processIDBytes)
			} else {
				newProcess, err = u.db.GetElectionPublic(entityId, processIDBytes)
			}
			if err != nil {
				log.Warnf("could not get election,"+
					" process %x may no be in db: %v", processIDBytes, err)
				continue
			}
			newProcess.OrgEthAddress = entityId
			newProcess.ProcessID = processIDBytes

			// filter processes by date
			switch filter {
			case "active":
				if newProcess.StartBlock < int(currentHeight) && newProcess.EndBlock > int(currentHeight) {
					appendProcess(&priv, &pub, newProcess, private)
				}
			case "upcoming":
				if newProcess.StartBlock > int(currentHeight) {
					appendProcess(&priv, &pub, newProcess, private)
				}
			case "ended":
				if newProcess.EndBlock < int(currentHeight) {
					appendProcess(&priv, &pub, newProcess, private)
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
	var aggregatedResults []types.Result
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

func appendProcess(priv, pub *[]types.APIElectionSummary, newProcess *types.Election,
	private bool) {
	if private {
		*priv = append(*priv, reflectElectionPrivate(*newProcess))
	} else {
		if newProcess.Confidential {
			*priv = append(*priv, reflectElectionPublic(*newProcess))
		} else {
			*pub = append(*pub, reflectElectionPublic(*newProcess))
		}
	}
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
