package transactions

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	dvotedb "go.vocdoni.io/dvote/db"
	"go.vocdoni.io/dvote/db/metadb"
	"go.vocdoni.io/dvote/util"
)

var kv *TxCacheDB

func TestMain(m *testing.M) {
	storage, err := ioutil.TempDir("", ".transactions-test")
	if err != nil {
		log.Fatal(err)
	}
	db, err := metadb.New(dvotedb.TypePebble, filepath.Join(storage, "metadb"))
	if err != nil {
		log.Fatal(err)
	}
	kv = NewTxKv(db)
	code := m.Run()
	if err := os.RemoveAll(storage); err != nil {
		log.Fatal(err)
	}
	os.Exit(code)
}

var integratorPrivKey []byte

func TestStoreTx(t *testing.T) {
	t.Parallel()
	integratorPrivKey = util.RandomBytes(32)
	var hashes [][]byte
	for i := 0; i < 10; i++ {
		query := SerializableTx{
			Body:         nil,
			CreationTime: time.Now(),
		}
		if i%3 == 0 {
			query.Type = CreateElection
			query.Body = CreateElectionTx{
				IntegratorPrivKey: integratorPrivKey,
				Title:             "new election",
				Confidential:      true,
			}
		} else if i%3 == 1 {
			query.Type = CreateOrganization
			query.Body = CreateOrganizationTx{
				IntegratorPrivKey: integratorPrivKey,
				PublicAPIToken:    "token",
				HeaderURI:         "header",
				AvatarURI:         "avatar",
			}
		} else {
			query.Type = UpdateOrganization
			query.Body = UpdateOrganizationTx{
				IntegratorPrivKey: integratorPrivKey,
				HeaderUri:         "updateheader",
				AvatarUri:         "updateavatar",
			}
		}
		hash := util.RandomBytes(32)
		hashes = append(hashes, hash)
		qt.Assert(t, kv.StoreTx(hash, query), qt.IsNil)
	}

	for i, hash := range hashes {
		tx, err := kv.GetTx(hash)
		qt.Assert(t, err, qt.IsNil)
		if i%3 == 0 {
			testGetElection(t, tx.Type, CreateElection, tx.Body)
		} else if i%3 == 1 {
			testGetElection(t, tx.Type, CreateOrganization, tx.Body)
		} else {
			testGetElection(t, tx.Type, UpdateOrganization, tx.Body)
		}
	}

	for _, hash := range hashes {
		qt.Assert(t, kv.DeleteTx(hash), qt.IsNil)
		tx, err := kv.GetTx(hash)
		qt.Assert(t, err, qt.IsNil)
		qt.Assert(t, tx, qt.IsNil)
	}
}

func TestStoreTxTime(t *testing.T) {
	t.Parallel()
	var hashes [][]byte
	var times []time.Time
	for i := 0; i < 10; i++ {
		hash := util.RandomBytes(32)
		time := time.Now()
		hashes = append(hashes, hash)
		times = append(times, time)
		qt.Assert(t, kv.StoreTxTime(hash, time), qt.IsNil)
	}

	for i, hash := range hashes {
		time, err := kv.GetTxTime(hash)
		qt.Assert(t, err, qt.IsNil)
		qt.Assert(t, time.Equal(times[i]), qt.IsTrue)
	}
}

func testGetElection(t *testing.T, Type SerializableTxType,
	expected SerializableTxType, tx TxBody) {
	qt.Assert(t, Type, qt.Equals, expected)
	switch Type {
	case CreateElection:
		query, ok := tx.(CreateElectionTx)
		qt.Assert(t, ok, qt.IsTrue)
		qt.Assert(t, bytes.Compare(query.IntegratorPrivKey, integratorPrivKey), qt.Equals, 0)
		qt.Assert(t, query.Title, qt.Equals, "new election")
		qt.Assert(t, query.Confidential, qt.IsTrue)
	case CreateOrganization:
		query, ok := tx.(CreateOrganizationTx)
		qt.Assert(t, ok, qt.IsTrue)
		qt.Assert(t, bytes.Compare(query.IntegratorPrivKey, integratorPrivKey), qt.Equals, 0)
		qt.Assert(t, query.PublicAPIToken, qt.Equals, "token")
		qt.Assert(t, query.HeaderURI, qt.Equals, "header")
		qt.Assert(t, query.AvatarURI, qt.Equals, "avatar")
	case UpdateOrganization:
		query, ok := tx.(UpdateOrganizationTx)
		qt.Assert(t, ok, qt.IsTrue)
		qt.Assert(t, bytes.Compare(query.IntegratorPrivKey, integratorPrivKey), qt.Equals, 0)
		qt.Assert(t, query.HeaderUri, qt.Equals, "updateheader")
		qt.Assert(t, query.AvatarUri, qt.Equals, "updateavatar")
	default:
		t.Fail()
	}
}
