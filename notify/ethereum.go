package notify

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/vocdoni/go-dvote/chain"
	"gitlab.com/vocdoni/go-dvote/types"
)

var ethereumEventList = []string{
	"ProcessCreated(address,bytes32,string)",
	"ResultsPublished(bytes32,string)",
}

type (
	eventProcessCreated struct {
		EntityAddress [20]byte
		ProcessId     [32]byte // no-lint
		MerkleTree    string
	}
	resultsPublished struct {
		ProcessId [32]byte // no-lint
		Results   string
	}
)

var (
	logProcessCreated       = []byte(ethereumEventList[0])
	logResultsPublished     = []byte(ethereumEventList[1])
	HashLogProcessCreated   = crypto.Keccak256Hash(logProcessCreated)
	HashLogResultsPublished = crypto.Keccak256Hash(logResultsPublished)
)

// ProcessMeta returns the info of a newly created process from the event raised and ethereum storage
func ProcessMeta(contractABI *abi.ABI, eventData []byte, ph *chain.ProcessHandle) (*types.NewProcessTx, error) {
	var eventProcessCreated eventProcessCreated
	err := contractABI.Unpack(&eventProcessCreated, "ProcessCreated", eventData)
	if err != nil {
		return nil, err
	}
	return ph.ProcessTxArgs(eventProcessCreated.ProcessId)
}

// @jordipainan TODO: func ResultsMeta()