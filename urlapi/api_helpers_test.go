package urlapi

import (
	"bytes"
	"encoding/hex"
	"testing"

	qt "github.com/frankban/quicktest"
	sk "github.com/vocdoni/blind-csp/saltedkey"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/util"
	"go.vocdoni.io/dvote/crypto/ethereum"
)

// testing non-handler methods

func TestAggregateResults(t *testing.T) {
	type choice struct {
		title  string
		result string
	}
	type question struct {
		choices []choice
	}
	type testProcess struct {
		questions []question
	}
	testProcs := []testProcess{
		// single question, single choice
		{questions: []question{{choices: []choice{{title: "one", result: "0"}}}}},
		// multiple question, single choice
		{questions: []question{{choices: []choice{{title: "one", result: "0"},
			{title: "two", result: "1"}, {title: "three", result: "2"}}}}},
		// multiple question, multiple choice
		{questions: []question{{choices: []choice{{title: "one", result: "0"},
			{title: "two", result: "1"}, {title: "three", result: "2"}}},
			{choices: []choice{{title: "title1", result: "900"}, {title: "title2", result: "10000"},
				{title: "title3", result: "2000"}}}, {choices: []choice{{title: "one", result: "0"}}}}},
	}

	getMetaAndResults := func(proc testProcess) (*types.ProcessMetadata, *types.VochainResults) {
		testMeta := &types.ProcessMetadata{
			Questions: []types.QuestionMeta{},
			Results:   types.ProcessResultsDetails{Aggregation: "discrete-values"},
		}
		testResults := &types.VochainResults{
			Results: [][]string{},
		}
		for _, question := range proc.questions {
			result := []string{}
			questionMeta := types.QuestionMeta{}
			for j, choice := range question.choices {
				result = append(result, choice.result)
				questionMeta.Choices = append(questionMeta.Choices, types.ChoiceMetadata{
					Title: map[string]string{"default": choice.title},
					Value: uint32(j),
				})
			}
			testResults.Results = append(testResults.Results, result)
			testMeta.Questions = append(testMeta.Questions, questionMeta)
		}
		return testMeta, testResults
	}

	for _, proc := range testProcs {
		results, err := aggregateResults(getMetaAndResults(proc))
		qt.Assert(t, err, qt.IsNil)
		for i, result := range results {
			for j, title := range result.Title {
				qt.Assert(t, title, qt.Equals, proc.questions[i].choices[j].title)
			}
			for j, value := range result.Value {
				qt.Assert(t, value, qt.Equals, proc.questions[i].choices[j].result)
			}
		}
	}

	// test failure
	meta1, results1 := getMetaAndResults(testProcs[0])
	// both empty
	_, err := aggregateResults(&types.ProcessMetadata{}, &types.VochainResults{})
	qt.Assert(t, err, qt.IsNotNil)
	// results empty
	_, err = aggregateResults(meta1, &types.VochainResults{})
	qt.Assert(t, err, qt.IsNotNil)
	// metadata empty
	_, err = aggregateResults(&types.ProcessMetadata{}, results1)
	qt.Assert(t, err, qt.IsNotNil)

	meta2, results2 := getMetaAndResults(testProcs[1])
	// meta.Questions longer than results.questions
	_, err = aggregateResults(meta2, results1)
	qt.Assert(t, err, qt.IsNotNil)
	// no need to check longer results.Questions, results can contain extra "values"

	meta3, results3 := getMetaAndResults(testProcs[2])
	// meta choices longer than results choices
	_, err = aggregateResults(meta3, results2)
	qt.Assert(t, err, qt.IsNotNil)
	// meta choices shorter than results choices
	_, err = aggregateResults(meta2, results3)
	qt.Assert(t, err, qt.IsNotNil)
}

func TestAppendProcess(t *testing.T) {
	private := true
	// upcoming process
	electionList := []types.APIElectionSummary{}
	appendProcess(&electionList, &types.Election{
		StartBlock: 200,
	}, private, 199)
	qt.Assert(t, electionList[0].Status, qt.Equals, "UPCOMING")
	// active process
	electionList = []types.APIElectionSummary{}
	appendProcess(&electionList, &types.Election{
		StartBlock: 200,
		EndBlock:   400,
	}, private, 200)
	qt.Assert(t, electionList[0].Status, qt.Equals, "ACTIVE")
	electionList = []types.APIElectionSummary{}
	appendProcess(&electionList, &types.Election{
		StartBlock: 200,
		EndBlock:   400,
	}, private, 399)
	qt.Assert(t, electionList[0].Status, qt.Equals, "ACTIVE")
	// ended process
	electionList = []types.APIElectionSummary{}
	appendProcess(&electionList, &types.Election{
		StartBlock: 200,
		EndBlock:   400,
	}, private, 400)
	qt.Assert(t, electionList[0].Status, qt.Equals, "ENDED")
	// confidential process, private request
	electionList = []types.APIElectionSummary{}
	appendProcess(&electionList, &types.Election{
		Confidential:    true,
		MetadataPrivKey: []byte{0, 1, 2, 3, 4},
	}, private, 0)
	qt.Assert(t, bytes.Compare(electionList[0].MetadataPrivKey, []byte{0, 1, 2, 3, 4}), qt.Equals, 0)

	// confidential process, public request
	private = false
	electionList = []types.APIElectionSummary{}
	appendProcess(&electionList, &types.Election{
		Confidential:    true,
		MetadataPrivKey: []byte{0, 1, 2, 3, 4},
	}, private, 0)
	qt.Assert(t, len(electionList), qt.Equals, 0)
}

