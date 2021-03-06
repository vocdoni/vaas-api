package testapi

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	blind "github.com/arnaucube/go-blindsecp256k1"
	qt "github.com/frankban/quicktest"
	"go.vocdoni.io/api/test/testcommon"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/crypto/ethereum"
	dvotetypes "go.vocdoni.io/dvote/types"
	"go.vocdoni.io/dvote/util"
	"go.vocdoni.io/dvote/vochain"
	"go.vocdoni.io/proto/build/go/models"
	"google.golang.org/protobuf/proto"
)

type authReq struct {
	AuthData  []string            `json:"authData,omitempty"`
	TokenR    string              `json:"tokenR,omitempty"`
	Signature dvotetypes.HexBytes `json:"signature,omitempty"`
	Payload   dvotetypes.HexBytes `json:"payload,omitempty"`
	Vote      string              `json:"vote,omitempty"`
	Nullifier dvotetypes.HexBytes `json:"nullifier,omitempty"`
}

func TestGetElectionsPub(t *testing.T) {
	t.Parallel()
	// test get elections (pub)
	for _, election := range testElections {
		var electionResp types.APIElectionInfo
		statusCode := DoRequest(t,
			fmt.Sprintf("%s/v1/pub/elections/%x", API.URL, election.ElectionID),
			testOrganizations[0].APIToken, "GET", types.APIRequest{}, &electionResp)
		if election.Confidential {
			qt.Assert(t, statusCode, qt.Equals, 400)
			break
		}
		qt.Assert(t, statusCode, qt.Equals, 200)
		qt.Assert(t, electionResp.Description, qt.Equals, election.Description)
		qt.Assert(t, electionResp.Title, qt.Equals, election.Title)
		qt.Assert(t, electionResp.Header, qt.Equals, election.Header)
		qt.Assert(t, electionResp.StreamURI, qt.Equals, election.StreamURI)
		qt.Assert(t, electionResp.ProofType, qt.Equals, election.ProofType)
		qt.Assert(t, electionResp.Questions, qt.HasLen, len(election.Questions))
		qt.Assert(t, hex.EncodeToString(electionResp.OrganizationID),
			qt.Equals, hex.EncodeToString(election.OrganizationID))
		qt.Assert(t, electionResp.ElectionID, qt.Not(qt.HasLen), 0)
		for i, question := range electionResp.Questions {
			qt.Assert(t, question.Title, qt.Equals, election.Questions[i].Title)
			qt.Assert(t, question.Description, qt.Equals, election.Questions[i].Description)
			qt.Assert(t, question.Choices, qt.HasLen, len(election.Questions[i].Choices))
			for j, choice := range question.Choices {
				qt.Assert(t, choice.Value, qt.Equals, uint32(j))
				qt.Assert(t, choice.Title, qt.Equals, fmt.Sprintf("Choice%d", j))
			}
		}
	}
}

func TestGetElectionsPriv(t *testing.T) {
	t.Parallel()
	// test get elections (priv)
	for _, election := range testElections {
		cspSignature := fmt.Sprintf("%x",
			testcommon.GetCSPSignature(t, election.ElectionID, API.CSP.CspSignKeys))
		var electionResp types.APIElectionInfo
		statusCode := DoRequest(t,
			fmt.Sprintf("%s/v1/pub/elections/%x/auth/%s", API.URL, election.ElectionID, cspSignature),
			testOrganizations[0].APIToken, "GET", types.APIRequest{}, &electionResp)
		qt.Assert(t, statusCode, qt.Equals, 200)
		qt.Assert(t, electionResp.Description, qt.Equals, election.Description)
		qt.Assert(t, electionResp.Title, qt.Equals, election.Title)
		qt.Assert(t, electionResp.Header, qt.Equals, election.Header)
		qt.Assert(t, electionResp.StreamURI, qt.Equals, election.StreamURI)
		qt.Assert(t, electionResp.ProofType, qt.Equals, election.ProofType)
		qt.Assert(t, electionResp.Questions, qt.HasLen, len(election.Questions))
		qt.Assert(t, hex.EncodeToString(electionResp.OrganizationID),
			qt.Equals, hex.EncodeToString(election.OrganizationID))
		qt.Assert(t, electionResp.ElectionID, qt.Not(qt.HasLen), 0)
		for i, question := range electionResp.Questions {
			qt.Assert(t, question.Title, qt.Equals, election.Questions[i].Title)
			qt.Assert(t, question.Description, qt.Equals, election.Questions[i].Description)
			qt.Assert(t, question.Choices, qt.HasLen, len(election.Questions[i].Choices))
			for j, choice := range question.Choices {
				qt.Assert(t, choice.Value, qt.Equals, uint32(j))
				qt.Assert(t, choice.Title, qt.Equals, fmt.Sprintf("Choice%d", j))
			}
		}
	}
}

