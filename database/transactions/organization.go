package transactions

import (
	"fmt"

	"github.com/google/uuid"
	"go.vocdoni.io/api/database"
)

// CreateOrganizationTx is the serializable transaction for creating an organization
type CreateOrganizationTx struct {
	TxBody
	IntegratorPrivKey []byte
	EthAddress        []byte
	EthPrivKeyCipher  []byte
	PlanID            uuid.NullUUID
	PublicAPIQuota    int32
	PublicAPIToken    string
	HeaderURI         string
	AvatarURI         string
}

func (tx CreateOrganizationTx) commit(db database.Database) error {
	_, err := db.CreateOrganization(tx.IntegratorPrivKey, tx.EthAddress,
		tx.EthPrivKeyCipher, tx.PlanID, int(tx.PublicAPIQuota),
		tx.PublicAPIToken, tx.HeaderURI, tx.AvatarURI)
	if err != nil {
		return fmt.Errorf("could not create organization: %w", err)
	}
	return nil
}

// UpdateOrganizationTx is the serializable transaction for updating an organization
//  commit commits the tx to the sql database
type UpdateOrganizationTx struct {
	TxBody
	IntegratorPrivKey []byte
	EthAddress        []byte
	HeaderUri         string
	AvatarUri         string
}

func (tx UpdateOrganizationTx) commit(db database.Database) error {
	_, err := db.UpdateOrganization(tx.IntegratorPrivKey, tx.EthAddress,
		tx.HeaderUri, tx.AvatarUri)
	if err != nil {
		return fmt.Errorf("could not update organization: %w", err)
	}
	return nil
}
