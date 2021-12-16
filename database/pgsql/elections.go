package pgsql

import (
	"fmt"
	"time"

	_ "github.com/jackc/pgx/stdlib"

	"go.vocdoni.io/api/types"
)

func (d *Database) CreateElection(election types.Election) (int, error) {

	election.CreatedUpdated = types.CreatedUpdated{
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// TODO: Calculate EntityID (consult go-dvote)
	insert := `INSERT INTO elections
			( organization_eth_address, integrator_api_key, process_id, title, census_id,
				start_date, end_date, start_block, end_block, confidential, hidden_results,
				json_metadata_bytes, json_metadata_hash, created_at, updated_at)
			VALUES ( :organization_eth_address, :integrator_api_key, :process_id, :title, :census_id,
				:start_date, :end_date, :start_block, :end_block, :confidential, :hidden_results,
				:json_metadata_bytes, :json_metadata_hash, :created_at, :updated_at)
			RETURNING id`
	result, err := d.db.NamedQuery(insert, election)
	if err != nil {
		return 0, fmt.Errorf("error creating election: %v", err)
	}
	if !result.Next() {
		return 0, fmt.Errorf("error creating election: there is no next result row")
	}
	var id int
	err = result.Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("error creating election: %v", err)
	}
	return id, nil
}

func (d *Database) GetElectionPublic(organizationEthAddress, processID []byte) (*types.Election, error) {
	var election types.Election
	selectIntegrator := `SELECT title, start_date, end_date, start_block, end_block, confidential, hidden_results,
							json_metadata_bytes, json_metadata_hash
						FROM elections WHERE organization_eth_address=$1 AND process_id=$2`
	row := d.db.QueryRowx(selectIntegrator, organizationEthAddress, processID)
	return &election, row.StructScan(&election)
}

func (d *Database) GetElection(integratorAPIKey, orgEthAddress, processID []byte) (*types.Election, error) {
	var election types.Election
	selectIntegrator := `SELECT title, census_id, start_date, end_date, start_block, end_block, confidential, hidden_results, 
							json_metadata_bytes, json_metadata_hash, created_at, updated_at
						FROM elections WHERE organization_eth_address =$1 AND integrator_api_key=$2
									AND process_id=$3`
	row := d.db.QueryRowx(selectIntegrator, orgEthAddress, integratorAPIKey, processID)
	err := row.StructScan(&election)
	if err != nil {
		return nil, err
	}

	return &election, nil
}

func (d *Database) ListElections(integratorAPIKey, orgEthAddress []byte) ([]types.Election, error) {
	var election []types.Election
	selectIntegrator := `SELECT title, start_date, end_date, start_block, end_block, confidential, hidden_results, 
							created_at, updated_at
						FROM elections WHERE organization_eth_address =$1 AND integrator_api_key=$2`
	return election, d.db.Select(&election, selectIntegrator, orgEthAddress, integratorAPIKey)
}
