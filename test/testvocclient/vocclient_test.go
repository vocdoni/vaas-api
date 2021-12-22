package testvocclient

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"go.vocdoni.io/api/vocclient"
	"go.vocdoni.io/dvote/crypto/ethereum"
)

var testUrl = "https://api-dev.vocdoni.net"

var testClient *vocclient.Client

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	signer := ethereum.NewSignKeys()
	err := signer.Generate()
	if err != nil {
		fmt.Printf("Error initializiting ethereum signer: %v", err)
		os.Exit(1)
	}
	testClient, err = vocclient.New(testUrl, signer)
	if err != nil {
		fmt.Printf("Error connecting to gateways: %v", err)
		os.Exit(1)
	}
	fmt.Printf("Connected to test endpoint %s\n", testClient.ActiveEndpoint())
	os.Exit(m.Run())
}

func TestBadMethod(t *testing.T) {
	root, err := testClient.GetRoot("0xzzzzzzzz")
	qt.Assert(t, err, qt.IsNotNil)
	qt.Assert(t, len(root) == 0, qt.IsTrue)
}

func TestCurrentBlock(t *testing.T) {
	height, err := testClient.GetCurrentBlock()
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, height > 0, qt.IsTrue)
}

func TestGetprocess(t *testing.T) {
	processList, err := testClient.GetProcessList([]byte{}, "", "", "", 0, false, 0, 100)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, processList, qt.Not(qt.HasLen), 0)
	qt.Assert(t, processList, qt.Not(qt.HasLen), 1)
	pid, err := hex.DecodeString(processList[1])
	qt.Assert(t, err, qt.IsNil)
	process, err := testClient.GetProcess(pid)
	qt.Assert(t, err, qt.IsNil)
	qt.Assert(t, process.EntityID, qt.Not(qt.HasLen), 0)
	qt.Assert(t, process.EndBlock, qt.Not(qt.Equals), 0)
	qt.Assert(t, hex.EncodeToString(process.ID), qt.Equals, hex.EncodeToString(pid))
}