func TestVote(t *testing.T) {
	t.Parallel()

	// test signed ecdsa voting
	signedNullifier := submitVoteSigned(t, testActiveElections[0].ElectionID,
		API.CSP.CspSignKeys, testOrganizations[1].APIToken)

	// test blind signature voting
	blindNullifier := submitVoteBlind(t, testActiveElections[0].ElectionID,
		API.CSP.CspSignKeys, testOrganizations[1].APIToken)

	// verify both votes were accepted
	verifyNullifier(t, signedNullifier, testActiveElections[0].ElectionID,
		testOrganizations[1].APIToken)
	verifyNullifier(t, blindNullifier, testActiveElections[0].ElectionID,
		testOrganizations[1].APIToken)
}

func verifyNullifier(t *testing.T, nullifier, processID dvotetypes.HexBytes, orgAPIToken string) {
	var statusCode int
	var resp types.APIResponse
	for i := 0; i < 10; i++ {
		if i > 0 {
			// sleep total of 30 seconds for vote to be confirmed
			time.Sleep(time.Second * 3)
		}
		statusCode = DoRequest(t,
			fmt.Sprintf("%s/v1/pub/nullifiers/%x", API.URL, nullifier),
			orgAPIToken, "GET", types.APIRequest{}, &resp)
		qt.Assert(t, statusCode, qt.Equals, 200)
		// if vote is confirmed, break loop
		if resp.Registered != nil && *resp.Registered {
			break
		}
	}
	qt.Assert(t, *resp.Registered, qt.IsTrue)
	qt.Assert(t, hex.EncodeToString(resp.ElectionID), qt.Equals, hex.EncodeToString(processID))
}

func submitVoteSigned(t *testing.T, processID []byte,
	cspSignKeys *ethereum.SignKeys, orgAPIToken string) dvotetypes.HexBytes {

	voterWallet := ethereum.NewSignKeys()
	err := voterWallet.Generate()
	if err != nil {
		t.Fatal(err)
	}
	signedPID, err := voterWallet.SignEthereum(processID)
	if err != nil {
		t.Fatal(err)
	}

	// fetch tokenR from CSP
	req := authReq{AuthData: []string{hex.EncodeToString(signedPID)}}
	var aResp authReq
	statusCode := DoRequest(t, fmt.Sprintf("http://%s:%d%s/%x/ecdsa/auth",
		testcommon.TestHost, testcommon.TestCSPPort, testcommon.TestCSPPath,
		processID), orgAPIToken, "POST", req, &aResp)
	qt.Assert(t, statusCode, qt.Equals, 200)

	// fetch non-blind signature from csp
	caBundle := &models.CAbundle{ProcessId: processID, Address: voterWallet.Address().Bytes()}
	hexCaBundle, err := proto.Marshal(caBundle)
	if err != nil {
		t.Fatal(err)
	}
	req = authReq{TokenR: aResp.TokenR, Payload: hexCaBundle}
	statusCode = DoRequest(t, fmt.Sprintf("http://%s:%d%s/%x/ecdsa/sign",
		testcommon.TestHost, testcommon.TestCSPPort,
		testcommon.TestCSPPath, processID), orgAPIToken, "POST", req, &aResp)
	qt.Assert(t, statusCode, qt.Equals, 200)

	// create and submit vote package with proof
	vote := &vochain.VotePackage{
		Nonce: util.RandomHex(32),
		Votes: []int{1},
	}
	voteBytes, err := json.Marshal(vote)
	if err != nil {
		t.Fatal(err)
	}

	voteTx := &models.Tx{
		Payload: &models.Tx_Vote{
			Vote: &models.VoteEnvelope{
				Nonce:     util.RandomBytes(32),
				ProcessId: processID,
				Proof: &models.Proof{
					Payload: &models.Proof_Ca{
						Ca: &models.ProofCA{
							Type:      models.ProofCA_ECDSA_PIDSALTED,
							Bundle:    caBundle,
							Signature: aResp.Signature,
						},
					},
				},
				VotePackage: voteBytes,
			},
		},
	}

	signedVoteTx := &models.SignedTx{}

	signedVoteTx.Tx, err = proto.Marshal(voteTx)
	if err != nil {
		t.Fatal(err)
	}
	signedVoteTx.Signature, err = voterWallet.SignVocdoniTx(signedVoteTx.Tx, "")
	if err != nil {
		t.Fatal(err)
	}

	signedVoteTxBytes, err := proto.Marshal(signedVoteTx)
	if err != nil {
		t.Fatal(err)
	}
	req = authReq{Vote: base64.StdEncoding.EncodeToString(signedVoteTxBytes)}
	statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/pub/elections/%x/vote", API.URL, processID),
		orgAPIToken, "POST", req, &aResp)
	qt.Assert(t, statusCode, qt.Equals, 200)
	t.Logf("submitted vote with nullifier %x", aResp.Nullifier)
	qt.Assert(t, aResp.Nullifier, qt.Not(qt.HasLen), 0)
	return aResp.Nullifier
}

