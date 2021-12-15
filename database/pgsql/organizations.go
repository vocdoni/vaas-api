package pgsql

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/stdlib"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/log"
)

func (d *Database) CreateOrganization(integratorAPIKey, ethAddress, ethPrivKeyCipher []byte, planID uuid.NullUUID, publiApiQuota int, publicApiToken, headerUri, avatarUri string) (int, error) {
	integrator, err := d.GetIntegratorByKey(integratorAPIKey)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("Tried to createOrganization by uknown API Key %x", integratorAPIKey)
			return 0, fmt.Errorf("unkown API key: %x", integratorAPIKey)
		} else {
			return 0, fmt.Errorf("createOrganization DB error: %v", err)
		}
	}

	organization := &types.Organization{
		EthAddress:        ethAddress,
		IntegratorID:      integrator.ID,
		EthPrivKeyCicpher: ethPrivKeyCipher,
		IntegratorApiKey:  integrator.SecretApiKey,
		HeaderURI:         headerUri,
		AvatarURI:         avatarUri,
		PublicAPIToken:    publicApiToken,
		QuotaPlanID:       planID,
		PublicAPIQuota:    publiApiQuota,
		CreatedUpdated: types.CreatedUpdated{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	// TODO: Calculate EntityID (consult go-dvote)
	insert := `INSERT INTO organizations
			( integrator_id, integrator_api_key, eth_address, eth_priv_key_cipher, 
				header_uri, avatar_uri, public_api_token, quota_plan_id,
				public_api_quota, created_at, updated_at)
			VALUES ( :integrator_id, :integrator_api_key, :eth_address, :eth_priv_key_cipher, 
				:header_uri, :avatar_uri, :public_api_token, :quota_plan_id,
				:public_api_quota, :created_at, :updated_at)
			RETURNING id`
	result, err := d.db.NamedQuery(insert, organization)
	if err != nil {
		return 0, fmt.Errorf("error creating organization: %v", err)
	}
	if !result.Next() {
		return 0, fmt.Errorf("error creating organization: there is no next result row")
	}
	var id int
	err = result.Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("error creating organization: %v", err)
	}
	return id, nil
}

func (d *Database) GetOrganization(integratorAPIKey, ethAddress []byte) (*types.Organization, error) {
	var organization types.Organization
	selectOrganization := `SELECT id , integrator_id, integrator_api_key, eth_address, eth_priv_key_cipher, 
								header_uri, avatar_uri, public_api_token, quota_plan_id,
								public_api_quota, created_at, updated_at  
							FROM organizations WHERE integrator_api_key=$1 AND eth_address=$2`
	row := d.db.QueryRowx(selectOrganization, integratorAPIKey, ethAddress)
	err := row.StructScan(&organization)
	if err != nil {
		return nil, err
	}

	return &organization, nil
}

func (d *Database) DeleteOrganization(integratorAPIKey, ethAddress []byte) error {
	if len(integratorAPIKey) == 0 || len(ethAddress) == 0 {
		return fmt.Errorf("invalid arguments")
	}
	deleteQuery := `DELETE FROM organizations WHERE integrator_api_key=$1 AND eth_address=$2`
	result, err := d.db.Exec(deleteQuery, integratorAPIKey, ethAddress)
	if err != nil {
		return fmt.Errorf("error deleting organization: %v", err)
	}
	// var rows int64
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error veryfying deleted organization: %v", err)
	}
	if rows != 1 {
		return fmt.Errorf("nothing to delete")
	}
	return nil
}

