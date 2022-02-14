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

const (
	filterUnknown  = "UNKNOWN"
	filterPaused   = "PAUSED"
	filterCanceled = "CANCELED"
	filterUpcoming = "UPCOMING"
	filterActive   = "ACTIVE"
	filterEnded    = "ENDED"
	filterReady    = "READY"
	filterBlind    = "BLIND"
	filterSigned   = "SIGNED"
)

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
	// 	return orgPermissionsInfo{}, fmt.Errorf(
	// "entity %s does not belong to this integrator", organizationID.String())
	// }
	return orgPermissionsInfo{
		integratorPrivKey: integratorPrivKey,
		entityID:          organizationID.Bytes(),
		organization:      organization,
	}, nil
}

func (u *URLAPI) parseProcessInfo(vc *indexertypes.Process,
	results *types.VochainResults, meta *types.ProcessMetadata,
	proofType types.ProofType) (types.APIElectionInfo, error) {
	process := types.APIElectionInfo{
		Description:        meta.Description["default"],
		OrganizationID:     vc.EntityID,
		Header:             meta.Media.Header,
		ElectionID:         vc.ID,
		ResultsAggregation: meta.Results.Aggregation,
		ResultsDisplay:     meta.Results.Display,
		StreamURI:          meta.Media.StreamURI,
		Title:              meta.Title["default"],
		ProofType:          proofType,
	}
	if vc.Envelope.EncryptedVotes {
		keys, err := u.vocClient.GetProcessPubKeys(vc.ID)
		if err != nil {
			log.Errorf("could not get process keys: %v", err)
		} else {
			process.EncryptionPubKeys = keys
		}
	}
	if proofType == types.PROOF_TYPE_BLIND {
		process.Type = "blind-"
	} else if proofType == types.PROOF_TYPE_ECDSA {
		process.Type = "signed-"
	}
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
			newQuestion.Choices = append(newQuestion.Choices,
				types.Choice{Title: choice.Title["default"], Value: choice.Value})
		}
		process.Questions = append(process.Questions, newQuestion)
	}

	// Digest status to something more usable by the client
	switch vc.Status {
	case int32(models.ProcessStatus_PROCESS_UNKNOWN):
		process.Status = filterUnknown
	case int32(models.ProcessStatus_PAUSED):
		process.Status = filterPaused
	case int32(models.ProcessStatus_CANCELED):
		process.Status = filterCanceled
	default:
		blockHeight, _, _ := u.vocClient.GetBlockTimes()
		if vc.StartBlock >= blockHeight {
			process.Status = filterUpcoming
		} else if vc.StartBlock < blockHeight && vc.EndBlock > blockHeight {
			process.Status = filterActive
		} else {
			process.Status = filterEnded
		}
	}

	var err error
	if results != nil && vc.HaveResults {
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
	currentHeight, times, _ := u.vocClient.GetBlockTimes()
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
	currentHeight, times, _ := u.vocClient.GetBlockTimes()
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
	default:
		t = getMaxTimeFrom(4)
	}
	// Multiply by 1000 because t is represented in seconds, not ms.
	// Dividing t first can floor the integer, leading to divide-by-zero
	blockDiff := (uint32(absDiff*1000) / t)
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
	private bool) ([]types.APIElectionSummary, error) {
	var electionList []types.APIElectionSummary
	filter = strings.ToUpper(filter)

	// translation of API filter to vochain/gateway filter
	gwFilter := filter
	if gwFilter == filterActive || gwFilter == filterUpcoming {
		gwFilter = filterReady
	} else if gwFilter == filterBlind || gwFilter == filterSigned {
		gwFilter = ""
	}

	fullProcessList, err := u.fetchProcessList(entityId, gwFilter)
	if err != nil {
		return nil, err
	}

	currentHeight, _, _ := u.vocClient.GetBlockTimes()

	// fetch all processes from db
	for _, processID := range fullProcessList {
		processIDBytes, err := hex.DecodeString(processID)
		if err != nil {
			log.Errorf("cannot decode process id %s: %v", processID, err)
			continue
		}
		var newProcess *types.Election
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

		switch filter {
		case filterPaused, filterCanceled, filterEnded, "":
			// if filtering already handled, append all processes
			appendProcess(&electionList, newProcess, private, filter)
		case filterActive:
			// filter processes by date
			if newProcess.StartBlock < int(currentHeight) && newProcess.EndBlock > int(currentHeight) {
				appendProcess(&electionList, newProcess, private, filter)
			}
		case filterUpcoming:
			// filter processes by date
			if newProcess.StartBlock > int(currentHeight) {
				appendProcess(&electionList, newProcess, private, filter)
			}
		case filterBlind:
			// filter processes by proof type
			if newProcess.ProofType == "blind" {
				appendProcess(&electionList, newProcess, private, "")
			}
		case filterSigned:
			// filter processes by proof type
			if newProcess.ProofType == "ecdsa" {
				appendProcess(&electionList, newProcess, private, "")
			}
		default:
			return nil, fmt.Errorf("%s not a valid filter", filter)
		}
	}
	return electionList, nil
}

func (u *URLAPI) fetchProcessList(entityId []byte, status string) ([]string, error) {
	var fullProcessList []string
	for {
		tempProcessList, err := u.vocClient.GetProcessList(entityId,
			status, "", "", 0, false, len(fullProcessList), 64)
		if err != nil {
			return nil, fmt.Errorf("unable to get process list from vochain: %w", err)
		}
		fullProcessList = append(fullProcessList, tempProcessList...)
		if len(tempProcessList) < 64 {
			break
		}
	}
	return fullProcessList, nil
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
	if meta.Results.Aggregation != "discrete-values" &&
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

func appendProcess(electionList *[]types.APIElectionSummary, newProcess *types.Election,
	private bool, status string) {
	if private {
		newProc := reflectElectionPrivate(*newProcess)
		newProc.Status = status
		*electionList = append(*electionList, newProc)
	} else {
		if !newProcess.Confidential {
			newProc := reflectElectionPublic(*newProcess)
			newProc.Status = status
			*electionList = append(*electionList, newProc)
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
		Confidential:    &election.Confidential,
		HiddenResults:   &election.HiddenResults,
		MetadataPrivKey: election.MetadataPrivKey,
		ProofType:       types.ProofType(election.ProofType),
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
		Confidential:  &election.Confidential,
		HiddenResults: &election.HiddenResults,
		ProofType:     types.ProofType(election.ProofType),
	}
	// uuid.Nil returns a full zero-value uuid string. if there is no census uuid,
	// set the censusID string to empty so it is left out of the json response.
	if election.CensusID.UUID == uuid.Nil {
		newElection.CensusID = ""
	}
	return newElection
}
