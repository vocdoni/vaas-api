package pgsql

import (
	"fmt"
	"time"

	_ "github.com/jackc/pgx/stdlib"

	"go.vocdoni.io/api/types"
)

func (d *Database) CreatePlan(name string, maxCensusSize, maxProcessCount int) (int, error) {
	plan := &types.QuotaPlan{
		Name:            name,
		MaxCensusSize:   maxCensusSize,
		MaxProcessCount: maxProcessCount,
		CreatedUpdated: types.CreatedUpdated{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	// TODO: Calculate EntityID (consult go-dvote)
	insert := `INSERT INTO quota_plans
			( name, max_census_size, max_process_count, created_at, updated_at)
			VALUES ( :name, :max_census_size, :max_process_count,  :created_at, :updated_at)
			RETURNING id`
	result, err := d.db.NamedQuery(insert, plan)
	if err != nil {
		return 0, fmt.Errorf("error creating plan: %w", err)
	}
	if !result.Next() {
		return 0, fmt.Errorf("error creating plan: there is no next result row")
	}
	var id int
	err = result.Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("error creating plan: %w", err)
	}
	return id, nil
}

func (d *Database) GetPlan(id int) (*types.QuotaPlan, error) {
	var plan types.QuotaPlan
	selectplan := `SELECT id, name, max_census_size, max_process_count, created_at, updated_at
						FROM quota_plans WHERE id=$1`
	row := d.db.QueryRowx(selectplan, id)
	err := row.StructScan(&plan)
	if err != nil {
		return nil, err
	}

	return &plan, nil
}

func (d *Database) GetPlanByName(name string) (*types.QuotaPlan, error) {
	var plan types.QuotaPlan
	selectplan := `SELECT id, name, max_census_size, max_process_count, created_at, updated_at
						FROM quota_plans WHERE name=$1`
	row := d.db.QueryRowx(selectplan, name)
	err := row.StructScan(&plan)
	if err != nil {
		return nil, err
	}

	return &plan, nil
}

func (d *Database) DeletePlan(id int) error {
	deleteQuery := `DELETE FROM quota_plans WHERE id = $1`
	result, err := d.db.Exec(deleteQuery, id)
	if err != nil {
		return fmt.Errorf("error deleting plan: %w", err)
	}
	// var rows int64
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error veryfying deleted plan: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("nothing to delete")
	}
	return nil
}

func (d *Database) UpdatePlan(id, newMaxCensusSize, neWMaxProcessCount int, newName string) (int, error) {
	integrator := &types.QuotaPlan{ID: id, Name: newName, MaxCensusSize: newMaxCensusSize, MaxProcessCount: neWMaxProcessCount}
	update := `UPDATE quota_plans SET
				name = COALESCE(NULLIF(:name, ''), name),
				max_process_count = COALESCE(NULLIF(:max_process_count, 0), max_process_count),
				max_census_size = COALESCE(NULLIF(:max_census_size, 0), max_census_size),
				updated_at = now()
				WHERE (id = :id )
				AND  (:name IS DISTINCT FROM name 
					OR :max_process_count IS DISTINCT FROM max_process_count 					
					OR TEXT(:max_census_size) IS DISTINCT FROM TEXT(max_census_size)
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

func (d *Database) GetPlansList() ([]types.QuotaPlan, error) {
	selectQuery := `SELECT * FROM quota_plans`
	var plans []types.QuotaPlan
	if err := d.db.Select(&plans, selectQuery); err != nil {
		return nil, err
	}
	return plans, nil
}
