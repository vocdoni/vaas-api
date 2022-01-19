package transactions

import (
	"fmt"

	"github.com/google/uuid"
	"go.vocdoni.io/api/database"
)

type CreateOrganizationTx struct {
	IntegratorPrivKey []byte
	EthAddress        []byte
	EthPrivKeyCipher  []byte
	PlanID            uuid.NullUUID
	PublicApiQuota    int
	PublicApiToken    string
	HeaderUri         string
	AvatarUri         string
}

func (tx CreateOrganizationTx) Commit(db *database.Database) (int, error) {
	id, err := (*db).CreateOrganization(tx.IntegratorPrivKey, tx.EthAddress,
		tx.EthPrivKeyCipher, tx.PlanID, tx.PublicApiQuota,
		tx.PublicApiToken, tx.HeaderUri, tx.AvatarUri)
	if err != nil {
		return 0, fmt.Errorf("could not create organization: %w", err)
	}
	return id, nil
}

type UpdateOrganizationTx struct {
	IntegratorPrivKey []byte
	EthAddress        []byte
	HeaderUri         string
	AvatarUri         string
}

func (tx UpdateOrganizationTx) Commit(db *database.Database) (int, error) {
	_, err := (*db).UpdateOrganization(tx.IntegratorPrivKey, tx.EthAddress,
		tx.HeaderUri, tx.AvatarUri)
	if err != nil {
		return 0, fmt.Errorf("could not update organization: %w", err)
	}
	return 0, nil
}
