package pgsql

import (
	"fmt"
	"time"

	_ "github.com/jackc/pgx/stdlib"

	"go.vocdoni.io/api/types"
)

func (d *Database) CreateElection(integratorAPIKey, orgEthAddress, processID []byte, title string, startDate, endDate time.Time, censusID, startBlock, endBlock int, confidential, hiddenResults bool) (int32, error) {
	election := &types.Election{
		OrgEthAddress:    orgEthAddress,
		IntegratorApiKey: integratorAPIKey,
		ProcessID:        processID,
		Title:            title,
		CensusID:         censusID,
		StartDate:        startDate,
		EndDate:          endDate,
		StartBlock:       startBlock,
		EndBlock:         endBlock,
		Confidential:     confidential,
		HiddenResults:    hiddenResults,
		CreatedUpdated: types.CreatedUpdated{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	// TODO: Calculate EntityID (consult go-dvote)
	insert := `INSERT INTO integrators
			( organization_eth_address, integrator_api_key, process_id, title, census_id,
				start_date, end_date, start_block, end_block, confidential, hidden_results, created_at, updated_at)
			VALUES ( :organization_eth_address, :integrator_api_key, :process_id, :title, :census_id,
				:start_date, :end_date, :start_block, :end_block, :confidential, :hidden_results, :created_at, :updated_at)
			RETURNING id`
	result, err := d.db.NamedQuery(insert, election)
	if err != nil {
		return 0, fmt.Errorf("error creating election: %w", err)
	}
	if !result.Next() {
		return 0, fmt.Errorf("error creating organization: there is no next result row")
	}
	var id int32
	err = result.Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("error creating election: %w", err)
	}
	return id, nil
}

func (d *Database) GetElection(integratorAPIKey, orgEthAddress, processID []byte) (*types.Election, error) {
	var election *types.Election
	selectIntegrator := `SELECT title, census_id, start_date, end_date, start_block, end_block, confidential, hidden_results, 
							created_at, updated_at
						FROM elections WHERE organization_eth_address =$1 AND integrator_api_key=$2
									AND process_id=$3`
	row := d.db.QueryRowx(selectIntegrator, orgEthAddress, integratorAPIKey, processID)
	err := row.StructScan(&election)
	if err != nil {
		return nil, err
	}

	return election, nil
}

func (d *Database) GetElectionList(integratorAPIKey, orgEthAddress, processID []byte) ([]types.Election, error) {
	var election []types.Election
	selectIntegrator := `SELECT title, start_date, end_date, start_block, end_block, confidential, hidden_results, 
							created_at, updated_at
						FROM elections WHERE organization_eth_address =$1 AND integrator_api_key=$2`
	err := d.db.Select(&election, selectIntegrator, orgEthAddress, integratorAPIKey, processID)
	if err != nil {
		return nil, err
	}

	return election, nil
}
