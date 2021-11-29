package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"math/rand"

	"fmt"
	"reflect"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"go.vocdoni.io/api/ethclient"

	"go.vocdoni.io/api/database"
	"go.vocdoni.io/api/database/pgsql"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
)

type VotingService struct {
	db  database.Database
	eth *ethclient.Eth
}

// NewVotingService creates a new registry handler for the Router
func NewVotingService(d database.Database, ethclient *ethclient.Eth) *VotingService {
	return &VotingService{db: d, eth: ethclient}
}

func (m *VotingService) HasEthClient() bool {
	return m.eth != nil
}

func (m *VotingService) SignUp(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var entityInfo *types.EntityInfo
	var target *types.Target
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	var newEntity *types.EntityInfo
	if err := util.DecodeJsonMessage(newEntity, "entity", ctx); err != nil {
		return fmt.Errorf("cannot recover entity: %v", err)
	}

	entityInfo = &types.EntityInfo{CensusManagersAddresses: [][]byte{entityID}, Origins: []types.Origin{types.Token}}
	if newEntity != nil {
		// For now control which EntityInfo fields end up to the DB
		entityInfo.Name = newEntity.Name
		entityInfo.Email = newEntity.Email
		entityInfo.Size = newEntity.Size
		entityInfo.Type = newEntity.Type
	}

	// Add Entity
	if err = m.db.AddEntity(entityID, entityInfo); err != nil && !strings.Contains(err.Error(), "entities_pkey") {
		return fmt.Errorf("cannot add entity %x to the DB: (%v)", signaturePubKey, err)
	}

	target = &types.Target{EntityID: entityID, Name: "all", Filters: json.RawMessage([]byte("{}"))}
	if _, err = m.db.AddTarget(entityID, target); err != nil && !strings.Contains(err.Error(), "result has no rows") {
		return fmt.Errorf("cannot create entity's %x generic target: (%v)", signaturePubKey, err)
	}

	entityAddress := ethcommon.BytesToAddress(entityID)
	// do not try to send tokens if ethclient is nil
	if m.eth != nil {
		// send the default amount of faucet tokens iff wallet balance is zero
		sent, err := m.eth.SendTokens(context.Background(), entityAddress, 0, 0)
		if err != nil {
			if !strings.Contains(err.Error(), "maxAcceptedBalance") {
				return fmt.Errorf("error sending tokens to entity %s : %v", entityAddress.String(), err)
			}
			log.Warnf("signUp not sending tokens to entity %s : %v", entityAddress.String(), err)
		}
		response.Count = int(sent.Int64())
	}

	log.Debugf("Entity: %s signUp", entityAddress.String())
	return util.SendResponse(response, ctx)
}

func (m *VotingService) GetEntity(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if response.Entity, err = m.db.Entity(entityID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("entity requesting its info with getEntity not found")
		}
		return fmt.Errorf("cannot retrieve details of entity %x: (%v)", entityID, err)
	}

	log.Infof("listing details of Entity %x", entityID)
	return util.SendResponse(response, ctx)
}

func (m *VotingService) UpdateEntity(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	var newEntity *types.EntityInfo
	entityBytes, err := base64.StdEncoding.DecodeString(ctx.URLParam("entity"))
	if err != nil {
		return fmt.Errorf("cannot decode json string: (%s): %v", ctx.URLParam("entity"), err)
	}
	if err = json.Unmarshal(entityBytes, newEntity); err != nil {
		return fmt.Errorf("cannot recover entity or no data available for entity %s: %v", entityID, err)
	}

	entityInfo := &types.EntityInfo{
		Name:  newEntity.Name,
		Email: newEntity.Email,
		// Initialize values to accept empty spaces from the UI
		CallbackURL:    "",
		CallbackSecret: "",
	}
	if len(newEntity.CallbackURL) > 0 {
		entityInfo.CallbackURL = newEntity.CallbackURL
	}
	if len(newEntity.CallbackSecret) > 0 {
		entityInfo.CallbackSecret = newEntity.CallbackSecret
	}

	// Add Entity
	if response.Count, err = m.db.UpdateEntity(entityID, entityInfo); err != nil {
		return fmt.Errorf("cannot update entity %x to the DB: (%v)", entityID, err)
	}

	log.Debugf("Entity: %x entityUpdate", entityID)
	return util.SendResponse(response, ctx)
}

