package testdb

import (
	"encoding/hex"
	"fmt"

	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"go.vocdoni.io/api/types"
)

var Signers = []struct {
	Pub  string
	Priv string
}{
	// User() no rows
	{"03f9d1e41906436bf2e8aab319383dea6a4c06426266955293fd92b41f6346256f", "1c1c5c24be0d76e5f7c853902e9e23ced013a597aca7573861c8cd0a160ca375"},
	// user no rows failed AddUser()
	{"03733ca0d2462ef3cd4dbd331d5ec27a63eeb13afbaf03f236847479c3e8d7fd94", "1c1c5c24be0d76e5f7c853902e9e23ced013a597aca7573861c8cd0a160ca355"},
	// failed User()
	{"0399d0ad8447520e66df7db954b0936f4b141a01ba6213dda88c9df7293b66262e", "1c1c5c24be0d76e5f7c853902e9e23ced013a597aca7573861c8cd0a160ca357"},
	// MemberPubKey() no rows
	{"026163a9bc3425426bbb7f0fde6c9bb4504493415a34b99a84162fe01640a784a3", "1c1c5c24be0d76e5f7c853902e9e23ced013a597aca7573861c8cd0a160ca372"},
}

type Database struct {
}

func New() (*Database, error) {
	return &Database{}, nil
}

func (d *Database) Ping() error {
	return nil
}

func (d *Database) Close() error {
	return nil
}

func (d *Database) Entity(entityID []byte) (*types.Organization, error) {
	if fmt.Sprintf("%x", entityID) == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return nil, fmt.Errorf("error adding entity with id: %x", entityID)
	}
	if fmt.Sprintf("%x", entityID) == "f6da3e4864d566faf82163a407e84a9001592678" {
		return nil, fmt.Errorf("cannot fetch entity with ID: %x", entityID)
	}

	entity := types.Organization{}
	// entity.ID = entityID
	// entity.Name = "test entity"
	// entity.Email = "entity@entity.org"

	// failEidID := hex.EncodeToString(entityID)
	// if failEidID == "ca526af2aaa0f3e9bb68ab80de4392590f7b153a" {
	// 	entity.ID = []byte{1}
	// }

	return &entity, nil
}

func (d *Database) EntitiesID() ([]string, error) {
	return nil, nil
}

func (d *Database) AddEntity(entityID []byte) error {
	if fmt.Sprintf("%x", entityID) == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return fmt.Errorf("error adding entity with id: %x", entityID)
	}
	return nil
}

func (d *Database) DeleteEntity(entityID []byte) error {
	if fmt.Sprintf("%x", entityID) == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return fmt.Errorf("error deleting entity with id: %x", entityID)
	}
	return nil
}

func (d *Database) UpdateEntity(entityID []byte) (int, error) {
	failEid := hex.EncodeToString(entityID)
	if failEid == "09fa012e40f844b073fab7fcbd7f7a5716c1a365" {
		return 0, fmt.Errorf("error updating entity with id: %s", failEid)
	}
	return 1, nil
}

func (d *Database) Migrate(dir migrate.MigrationDirection) (int, error) {
	return 0, nil
}

func (d *Database) MigrateStatus() (int, int, string, error) {
	return 0, 0, "", nil
}

func (d *Database) MigrationUpSync() (int, error) {
	return 0, nil
}
