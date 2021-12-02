package pgsql

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	_ "github.com/jackc/pgx/stdlib"

	"go.vocdoni.io/api/types"
)

func (d *Database) CreateOrganization(integratorID, planID, apiQuota int, ethAddress, metadataPrivKey []byte, publicApiToken, headerUri, avatarUri string) (int32, error) {
	organization := &types.Organization{
		EthAddress:   ethAddress,
		IntegratorID: integratorID,
		CreatedUpdated: types.CreatedUpdated{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	// TODO: Calculate EntityID (consult go-dvote)
	insert := `INSERT INTO organizations
			(id, integrator_id is_authorized, email, name, size, created_at, updated_at)
			VALUES (:id, integrator_id :is_authorized, :email, :name, :size, :created_at, :updated_at)`
	result, err := d.db.NamedQuery(insert, organization)
	if err != nil || !result.Next() {
		return 0, fmt.Errorf("error inserting tag: %w", err)
	}
	var id int32
	err = result.Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("error inserting tag: %w", err)
	}
	return id, nil
}

func (d *Database) GetOrganization(integratorID int, entityID []byte) (*types.Organization, error) {
	var organization *types.Organization
	selectEntity := `SELECT id, is_authorized, email, name, type, size, callback_url, callback_secret, census_managers_addresses as "pg_census_managers_addresses"  
						FROM organizations WHERE id=$1`
	row := d.db.QueryRowx(selectEntity, entityID)
	err := row.StructScan(&organization)
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (d *Database) DeleteOrganization(integratorID int, entityID []byte) error {
	if len(entityID) == 0 {
		return fmt.Errorf("invalid arguments")
	}

	deleteQuery := `DELETE FROM organizations WHERE id = $1 AND integrator_id = $2`
	result, err := d.db.Exec(deleteQuery, entityID, integratorID)
	if err != nil {
		return fmt.Errorf("error deleting organization: %w", err)
	}
	// var rows int64
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error veryfying deleted organization: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("nothing to delete")
	}
	return nil
}

func (d *Database) AuthorizeOrganization(integratorID int, ethAddress []byte) error {
	organization := &types.Organization{EthAddress: ethAddress, IntegratorID: integratorID}
	update := `UPDATE organizations SET
				is_authorized = COALESCE(NULLIF(:is_authorized, false), is_authorized),
				updated_at = now()
				WHERE (id = :id  AND integrator_id = :integrator_id)
				AND  :is_authorized IS DISTINCT FROM is_authorized`
	result, err := d.db.NamedExec(update, organization)
	if err != nil {
		return fmt.Errorf("error updating organization: %w", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return fmt.Errorf("cannot get affected rows: %w", err)
	} else if rows == 0 { /* Nothing to update? */
		return fmt.Errorf("already authorized")
	} else if rows != 1 { /* Nothing to update? */
		return fmt.Errorf("could not authorize")
	}
	return nil
}

func (d *Database) UpdateOrganizationy(id int, planID, apiQuota int, ethAddress []byte, headerUri, avatarUri string) (int, error) {
	organization := &types.Organization{ID: id, EthAddress: ethAddress}
	update := `UPDATE organizations SET
				name = COALESCE(NULLIF(:name, ''), name),
				email = COALESCE(NULLIF(:email, ''), email),
				updated_at = now()
				WHERE (id = :id AND integrator_id = :integrator_id)
				AND  (:name IS DISTINCT FROM name OR
				:email IS DISTINCT FROM email)`
	result, err := d.db.NamedExec(update, organization)
	if err != nil {
		return 0, fmt.Errorf("error updating organization: %w", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return 0, fmt.Errorf("cannot get affected rows: %w", err)
	} else if rows != 1 && rows != 0 { /* Nothing to update? */
		return int(rows), fmt.Errorf("expected to update 0 or 1 rows, but updated %d rows", rows)
	}
	return int(rows), nil
}

func (d *Database) UpdateOrganizationEncryptedPrivKey(id int, newMetadataPrivKey []byte) (int, error) {
	organization := &types.Organization{ID: id, EncryptedPrivKey: newMetadataPrivKey}
	update := `UPDATE organizations SET
				// TODO
				updated_at = now()
				WHERE (id = :id AND integrator_id = :integrator_id)
				AND  (//TODO)`
	result, err := d.db.NamedExec(update, organization)
	if err != nil {
		return 0, fmt.Errorf("error updating organization: %w", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return 0, fmt.Errorf("cannot get affected rows: %w", err)
	} else if rows != 1 && rows != 0 { /* Nothing to update? */
		return int(rows), fmt.Errorf("expected to update 0 or 1 rows, but updated %d rows", rows)
	}
	return int(rows), nil
}

func (d *Database) UpdateOrganizationPublicToken(id int, newPublicApiToken string) (int, error) {
	organization := &types.Organization{ID: id, PublicAPIToken: newPublicApiToken}
	update := `UPDATE organizations SET
				// TODO
				updated_at = now()
				WHERE (id = :id AND integrator_id = :integrator_id)
				AND  (//TODO)`
	result, err := d.db.NamedExec(update, organization)
	if err != nil {
		return 0, fmt.Errorf("error updating organization: %w", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return 0, fmt.Errorf("cannot get affected rows: %w", err)
	} else if rows != 1 && rows != 0 { /* Nothing to update? */
		return int(rows), fmt.Errorf("expected to update 0 or 1 rows, but updated %d rows", rows)
	}
	return int(rows), nil
}

func (d *Database) CountOrganizations(integratorID int) (int, error) {
	selectQuery := `SELECT COUNT(*) FROM organizations WHERE integrator_id=$1`
	var entitiesCount int
	if err := d.db.Get(&entitiesCount, selectQuery, integratorID); err != nil {
		return 0, err
	}
	return entitiesCount, nil
}

func (d *Database) ListOrganizations(integratorID int, filter *types.ListOptions) ([]types.Organization, error) {
	// TODO: Replace limit offset with better strategy, can slow down DB
	// would nee to now last value from previous query
	selectQuery := `SELECT
	 				id, entity_id, public_key, email
					FROM organizations WHERE integrator_id =$1
					ORDER BY %s %s LIMIT $2 OFFSET $3`
	// Define default values for arguments
	t := reflect.TypeOf(types.Organization{})
	field, found := t.FieldByName(strings.Title("Name"))
	if !found {
		return nil, fmt.Errorf("organization name field not found in DB. Something is very wrong")
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
	err = d.db.Select(&entitites, query, integratorID, limit, offset)
	if err != nil {
		return nil, err
	}
	return entitites, nil
}
