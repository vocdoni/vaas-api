package pgsql

import (
	"fmt"
	"math/big"
	"time"

	_ "github.com/jackc/pgx/stdlib"

	"go.vocdoni.io/api/types"
)

func (d *Database) CreateElection(integratorAPIKey, orgEthAddress, processID []byte, title string, censusID int, startBlock, endBlock big.Int, confidential, hiddenResults bool) (int32, error) {
	election := &types.Election{
		OrgEthAddress:    orgEthAddress,
		IntegratorApiKey: integratorAPIKey,
		ProcessID:        processID,
		Title:            title,
		CensusID:         censusID,
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
				start_block, end_block, confidential, hidden_result, created_at, updated_at)
			VALUES ( :organization_eth_address, :integrator_api_key, :process_id, :title, :census_id,
				:start_block, :end_block, confidential, :hidden_result, :created_at, :updated_at)
			RETURNING id`
	result, err := d.db.NamedQuery(insert, election)
	if err != nil || !result.Next() {
		return 0, fmt.Errorf("error creating election: %w", err)
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
	selectIntegrator := `SELECT title, census_id, start_block, end_block, confidential, hidden_result, 
							created_at, updated_at
						FROM integrators WHERE organization_eth_address =$1 AND integrator_api_key 
									AND process_id=$2`
	row := d.db.QueryRowx(selectIntegrator, orgEthAddress, integratorAPIKey, processID)
	err := row.StructScan(&election)
	if err != nil {
		return nil, err
	}

	return election, nil
}