func (m *VotingService) ListMembers(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	var listOptions *types.ListOptions
	if err = util.DecodeJsonMessage(listOptions, "listOptions", ctx); err != nil {
		return err
	}

	// check filter
	if err = checkOptions(listOptions, ctx.URLParam("method")); err != nil {
		return fmt.Errorf("invalid filter options %x: (%v)", signaturePubKey, err)
	}

	// Query for members
	if response.Members, err = m.db.ListMembers(entityID, listOptions); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no members found")
		}
		return fmt.Errorf("cannot retrieve members of %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity: %x listMembers %d members", signaturePubKey, len(response.Members))
	return util.SendResponse(response, ctx)
}

func (m *VotingService) GetMember(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var memberID uuid.UUID
	var err error
	var response types.MetaResponse

	if len(ctx.URLParam("memberID")) == 0 {
		return fmt.Errorf("memberID is nil on getMember")
	}

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	if memberID, err = uuid.Parse(ctx.URLParam("memberID")); err != nil {
		return fmt.Errorf("cannot decode memberID: (%s): %v", ctx.URLParam("memberID"), err)
	}
	if response.Member, err = m.db.Member(entityID, &memberID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("member not found")
		}
		return fmt.Errorf("cannot retrieve member %q for entity %x: (%v)", ctx.URLParam("memberID"), signaturePubKey, err)
	}

	// TODO: Change when targets are implemented
	var targets []types.Target
	targets, err = m.db.ListTargets(entityID)
	if err == sql.ErrNoRows || len(targets) == 0 {
		log.Warnf("no targets found for member %q of entity %x", memberID.String(), signaturePubKey)
		response.Target = &types.Target{}
	} else if err == nil {
		response.Target = &targets[0]
	} else {
		return fmt.Errorf("error retrieving member %q targets for entity %x: (%v)", memberID.String(), signaturePubKey, err)
	}

	log.Infof("listing member %q for Entity with public Key %x", memberID.String(), signaturePubKey)
	return util.SendResponse(response, ctx)
}

func (m *VotingService) UpdateMember(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var member *types.Member
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	if err = util.DecodeJsonMessage(member, "member", ctx); err != nil {
		return err
	}

	// If a string Member property is sent as "" then it is not updated
	if response.Count, err = m.db.UpdateMember(entityID, &member.ID, &member.MemberInfo); err != nil {
		return fmt.Errorf("cannot update member %q for entity %x: (%v)", member.ID.String(), signaturePubKey, err)
	}

	log.Infof("update member %q for Entity with public Key %x", member.ID.String(), signaturePubKey)
	return util.SendResponse(response, ctx)
}

func (m *VotingService) DeleteMembers(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var memberIDs []uuid.UUID
	var err error
	var response types.MetaResponse

	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err := util.DecodeJsonMessage(&memberIDs, "memberIDs", ctx); err != nil {
		return err
	}
	response.Count, response.InvalidIDs, err = m.db.DeleteMembers(entityID, memberIDs)
	if err != nil {
		return fmt.Errorf("error deleting members for entity %x: (%v)", entityID, err)
	}

	log.Infof("deleted %d members, found %d invalid tokens, for Entity with public Key %x", response.Count, len(response.InvalidIDs), entityID)
	return util.SendResponse(response, ctx)
}

