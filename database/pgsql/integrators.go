package pgsql

import (
	"fmt"
	"time"

	_ "github.com/jackc/pgx/stdlib"

	"go.vocdoni.io/api/types"
)

func (d *Database) CreateIntegrator(integratorID []byte, info *types.IntegratorInfo) error {
	tx, err := d.db.Beginx()
	if err != nil {
		return fmt.Errorf("cannot initialize postgres transaction: %w", err)
	}
	integrator := &types.Integrator{
		IntegratorInfo: *info,
		ID:             integratorID,
		CreatedUpdated: types.CreatedUpdated{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	// TODO: Calculate EntityID (consult go-dvote)
	insert := `INSERT INTO integrators
			(id, is_authorized, email, name, size, created_at, updated_at)
			VALUES (:id, integrator_id :is_authorized, :email, :name, :size, :created_at, :updated_at)`
	_, err = tx.NamedExec(insert, integrator)
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

func (d *Database) GetIntegrator(integratorID []byte) (*types.Integrator, error) {
	var integrator *types.Integrator
	selectIntegrator := `SELECT id, is_authorized, email, name, size   
						FROM entities WHERE id=$1`
	row := d.db.QueryRowx(selectIntegrator, integratorID)
	err := row.StructScan(&integrator)
	if err != nil {
		return nil, err
	}

	return integrator, nil
}

func (d *Database) DeleteIntegrator(entityID []byte) error {
	if len(entityID) == 0 {
		return fmt.Errorf("invalid arguments")
	}

	deleteQuery := `DELETE FROM entities WHERE id = $1`
	result, err := d.db.Exec(deleteQuery, entityID)
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

func (d *Database) AuthorizeIntegrator(integratorID []byte) error {
	integrator := &types.Integrator{ID: integratorID, IsAuthorized: true}
	update := `UPDATE entities SET
				is_authorized = COALESCE(NULLIF(:is_authorized, false), is_authorized),
				updated_at = now()
				WHERE (id = :id )
				AND  :is_authorized IS DISTINCT FROM is_authorized`
	result, err := d.db.NamedExec(update, integrator)
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

func (d *Database) UpdateIntegrator(entityID []byte, info *types.IntegratorInfo) (int, error) {
	integrator := &types.Integrator{ID: entityID, IntegratorInfo: *info}
	update := `UPDATE entities SET
				name = COALESCE(NULLIF(:name, ''), name),
				email = COALESCE(NULLIF(:email, ''), email),
				updated_at = now()
				WHERE (id = :id )
				AND  (:name IS DISTINCT FROM name OR
				:email IS DISTINCT FROM email)`
	result, err := d.db.NamedExec(update, integrator)
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

func (d *Database) CountIntegrators(integratorID []byte) (int, error) {
	if len(integratorID) == 0 {
		return 0, fmt.Errorf("invalid entity id")
	}
	selectQuery := `SELECT COUNT(*) FROM integrators WHERE integrator_id=$1`
	var count int
	if err := d.db.Get(&count, selectQuery, integratorID); err != nil {
		return 0, err
	}
	return count, nil
}