func submitVoteBlind(t *testing.T, processID []byte,
	cspSignKeys *ethereum.SignKeys, orgAPIToken string) dvotetypes.HexBytes {
	voterWallet := ethereum.NewSignKeys()
	err := voterWallet.Generate()
	if err != nil {
		t.Fatal(err)
	}
	signedPID, err := voterWallet.SignEthereum(processID)
	if err != nil {
		t.Fatal(err)
	}

	// fetch tokenR from CSP
	req := authReq{AuthData: []string{hex.EncodeToString(signedPID)}}
	var aResp authReq
	statusCode := DoRequest(t, fmt.Sprintf("http://%s:%d%s/%x/blind/auth",
		testcommon.TestHost, testcommon.TestCSPPort, testcommon.TestCSPPath,
		processID), orgAPIToken, "POST", req, &aResp)
	qt.Assert(t, statusCode, qt.Equals, 200)

	hexTokenR, err := hex.DecodeString(aResp.TokenR)
	if err != nil {
		t.Fatal(err)
	}
	// get blind point from tokenR
	blindPoint, err := blind.NewPointFromBytesUncompressed(hexTokenR)
	if err != nil {
		t.Fatal(err)
	}

	// create CA bundle and blind it
	caBundle := &models.CAbundle{ProcessId: processID, Address: voterWallet.Address().Bytes()}
	hexCaBundle, err := proto.Marshal(caBundle)
	if err != nil {
		t.Fatal(err)
	}
	hexBlinded, userSecretData, err := blind.Blind(
		new(big.Int).SetBytes(ethereum.HashRaw(hexCaBundle)), blindPoint)
	if err != nil {
		t.Fatal(err)
	}

	req = authReq{TokenR: aResp.TokenR, Payload: hexBlinded.Bytes()}
	statusCode = DoRequest(t, fmt.Sprintf("http://%s:%d%s/%x/blind/sign",
		testcommon.TestHost, testcommon.TestCSPPort,
		testcommon.TestCSPPath, processID), orgAPIToken, "POST", req, &aResp)
	qt.Assert(t, statusCode, qt.Equals, 200)

	// unblind received signature with the saved userSecretData
	unblindedSignature := blind.Unblind(new(big.Int).SetBytes(aResp.Signature), userSecretData)

	// create and submit vote package with proof
	vote := &vochain.VotePackage{
		Nonce: util.RandomHex(32),
		Votes: []int{1},
	}
	voteBytes, err := json.Marshal(vote)
	if err != nil {
		t.Fatal(err)
	}

	voteTx := &models.Tx{
		Payload: &models.Tx_Vote{
			Vote: &models.VoteEnvelope{
				Nonce:     util.RandomBytes(32),
				ProcessId: processID,
				Proof: &models.Proof{
					Payload: &models.Proof_Ca{
						Ca: &models.ProofCA{
							Type:      models.ProofCA_ECDSA_BLIND_PIDSALTED,
							Bundle:    caBundle,
							Signature: unblindedSignature.BytesUncompressed(),
						},
					},
				},
				VotePackage: voteBytes,
			},
		},
	}

	signedVoteTx := &models.SignedTx{}

	signedVoteTx.Tx, err = proto.Marshal(voteTx)
	if err != nil {
		t.Fatal(err)
	}
	signedVoteTx.Signature, err = voterWallet.SignVocdoniTx(signedVoteTx.Tx, "")
	if err != nil {
		t.Fatal(err)
	}

	signedVoteTxBytes, err := proto.Marshal(signedVoteTx)
	if err != nil {
		t.Fatal(err)
	}
	req = authReq{Vote: base64.StdEncoding.EncodeToString(signedVoteTxBytes)}
	statusCode = DoRequest(t,
		fmt.Sprintf("%s/v1/pub/elections/%x/vote", API.URL, processID),
		orgAPIToken, "POST", req, &aResp)
	qt.Assert(t, statusCode, qt.Equals, 200)
	t.Logf("submitted vote with nullifier %x", aResp.Nullifier)
	qt.Assert(t, aResp.Nullifier, qt.Not(qt.HasLen), 0)
	return aResp.Nullifier
}