func (m *VotingService) CountMembers(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	// Query for members
	if response.Count, err = m.db.CountMembers(entityID); err != nil {
		return fmt.Errorf("cannot count members for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity %q countMembers: %d members", signaturePubKey, response.Count)
	return util.SendResponse(response, ctx)
}

func (m *VotingService) GenerateTokens(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var amount int
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(&amount, "amount", ctx); err != nil {
		return err
	}
	if amount < 1 {
		return fmt.Errorf("invalid token amount requested by %x", signaturePubKey)
	}

	response.Tokens = make([]uuid.UUID, amount)
	for idx := range response.Tokens {
		response.Tokens[idx] = uuid.New()
	}
	// TODO: Probably I need to initialize tokens
	if err = m.db.CreateMembersWithTokens(entityID, response.Tokens); err != nil {
		return fmt.Errorf("could not register generated tokens for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity: %x generateTokens: %d tokens", signaturePubKey, len(response.Tokens))
	return util.SendResponse(response, ctx)
}

func (m *VotingService) ExportTokens(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var members []types.Member
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	// TODO: Probably I need to initialize tokens
	if members, err = m.db.MembersTokensEmails(entityID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no members found")
		}
		return fmt.Errorf("could not retrieve members tokens for %x: (%v)", signaturePubKey, err)
	}
	response.MembersTokens = make([]types.TokenEmail, len(members))
	for idx, member := range members {
		response.MembersTokens[idx] = types.TokenEmail{Token: member.ID, Email: member.Email}
	}

	log.Debugf("Entity: %x exportTokens: %d tokens", signaturePubKey, len(members))
	return util.SendResponse(response, ctx)
}

func (m *VotingService) ImportMembers(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var membersInfo []types.MemberInfo
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(&membersInfo, "membersInfo", ctx); err != nil {
		return err
	}
	if len(membersInfo) < 1 {
		return fmt.Errorf("no member data provided for import members by %x", signaturePubKey)
	}

	for idx := range membersInfo {
		membersInfo[idx].Origin = types.Token
	}

	// Add members
	if err = m.db.ImportMembers(entityID, membersInfo); err != nil {
		return fmt.Errorf("could not import members for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity: %x importMembers: %d members", signaturePubKey, len(membersInfo))
	return util.SendResponse(response, ctx)
}

func (m *VotingService) CountTargets(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	// Query for members
	if response.Count, err = m.db.CountTargets(entityID); err != nil {
		return fmt.Errorf("cannot count targets for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity %x countTargets: %d targets", signaturePubKey, response.Count)
	return util.SendResponse(response, ctx)
}

func (m *VotingService) ListTargets(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var listOptions *types.ListOptions
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(listOptions, "listOptions", ctx); err != nil {
		return err
	}
	// check filter
	if err = checkOptions(listOptions, ctx.URLParam("method")); err != nil {
		return fmt.Errorf("invalid filter options %x: (%v)", signaturePubKey, err)
	}

	// Retrieve targets
	// Implement filters in DB
	response.Targets, err = m.db.ListTargets(entityID)
	if err != nil || len(response.Targets) == 0 {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no targets found")
		}
		return fmt.Errorf("cannot query targets for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity: %x listTargets: %d targets", signaturePubKey, len(response.Targets))
	return util.SendResponse(response, ctx)
}

func (m *VotingService) GetTarget(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var targetID *uuid.UUID
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(targetID, "targetID", ctx); err != nil {
		return err
	}
	if response.Target, err = m.db.Target(entityID, targetID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("target %q not found for %x", targetID.String(), signaturePubKey)
		}
		return fmt.Errorf("could not retrieve target for %x: %+v", signaturePubKey, err)
	}

	response.Members, err = m.db.TargetMembers(entityID, targetID)
	if err != nil {
		return fmt.Errorf("members for requested target could not be retrieved")
	}
	log.Debugf("Entity: %x getTarget: %s", signaturePubKey, targetID.String())
	return util.SendResponse(response, ctx)
}

func (m *VotingService) DumpTarget(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var target *types.Target
	var signaturePubKey []byte
	var entityID []byte
	var targetID *uuid.UUID
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(targetID, "targetID", ctx); err != nil {
		return err
	}
	if target, err = m.db.Target(entityID, targetID); err != nil || target.Name != "all" {
		if err == sql.ErrNoRows {
			return fmt.Errorf("target %q not found for %x", targetID.String(), signaturePubKey)
		}
		return fmt.Errorf("could not retrieve target for %x: (%v)", signaturePubKey, err)
	}

	// TODO: Implement DumpTargetClaims filtered directly by target filters
	if response.Claims, err = m.db.DumpClaims(entityID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no claims found for %x", signaturePubKey)
		}
		return fmt.Errorf("cannot dump claims for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity: %x dumpTarget: %d claims", signaturePubKey, len(response.Claims))
	return util.SendResponse(response, ctx)
}

func (m *VotingService) DumpCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var err error
	var entityID []byte
	var censusID []byte
	var signaturePubKey []byte
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	censusID, err = util.DecodeCensusID(ctx.URLParam("censusID"), signaturePubKey)
	if err != nil {
		return fmt.Errorf("cannot decode census id %s for %x", ctx.URLParam("censusID"), entityID)
	}

	// TODO: Implement DumpTargetClaims filtered directly by target filters
	censusMembers, err := m.db.ExpandCensusMembers(entityID, censusID)
	if err != nil {
		return fmt.Errorf("cannot dump claims for %q: (%v)", entityID, err)
	}
	shuffledClaims := make([][]byte, len(censusMembers))
	shuffledIndexes := rand.Perm(len(censusMembers))
	for i, v := range shuffledIndexes {
		shuffledClaims[v] = censusMembers[i].DigestedPubKey
	}
	response.Claims = shuffledClaims

	log.Debugf("Entity: %x dumpCensus: %d claims", entityID, len(response.Claims))
	return util.SendResponse(response, ctx)
}

func (m *VotingService) AddCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var signaturePubKey []byte
	var targetID *uuid.UUID
	var entityID []byte
	var censusID []byte
	var census *types.CensusInfo
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(targetID, "targetID", ctx); err != nil {
		return err
	}
	if len(targetID) == 0 {
		return fmt.Errorf("invalid target id %q for %x", targetID.String(), signaturePubKey)
	}
	censusID, err = util.DecodeCensusID(ctx.URLParam("censusID"), signaturePubKey)
	if err != nil {
		return fmt.Errorf("cannot decode census id %s for %x", ctx.URLParam("censusID"), entityID)
	}
	if len(censusID) == 0 {
		return fmt.Errorf("invalid census id %q for %x", censusID, signaturePubKey)
	}
	if err = util.DecodeJsonMessage(census, "census", ctx); err != nil {
		return err
	}
	if census == nil {
		return fmt.Errorf("invalid census info for census %q for entity %x", censusID, signaturePubKey)
	}
	// size, err := m.db.AddCensusWithMembers(entityID, censusID, request.TargetID, request.Census)
	if err := m.db.AddCensus(entityID, censusID, targetID, census); err != nil {
		return fmt.Errorf("cannot add census %q  for: %q: (%v)", censusID, entityID, err)
	}

	log.Debugf("Entity: %x addCensus: %s  ", entityID, censusID)
	return util.SendResponse(response, ctx)
}

