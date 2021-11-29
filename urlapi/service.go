package urlapi

import (
	"fmt"

	"go.vocdoni.io/api/service"
)

func (u *URLAPI) EnableVotingServiceHandlers(s *service.VotingService) error {
	if s == nil {
		return fmt.Errorf("manager is nil")
	}
	u.service = s
	if err := u.enableEntityHandlers(); err != nil {
		return err
	}
	if err := u.enableIntegratorHandlers(); err != nil {
		return err
	}
	if err := u.enableSuperadminHandlers(); err != nil {
		return err
	}
	if err := u.enableVoterHandlers(); err != nil {
		return err
	}
	// if err := u.api.RegisterMethod(
	// 	"/manager/signUp",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	u.manager.SignUp,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/getEntity",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	u.manager.GetEntity,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/updateEntity",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.UpdateEntity,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/countMembers",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.CountMembers,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/listMembers",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.ListMembers,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/getMember",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.GetMember,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/updateMember",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.UpdateMember,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/deleteMembers",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.DeleteMembers,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/generateTokens",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.GenerateTokens,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/exportTokens",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.ExportTokens,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/importMembers",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.ImportMembers,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/countTargets",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.CountTargets,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/listTargets",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.ListTargets,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/getTarget",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.GetTarget,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/dumpTarget",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.DumpTarget,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/dumpCensus",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.DumpCensus,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/addCensus",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.AddCensus,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/updateCensus",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.UpdateCensus,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/getCensus",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.GetCensus,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/countCensus",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.CountCensus,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/listCensus",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.ListCensus,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/deleteCensus",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.DeleteCensus,
	// ); err != nil {
	// 	return err
	// }
	// if err := u.api.RegisterMethod(
	// 	"/manager/adminEntityList",
	// 	"GET",
	// 	bearerstdapi.MethodAccessTypePublic,
	// 	s.AdminEntityList,
	// ); err != nil {
	// 	return err
	// }
	// if s.HasEthClient() {
	// 	// do not expose this endpoint if the manager does not have an ethereum client
	// 	if err := u.api.RegisterMethod(
	// 		"/manager/requestGas",
	// 		"GET",
	// 		bearerstdapi.MethodAccessTypePublic,
	// 		s.RequestGas,
	// 	); err != nil {
	// 		return err
	// 	}
	// }
	return nil
}
