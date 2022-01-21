package transactions

import (
	"fmt"

	"github.com/google/uuid"
	"go.vocdoni.io/api/database"
)

type CreateOrganizationTx struct {
	TxBody
	IntegratorPrivKey []byte
	EthAddress        []byte
	EthPrivKeyCipher  []byte
	PlanID            uuid.NullUUID
	PublicApiQuota    int
	PublicApiToken    string
	HeaderUri         string
	AvatarUri         string
}

func (tx CreateOrganizationTx) commit(db *database.Database) error {
	_, err := (*db).CreateOrganization(tx.IntegratorPrivKey, tx.EthAddress,
		tx.EthPrivKeyCipher, tx.PlanID, tx.PublicApiQuota,
		tx.PublicApiToken, tx.HeaderUri, tx.AvatarUri)
	if err != nil {
		return fmt.Errorf("could not create organization: %w", err)
	}
	return nil
}

type UpdateOrganizationTx struct {
	TxBody
	IntegratorPrivKey []byte
	EthAddress        []byte
	HeaderUri         string
	AvatarUri         string
}

func (tx UpdateOrganizationTx) commit(db *database.Database) error {
	_, err := (*db).UpdateOrganization(tx.IntegratorPrivKey, tx.EthAddress,
		tx.HeaderUri, tx.AvatarUri)
	if err != nil {
		return fmt.Errorf("could not update organization: %w", err)
	}
	return nil
}