func (d *Database) UpdateOrganization(integratorAPIKey, ethAddress []byte, headerUri, avatarUri string) (int, error) {
	if len(integratorAPIKey) == 0 || len(ethAddress) == 0 {
		return 0, fmt.Errorf("invalid arguments")
	}
	organization := &types.Organization{IntegratorApiKey: integratorAPIKey, EthAddress: ethAddress, HeaderURI: headerUri, AvatarURI: avatarUri}
	update := `UPDATE organizations SET
				header_uri = COALESCE(NULLIF(:header_uri, ''), header_uri),
				avatar_uri = COALESCE(NULLIF(:avatar_uri, ''), avatar_uri),
				updated_at = now()
				WHERE (integrator_api_key=:integrator_api_key AND eth_address=:eth_address)
				AND  (:quota_plan_id IS DISTINCT FROM quota_plan_id OR
					:public_api_quota IS DISTINCT FROM public_api_quota OR
					:header_uri IS DISTINCT FROM header_uri OR
					:avatar_uri IS DISTINCT FROM avatar_uri)`
	result, err := d.db.NamedExec(update, organization)
	if err != nil {
		return 0, fmt.Errorf("error updating organization: %v", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return 0, fmt.Errorf("cannot get affected rows: %v", err)
	} else if rows != 1 && rows != 0 { /* Nothing to update? */
		return int(rows), fmt.Errorf("expected to update 0 or 1 rows, but updated %d rows", rows)
	}
	return int(rows), nil
}

func (d *Database) UpdateOrganizationPlan(integratorAPIKey, ethAddress []byte, planID uuid.NullUUID, apiQuota int) (int, error) {
	if len(integratorAPIKey) == 0 || len(ethAddress) == 0 {
		return 0, fmt.Errorf("invalid arguments")
	}
	type PlanData struct {
		IntegratorAPIKey []byte    `db:"integrator_api_key"`
		EthAddress       []byte    `db:"eth_address"`
		PlanID           uuid.UUID `db:"quota_plan_id"`
		APIQuota         int       `db:"public_api_quota"`
	}

	plan := PlanData{
		IntegratorAPIKey: integratorAPIKey,
		EthAddress:       ethAddress,
		PlanID:           planID.UUID,
		APIQuota:         apiQuota,
	}
	update := `UPDATE organizations SET
				quota_plan_id = COALESCE(NULLIF(:quota_plan_id, NULL), quota_plan_id),
				public_api_quota = COALESCE(NULLIF(:public_api_quota, 0), public_api_quota),
				updated_at = now()
				WHERE (integrator_api_key=:integrator_api_key AND eth_address=:eth_address)
				AND  (:quota_plan_id IS DISTINCT FROM quota_plan_id OR
					:public_api_quota IS DISTINCT FROM public_api_quota
				)`
	result, err := d.db.NamedExec(update, plan)
	if err != nil {
		return 0, fmt.Errorf("error updating organization: %v", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return 0, fmt.Errorf("cannot get affected rows: %v", err)
	} else if rows != 1 && rows != 0 { /* Nothing to update? */
		return int(rows), fmt.Errorf("expected to update 0 or 1 rows, but updated %d rows", rows)
	}
	return int(rows), nil
}

func (d *Database) UpdateOrganizationEthPrivKeyCipher(integratorAPIKey, ethAddress, newEthPrivKeyCicpher []byte) (int, error) {
	if len(integratorAPIKey) == 0 || len(ethAddress) == 0 {
		return 0, fmt.Errorf("invalid arguments")
	}
	organization := &types.Organization{IntegratorApiKey: integratorAPIKey, EthAddress: ethAddress, EthPrivKeyCicpher: newEthPrivKeyCicpher}
	update := `UPDATE organizations SET
				eth_priv_key_cipher = COALESCE(NULLIF(:eth_priv_key_cipher, '' ::::bytea ),  eth_priv_key_cipher),
				updated_at = now()
				WHERE (integrator_api_key=:integrator_api_key AND eth_address=:eth_address)
				AND  (encode(:eth_priv_key_cipher,'hex') IS DISTINCT FROM encode(eth_priv_key_cipher,'hex'))`
	result, err := d.db.NamedExec(update, organization)
	if err != nil {
		return 0, fmt.Errorf("error updating organization: %v", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return 0, fmt.Errorf("cannot get affected rows: %v", err)
	} else if rows != 1 && rows != 0 { /* Nothing to update? */
		return int(rows), fmt.Errorf("expected to update 0 or 1 rows, but updated %d rows", rows)
	}
	return int(rows), nil
}

func (d *Database) UpdateOrganizationPublicAPIToken(integratorAPIKey, ethAddress []byte, newPublicApiToken string) (int, error) {
	if len(integratorAPIKey) == 0 || len(ethAddress) == 0 {
		return 0, fmt.Errorf("invalid arguments")
	}
	organization := &types.Organization{IntegratorApiKey: integratorAPIKey, EthAddress: ethAddress, PublicAPIToken: newPublicApiToken}
	update := `UPDATE organizations SET
				public_api_token = COALESCE(NULLIF(:public_api_token, ''),  public_api_token),
				updated_at = now()
				WHERE (integrator_api_key=:integrator_api_key AND eth_address=:eth_address)
				AND  (:public_api_token IS DISTINCT FROM public_api_token)`
	result, err := d.db.NamedExec(update, organization)
	if err != nil {
		return 0, fmt.Errorf("error updating organization: %v", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return 0, fmt.Errorf("cannot get affected rows: %v", err)
	} else if rows != 1 && rows != 0 { /* Nothing to update? */
		return int(rows), fmt.Errorf("expected to update 0 or 1 rows, but updated %d rows", rows)
	}
	return int(rows), nil
}

func (d *Database) CountOrganizations(integratorAPIKey []byte) (int, error) {
	selectQuery := `SELECT COUNT(*) FROM organizations WHERE integrator_api_key=$1`
	var entitiesCount int
	if err := d.db.Get(&entitiesCount, selectQuery, integratorAPIKey); err != nil {
		return 0, err
	}
	return entitiesCount, nil
}

func (d *Database) ListOrganizations(integratorAPIKey []byte, filter *types.ListOptions) ([]types.Organization, error) {
	// TODO: Replace limit offset with better strategy, can slow down DB
	// would nee to now last value from previous query
	selectQuery := `SELECT
	 				id, eth_address, header_uri, avatar_uri, public_api_token
					FROM organizations WHERE integrator_api_key =$1
					ORDER BY %s %s LIMIT $2 OFFSET $3`
	// Define default values for arguments
	t := reflect.TypeOf(types.Organization{})
	field, found := t.FieldByName("ID")
	if !found {
		return nil, fmt.Errorf("organization id field not found in DB. Something is very wrong")
	}
	orderField := field.Tag.Get("db")
	order := "ASC"
	var limit, offset sql.NullInt32
	// default limit should be nil (Postgres BIGINT NULL)
	err := limit.Scan(nil)
	if err != nil {
		return nil, err
	}
	err = offset.Scan(0)
	if err != nil {
		return nil, err
	}
	// offset := 0
	if filter != nil {
		if len(filter.SortBy) > 0 {
			field, found := t.FieldByName(strings.Title(filter.SortBy))
			if found {
				if filter.Order == "descend" {
					order = "DESC"
				}
				orderField = field.Tag.Get("db")
			}
		}
		if filter.Skip > 0 {
			err = offset.Scan(filter.Skip)
			if err != nil {
				return nil, err
			}
		}
		if filter.Count > 0 {
			err = limit.Scan(filter.Count)
			if err != nil {
				return nil, err
			}
		}
	}

	query := fmt.Sprintf(selectQuery, orderField, order)
	var entitites []types.Organization
	err = d.db.Select(&entitites, query, integratorAPIKey, limit, offset)
	if err != nil {
		return nil, err
	}
	return entitites, nil
}
