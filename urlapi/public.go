package urlapi

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

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
		"/priv/entities/{entityId}/processes/*",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.listProcessesHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/processes/{processId}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.getProcessInfoPublicHandler,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/pub/processes/{processId}/auth/{signature}",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.getProcessInfoConfidentialHandler,
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

// GET https://server/v1/priv/entities/<entityId>/processes/signed
// GET https://server/v1/priv/entities/<entityId>/processes/blind
// GET https://server/v1/priv/entities/<entityId>/processes/active
// GET https://server/v1/priv/entities/<entityId>/processes/ended
// GET https://server/v1/priv/entities/<entityId>/processes/upcoming
// listProcessesHandler lists signed, blind, active, ended, or upcoming processes
func (u *URLAPI) listProcessesHandler(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// GET https://server/v1/pub/processes/<processId>
// getProcessInfoPublicHandler gets public process info
func (u *URLAPI) getProcessInfoPublicHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	var err error
	var resp types.APIProcess
	var processId []byte
	var process *types.Election
	var vochainProcess *indexertypes.Process
	var results *types.VochainResults
	var processMetadata *types.ProcessMetadata
	log.Debugf("get process id")
	if processId, err = util.GetBytesID(ctx, "processId"); err != nil {
		log.Error(err)
		return err
	}

	log.Debugf("get process db")
	// Fetch process from db
	// if process, err = u.db.GetElection([]byte{}, []byte{}, processId); err != nil {
	// log.Error(err)
	// 	return err
	// }

	// TODO REMOVE dummy process for testing
	process = &types.Election{
		CreatedUpdated:   types.CreatedUpdated{},
		ID:               2,
		OrgEthAddress:    []byte{012},
		IntegratorApiKey: []byte{012},
		ProcessID:        processId,
		Title:            "test election",
		CensusID:         3,
		StartBlock:       *big.NewInt(1518551),
		EndBlock:         *big.NewInt(30909000),
		Confidential:     true,
		HiddenResults:    true,
	}

	log.Debugf("get process vochain")
	// Fetch process from vochain
	if vochainProcess, err = u.vocClient.GetProcess(processId); err != nil {
		log.Error(err)
		return err
	}

	log.Debugf("get process results")
	// Fetch results
	if vochainProcess.HaveResults {
		if results, err = u.vocClient.GetResults(processId); err != nil {
			log.Error(err)
			return err
		}
	}

	log.Debugf("get process meta")
	// Fetch metadata
	metadataUri := vochainProcess.Metadata
	if processMetadata, err = u.vocClient.FetchProcessMetadata(metadataUri); err != nil {
		log.Error(err)
		return err
	}

	// Parse all the information
	log.Debugf("parse process info")
	resp = u.parseProcessInfo(process, vochainProcess, results, processMetadata)

	log.Debugf("send resp %v", resp)
	data, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("error marshaling JSON: %v", err)
		return fmt.Errorf("error marshaling JSON: %w", err)
	}
	if err = ctx.Send(data); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// GET https://server/v1/pub/processes/<processId>/auth/<signature>
// getProcessInfoConfidentialHandler gets process info, including private metadata,
//  checking the voter's signature for inclusion in the census
func (u *URLAPI) getProcessInfoConfidentialHandler(msg *bearerstdapi.BearerStandardAPIdata,
	ctx *httprouter.HTTPContext) error {
	log.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
	return fmt.Errorf("endpoint %s unimplemented", ctx.Request.URL.String())
}

// TODO add listProcessesInfoHandler

func (u *URLAPI) parseProcessInfo(db *types.Election, vc *indexertypes.Process,
	results *types.VochainResults, meta *types.ProcessMetadata) (process types.APIProcess) {
	var err error

	// TODO update when blind is added to election
	// if db.Blind {
	process.Type = "blind-"
	// 	} else {
	// 	resp.Type = "signed-"
	// }
	if db.Confidential {
		process.Type += "confidential-"
	} else {
		process.Type += "plain-"
	}
	if db.HiddenResults {
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
			log.Errorf("could not aggregate results: %v", err)
		}
	}
	process.EntityID = vc.EntityID
	process.ProcessID = db.ProcessID

	process.StartBlock = db.StartBlock.String()
	process.EndBlock = db.EndBlock.String()

	if process.StartDate, err = u.estimateBlockTime(uint32(db.StartBlock.Int64())); err != nil {
		log.Warnf("could not estimate startDate at %s: %v", db.StartBlock.String(), err)
	}

	if process.EndDate, err = u.estimateBlockTime(uint32(db.EndBlock.Int64())); err != nil {
		log.Warnf("could not estimate endDate at %s: %v", db.EndBlock.String(), err)
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
		return 10 // fallback
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
