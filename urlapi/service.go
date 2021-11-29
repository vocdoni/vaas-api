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
	return nil
}