func TestReflectElection(t *testing.T) {
	entityId := []byte{1, 2, 3}
	privKey := []byte{4, 5, 6}
	newElection := &types.Election{
		OrgEthAddress:   entityId,
		MetadataPrivKey: privKey,
	}
	priv := reflectElectionPrivate(*newElection)
	qt.Assert(t, bytes.Compare(priv.OrgEthAddress, entityId), qt.Equals, 0)
	qt.Assert(t, bytes.Compare(priv.MetadataPrivKey, privKey), qt.Equals, 0)
	pub := reflectElectionPublic(*newElection)
	qt.Assert(t, bytes.Compare(pub.OrgEthAddress, entityId), qt.Equals, 0)
	qt.Assert(t, bytes.Compare(pub.MetadataPrivKey, []byte{}), qt.Equals, 0)
}

func TestEntityKeyEncryption(t *testing.T) {
	globalKey := []byte("key")
	generateSignKey := func() []byte {
		ethSignKeys := ethereum.NewSignKeys()
		if err := ethSignKeys.Generate(); err != nil {
			t.Fatalf("could not generate sign keys: %v", err)
		}
		_, priv := ethSignKeys.HexString()
		entityPrivKey, err := hex.DecodeString(priv)
		if err != nil {
			t.Fatalf("could not decode sign keys: %v", err)
		}
		return entityPrivKey
	}
	var encryptedKeys [10][]byte
	var rawKeys [10][]byte
	var err error
	// Generate keys
	for i := 0; i < 10; i++ {
		rawKeys[i] = generateSignKey()
		encryptedKeys[i], err = util.EncryptSymmetric(rawKeys[i], globalKey)
		if err != nil {
			t.Fatal(err)
		}
	}
	for i, encryptedKey := range encryptedKeys {
		decryptedKey, err := decryptEntityKeys(encryptedKey, globalKey)
		qt.Assert(t, err, qt.IsNil)
		_, priv := decryptedKey.HexString()
		entityPrivKey, err := hex.DecodeString(priv)
		if err != nil {
			t.Fatalf("could not decode sign keys: %v", err)
		}
		qt.Assert(t, bytes.Compare(entityPrivKey, rawKeys[i]), qt.Equals, 0)
		decryptedKey, err = decryptEntityKeys(rawKeys[i], nil)
		qt.Assert(t, err, qt.IsNil)
		_, priv = decryptedKey.HexString()
		entityPrivKey, err = hex.DecodeString(priv)
		if err != nil {
			t.Fatalf("could not decode sign keys: %v", err)
		}
		qt.Assert(t, bytes.Compare(entityPrivKey, rawKeys[i]), qt.Equals, 0)
	}
}

func TestVerifyCspSharedSignature(t *testing.T) {
	processId, err := hex.DecodeString(
		"954ab8b2006959fcf79bb3cadf1f2018782d9c99c7c8da6b1fecc81de3b161cd")
	if err != nil {
		t.Fatal(err)
	}
	// generate new key pair to use as csp keys
	ethSignKeys := ethereum.NewSignKeys()
	if err := ethSignKeys.Generate(); err != nil {
		t.Fatalf("could not generate sign keys: %v", err)
	}
	// extract public key as hexString, decode
	pub, priv := ethSignKeys.HexString()
	publicKey, err := hex.DecodeString(pub)
	if err != nil {
		t.Fatal(err)
	}

	// create saltable private key
	saltedPrivKey, err := sk.NewSaltedKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	salt := [sk.SaltSize]byte{}
	copy(salt[:], processId)
	// generate salted signature with compressed private key
	signature, err := saltedPrivKey.SignECDSA(salt, processId)
	if err != nil {
		t.Fatal(err)
	}

	// decompress pub key hex
	rootPub, err := ethereum.DecompressPubKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}

	// affirmative case: verify a signature from a key salted with processId
	qt.Assert(t, verifyCspSharedSignature(processId, signature, rootPub), qt.IsNil)

	// failure cases:
	// deformed processId
	qt.Assert(t, verifyCspSharedSignature(
		append([]byte{1, 1, 1, 1, 1}, processId[5:]...), signature, rootPub), qt.IsNotNil)
	// deformed signature
	qt.Assert(t, verifyCspSharedSignature(processId, append(
		[]byte{1, 1, 1, 1, 1}, signature[5:]...), rootPub), qt.IsNotNil)
	// deformed rootPub
	qt.Assert(t, verifyCspSharedSignature(processId, signature,
		append([]byte{1, 1, 1, 1, 1}, rootPub[5:]...)), qt.IsNotNil)
}