func (m *VotingService) UpdateCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	// TODO Handle invalid claims
	var signaturePubKey []byte
	var entityID []byte
	var censusID []byte
	var invalidClaims [][]byte
	var census *types.CensusInfo
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	censusID, err = util.DecodeCensusID(ctx.URLParam("censusID"), signaturePubKey)
	if err != nil {
		return fmt.Errorf("cannot decode census id %s for %x", ctx.URLParam("censusID"), entityID)
	}
	if len(censusID) == 0 {
		return fmt.Errorf("invalid census id %q for %x", censusID, signaturePubKey)
	}

	if err = util.DecodeJsonMessage(census, "census", ctx); err != nil {
		return err
	}
	if census == nil {
		return fmt.Errorf("invalid census info for census %q for entity %x", censusID, signaturePubKey)
	}
	if err = util.DecodeJsonMessage(&invalidClaims, "invalidClaims", ctx); err != nil {
		return err
	}

	if len(invalidClaims) > 0 {
		return fmt.Errorf("invalid claims: %v", invalidClaims)
	}

	if response.Count, err = m.db.UpdateCensus(entityID, censusID, census); err != nil {
		return fmt.Errorf("cannot update census %q for %x: (%v)", censusID, entityID, err)
	}

	log.Debugf("Entity: %x updateCensus: %s \n %v", entityID, censusID, census)
	return util.SendResponse(response, ctx)
}

func (m *VotingService) GetCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var signaturePubKey []byte
	var entityID []byte
	var censusID []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	censusID, err = util.DecodeCensusID(ctx.URLParam("censusID"), signaturePubKey)
	if err != nil {
		return fmt.Errorf("cannot decode census id %s for %x", ctx.URLParam("censusID"), entityID)
	}
	if len(censusID) == 0 {
		return fmt.Errorf("invalid census id %q for %x", censusID, signaturePubKey)
	}

	response.Census, err = m.db.Census(entityID, censusID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("census %q not found for %x", censusID, signaturePubKey)
		}
		return fmt.Errorf("error in retrieving censuses for entity %x: (%v)", signaturePubKey, err)
	}

	response.Target, err = m.db.Target(entityID, &response.Census.TargetID)
	if err != nil {
		return fmt.Errorf("census target not found")
	}

	log.Debugf("Entity: %x getCensus:%s", signaturePubKey, censusID)
	return util.SendResponse(response, ctx)
}

func (m *VotingService) CountCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var signaturePubKey []byte
	var entityID []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	// Query for members
	if response.Count, err = m.db.CountCensus(entityID); err != nil {
		return fmt.Errorf("cannot count censuses for %x: (%v)", signaturePubKey, err)
	}

	log.Debugf("Entity %x countCensus: %d censuses", signaturePubKey, response.Count)
	return util.SendResponse(response, ctx)
}

