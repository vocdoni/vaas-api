package testpgsql

import (
	"log"
	"math/rand"
	"os"
	"testing"

	"go.vocdoni.io/api/config"
	"go.vocdoni.io/api/test/testcommon"
)

var API testcommon.TestAPI

func TestMain(m *testing.M) {
	API = testcommon.TestAPI{Port: 12000 + rand.Intn(1000)}
	db := &config.DB{
		Dbname:   "vaas",
		Password: "vocdoni",
		Host:     "127.0.0.1",
		Port:     5432,
		Sslmode:  "disable",
		User:     "vocdoni",
	}
	if err := API.Start(db, "", "", nil, 9000, testcommon.TestCSP{}); err != nil {
		log.Printf("SKIPPING: could not start the API: %v", err)
		return
	}
	os.Exit(m.Run())
	if err := API.DB.Ping(); err != nil {
		log.Printf("SKIPPING: could not connect to DB: %v", err)
		return
	}
	os.Exit(m.Run())
}
