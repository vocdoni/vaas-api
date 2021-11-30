package pgsql

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"time"

	_ "github.com/jackc/pgx/stdlib"

	"go.vocdoni.io/api/types"
)

func (d *Database) CreateEntity(integratorID, entityID []byte, info *types.EntityInfo) error {
	tx, err := d.db.Beginx()
	if err != nil {
		return fmt.Errorf("cannot initialize postgres transaction: %w", err)
	}
	entity := &types.Entity{
		EntityInfo:   *info,
		ID:           entityID,
		IntegratorID: integratorID,
		CreatedUpdated: types.CreatedUpdated{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	// TODO: Calculate EntityID (consult go-dvote)
	insert := `INSERT INTO entities
			(id, integrator_id is_authorized, email, name, size, created_at, updated_at)
			VALUES (:id, integrator_id :is_authorized, :email, :name, :size, :created_at, :updated_at)`
	_, err = tx.NamedExec(insert, entity)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("cannot perform db rollback %v after error %w", rollbackErr, err)
		}
		return fmt.Errorf("cannot add insert query in the transaction: %w", err)
	}
	if err := tx.Commit(); err != nil {
		if rollErr := tx.Rollback(); err != nil {
			return fmt.Errorf("something is very wrong: error rolling back: %v after final commit to DB: %w", rollErr, err)
		}
		return fmt.Errorf("error commiting transactions to the DB: %w", err)
	}
	return nil
}

func (d *Database) GetEntity(integratorID, entityID []byte) (*types.Entity, error) {
	var entity *types.Entity
	selectEntity := `SELECT id, is_authorized, email, name, type, size, callback_url, callback_secret, census_managers_addresses as "pg_census_managers_addresses"  
						FROM entities WHERE id=$1`
	row := d.db.QueryRowx(selectEntity, entityID)
	err := row.StructScan(&entity)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (d *Database) DeleteEntity(integratorID, entityID []byte) error {
	if len(entityID) == 0 {
		return fmt.Errorf("invalid arguments")
	}

	deleteQuery := `DELETE FROM entities WHERE id = $1 AND integrator_id = $2`
	result, err := d.db.Exec(deleteQuery, entityID, integratorID)
	if err != nil {
		return fmt.Errorf("error deleting entity: %w", err)
	}
	// var rows int64
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error veryfying deleted entity: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("nothing to delete")
	}
	return nil
}

// EntitiesID returns all the entities ID's
func (d *Database) EntitiesID() ([]string, error) {
	var entitiesIDs [][]byte
	entitiesQuery := `SELECT id FROM entities`
	err := d.db.Select(&entitiesIDs, entitiesQuery)
	if err != nil {
		return nil, err
	}
	entities := []string{}
	for _, e := range entitiesIDs {
		entities = append(entities, hex.EncodeToString(e))
	}
	return entities, nil
}

func (d *Database) AuthorizeEntity(integratorID, entityID []byte) error {
	entity := &types.Entity{ID: entityID, IntegratorID: integratorID, IsAuthorized: true}
	update := `UPDATE entities SET
				is_authorized = COALESCE(NULLIF(:is_authorized, false), is_authorized),
				updated_at = now()
				WHERE (id = :id  AND integrator_id = :integrator_id)
				AND  :is_authorized IS DISTINCT FROM is_authorized`
	result, err := d.db.NamedExec(update, entity)
	if err != nil {
		return fmt.Errorf("error updating entity: %w", err)
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

func (d *Database) UpdateEntity(integratorID, entityID []byte, info *types.EntityInfo) (int, error) {
	entity := &types.Entity{ID: entityID, IntegratorID: integratorID, EntityInfo: *info}
	update := `UPDATE entities SET
				name = COALESCE(NULLIF(:name, ''), name),
				email = COALESCE(NULLIF(:email, ''), email),
				updated_at = now()
				WHERE (id = :id AND integrator_id = :integrator_id)
				AND  (:name IS DISTINCT FROM name OR
				:email IS DISTINCT FROM email)`
	result, err := d.db.NamedExec(update, entity)
	if err != nil {
		return 0, fmt.Errorf("error updating entity: %w", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return 0, fmt.Errorf("cannot get affected rows: %w", err)
	} else if rows != 1 && rows != 0 { /* Nothing to update? */
		return int(rows), fmt.Errorf("expected to update 0 or 1 rows, but updated %d rows", rows)
	}
	return int(rows), nil
}

func (d *Database) CountEntities(integratorID []byte) (int, error) {
	if len(integratorID) == 0 {
		return 0, fmt.Errorf("invalid entity id")
	}
	selectQuery := `SELECT COUNT(*) FROM entities WHERE integrator_id=$1`
	var entitiesCount int
	if err := d.db.Get(&entitiesCount, selectQuery, integratorID); err != nil {
		return 0, err
	}
	return entitiesCount, nil
}

func (d *Database) ListEntities(integratorID []byte, filter *types.ListOptions) ([]types.Entity, error) {
	// TODO: Replace limit offset with better strategy, can slow down DB
	// would nee to now last value from previous query
	selectQuery := `SELECT
	 				id, entity_id, public_key, email
					FROM entities WHERE integrator_id =$1
					ORDER BY %s %s LIMIT $2 OFFSET $3`
	// Define default values for arguments
	t := reflect.TypeOf(types.EntityInfo{})
	field, found := t.FieldByName(strings.Title("Name"))
	if !found {
		return nil, fmt.Errorf("entity name field not found in DB. Something is very wrong")
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
	var entitites []types.Entity
	err = d.db.Select(&entitites, query, integratorID, limit, offset)
	if err != nil {
		return nil, err
	}
	return entitites, nil
}