func (m *VotingService) ListCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var signaturePubKey []byte
	var listOptions *types.ListOptions
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	if err = util.DecodeJsonMessage(listOptions, "listOptions", ctx); err != nil {
		return err
	}

	// check filter
	if err = checkOptions(listOptions, ctx.URLParam("method")); err != nil {
		return fmt.Errorf("invalid filter options %x: (%v)", signaturePubKey, err)
	}

	// Query for members
	// TODO Implement listCensus in Db that supports filters
	response.Censuses, err = m.db.ListCensus(entityID, listOptions)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no censuses found")
		}
		return fmt.Errorf("error in retrieving censuses for entity %x: (%v)", signaturePubKey, err)
	}
	log.Debugf("Entity: %x listCensuses: %d censuses", signaturePubKey, len(response.Censuses))
	return util.SendResponse(response, ctx)
}

func (m *VotingService) DeleteCensus(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var signaturePubKey []byte
	var entityID []byte
	var censusID []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	censusID, err = util.DecodeCensusID(ctx.URLParam("censusID"), signaturePubKey)
	if err != nil {
		return fmt.Errorf("cannot decode census id %s for %x", ctx.URLParam("censusID"), entityID)
	}
	if len(censusID) == 0 {
		return fmt.Errorf("invalid census id %q for %x", censusID, signaturePubKey)
	}

	err = m.db.DeleteCensus(entityID, censusID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("error deleting census %s for entity %x: (%v)", censusID, entityID, err)
	}

	log.Debugf("Entity: %x deleteCensus:%s", entityID, censusID)
	return util.SendResponse(response, ctx)
}

func (m *VotingService) AdminEntityList(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var signaturePubKey []byte
	var entityID []byte
	var err error
	var response types.MetaResponse

	if entityID, signaturePubKey, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}
	log.Debugf("%s", ethcommon.BytesToAddress((entityID)).String())
	if ethcommon.BytesToAddress((entityID)).String() != "0xCc41C6545234ac63F11c47bC282f89Ca77aB9945" {
		log.Warnf("invalid auth: (%v)", signaturePubKey)
		return fmt.Errorf("invalid auth")
	}

	// Query for members
	if response.Entities, err = m.db.AdminEntityList(); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no entities found")
		}
		return fmt.Errorf("cannot retrieve entities: (%v)", err)
	}

	log.Debugf("Entity: %x adminEntityList %d entities", signaturePubKey, len(response.Entities))
	return util.SendResponse(response, ctx)
}

func (m *VotingService) RequestGas(msg *bearerstdapi.BearerStandardAPIdata, ctx *httprouter.HTTPContext) error {
	var entityID []byte
	var err error
	var response types.MetaResponse

	if m.eth == nil {
		return fmt.Errorf("cannot request for tokens, ethereum client is nil")
	}
	if entityID, _, err = util.RetrieveEntityID(ctx); err != nil {
		return err
	}

	entityAddress := ethcommon.BytesToAddress(entityID)

	// check entity exists
	if _, err := m.db.Entity(entityID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("entity not found")
		}
		return fmt.Errorf("cannot retrieve details of entity %x: (%v)", entityID, err)
	}

	sent, err := m.eth.SendTokens(context.Background(), entityAddress, 0, 0)
	if err != nil {
		return fmt.Errorf("error sending tokens to entity %s : %v", entityAddress.String(), err)
	}

	response.Count = int(sent.Int64())
	return util.SendResponse(response, ctx)
}

func checkOptions(filter *types.ListOptions, method string) error {
	if filter == nil {
		return nil
	}
	// Check skip and count
	if filter.Skip < 0 || filter.Count < 0 {
		return fmt.Errorf("invalid skip/count")
	}
	var t reflect.Type
	// check method
	switch method {
	case "listMembers":
		t = reflect.TypeOf(types.MemberInfo{})
	case "listCensus":
		t = reflect.TypeOf(types.CensusInfo{})
	default:
		return fmt.Errorf("invalid method")
	}
	// Check sortby
	if len(filter.SortBy) > 0 {
		_, found := t.FieldByName(strings.Title(filter.SortBy))
		if !found {
			return fmt.Errorf("invalid filter field")
		}
		// sqli guard
		protectedOrderField := pgsql.ToOrderBySQLi(filter.SortBy)
		if protectedOrderField == -1 {
			return fmt.Errorf("invalid sort by field on query: %s", filter.SortBy)
		}
		// Check order
		if len(filter.Order) > 0 && !(filter.Order == "ascend" || filter.Order == "descend") {
			return fmt.Errorf("invalid filter order")
		}

	} else if len(filter.Order) > 0 {
		// Also check that order does not make sense without sortby
		return fmt.Errorf("invalid filter order")
	}
	return nil
}
