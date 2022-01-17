package testpgsql

import (
	"math/rand"
	"os"
	"strconv"
	"testing"

	"go.vocdoni.io/api/config"
	"go.vocdoni.io/api/test/testcommon"
	"go.vocdoni.io/dvote/log"
)

var API testcommon.TestAPI

func TestMain(m *testing.M) {
	API = testcommon.TestAPI{Port: 12000 + rand.Intn(1000)}
	// check for TEST_DB_HOST env var. If not exist, don't run the db tests
	dbHost := os.Getenv("TEST_DB_HOST")
	if dbHost == "" {
		log.Infof("SKIPPING: database host not set")
		return
	}
	dbPort, err := strconv.Atoi(os.Getenv("TEST_DB_PORT"))
	if err != nil {
		dbPort = 5432
	}
	db := &config.DB{
		Dbname:   "postgres",
		Password: "postgres",
		Host:     dbHost,
		Port:     dbPort,
		Sslmode:  "disable",
		User:     "postgres",
	}
	if err := API.Start(db, "", "", "", 9000); err != nil {
		log.Infof("SKIPPING: could not start the API: %v", err)
		return
	}
	if err := API.DB.Ping(); err != nil {
		log.Infof("SKIPPING: could not connect to DB: %v", err)
		return
	}
	os.Exit(m.Run())
}
