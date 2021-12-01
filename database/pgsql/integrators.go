package pgsql

import (
	"fmt"
	"time"

	_ "github.com/jackc/pgx/stdlib"

	"go.vocdoni.io/api/types"
)

func (d *Database) CreateIntegrator(secretApiKey, cspPubKey []byte, name, cspUrlPrefix string) (int32, error) {
	integrator := &types.Integrator{
		CreatedUpdated: types.CreatedUpdated{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	// TODO: Calculate EntityID (consult go-dvote)
	insert := `INSERT INTO integrators
			( secret_api_key, name, csp_pub_key, csp_url_prefix, created_at, updated_at)
			VALUES ( :secret_api_key, :name, :csp_pub_key, :csp_url_prefix, :created_at, :updated_at)
			RETURNING id`
	result, err := d.db.NamedQuery(insert, integrator)
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

func (d *Database) GetIntegrator(secretApiKey []byte) (*types.Integrator, error) {
	var integrator *types.Integrator
	selectIntegrator := `SELECT id, name, csp_url_prefix, csp_pub_key, created_at, updated_at
						FROM integrators WHERE secret_api_key=$1`
	row := d.db.QueryRowx(selectIntegrator, secretApiKey)
	err := row.StructScan(&integrator)
	if err != nil {
		return nil, err
	}

	return integrator, nil
}

func (d *Database) GetIntegratorByID(id int) (*types.Integrator, error) {
	var integrator *types.Integrator
	selectIntegrator := `SELECT secret_api_key, name, csp_url_prefix, csp_pub_key, created_at, updated_at 
						FROM integrators WHERE id=$1`
	row := d.db.QueryRowx(selectIntegrator, id)
	err := row.StructScan(&integrator)
	if err != nil {
		return nil, err
	}

	return integrator, nil
}

func (d *Database) DeleteIntegrator(secretApiKey []byte) error {
	deleteQuery := `DELETE FROM integrators WHERE secret_api_key = $1`
	result, err := d.db.Exec(deleteQuery, secretApiKey)
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

func (d *Database) UpdateIntegrator(secretApiKey, newCspPubKey []byte, newName, newCspUrlPrefix string) (int, error) {
	integrator := &types.Integrator{SecretApiKey: secretApiKey, CspPubKey: newCspPubKey, Name: newName, CspUrlPrefix: newCspUrlPrefix}
	update := `UPDATE integrators SET
				name = COALESCE(NULLIF(:name, ''), name),
				csp_url_prefix = COALESCE(NULLIF(:csp_url_prefix, ''), csp_url_prefix),
				csp_pub_key = COALESCE(NULLIF(:csp_pub_key, '' ::::bytea ),  csp_pub_key),
				secret_api_key = COALESCE(NULLIF(:secret_api_key, '' ::::bytea ),  secret_api_key),
				updated_at = now()
				WHERE (secret_api_key = :secret_api_key )
				AND  (:name IS DISTINCT FROM name 
					OR :csp_url_prefix IS DISTINCT FROM csp_url_prefix 					
					OR TEXT(:csp_pub_key) IS DISTINCT FROM TEXT(csp_pub_key)
					)`
	result, err := d.db.NamedExec(update, integrator)
	if err != nil {
		return 0, fmt.Errorf("error updating integrator: %w", err)
	}
	var rows int64
	if rows, err = result.RowsAffected(); err != nil {
		return 0, fmt.Errorf("cannot get affected rows: %w", err)
	} else if rows != 1 && rows != 0 { /* Nothing to update? */
		return int(rows), fmt.Errorf("expected to update 0 or 1 rows, but updated %d rows", rows)
	}
	return int(rows), nil
}

func (d *Database) UpdateIntegratorApiKey(secretApiKey, newSecretApiKey []byte) (int, error) {
	integrator, err := d.GetIntegrator(secretApiKey)
	if err != nil {
		return 0, fmt.Errorf("error updating integrator: %w", err)
	}
	integrator.SecretApiKey = newSecretApiKey
	update := `UPDATE integrators SET
				secret_api_key = COALESCE(NULLIF(:secret_api_key, '' ::::bytea ),  secret_api_key),
				updated_at = now()
				WHERE (id = :id )
				AND  (TEXT(:secret_api_key) IS DISTINCT FROM TEXT(secret_api_key))`
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

func (d *Database) CountIntegrators() (int, error) {
	selectQuery := `SELECT COUNT(*) FROM integrators`
	var count int
	if err := d.db.Get(&count, selectQuery); err != nil {
		return 0, err
	}
	return count, nil
}
