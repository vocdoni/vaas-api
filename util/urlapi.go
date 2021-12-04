package util

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/util"
)

func UnmarshalRequest(msg *bearerstdapi.BearerStandardAPIdata) (req types.APIRequest, err error) {
	err = json.Unmarshal(msg.Data, &req)
	if err != nil {
		return req, fmt.Errorf("could not decode request body %s: %v", string(msg.Data), err)
	}
	// Ensure all values are non-nil
	if req.CspPubKey == nil {
		req.CspPubKey = []byte{}
	}
	if req.Questions == nil {
		req.Questions = []types.Question{}
	}
	return
}

func GetBytesID(ctx *httprouter.HTTPContext) ([]byte, error) {
	organization := ctx.URLParam("id")
	organizationID, err := hex.DecodeString(util.TrimHex(organization))
	if err != nil {
		return nil, fmt.Errorf("could not parse urlParam EntityID")
	}
	return organizationID, nil
}

func GetIntID(ctx *httprouter.HTTPContext) (int, error) {
	id := ctx.URLParam("id")
	intID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("could not parse urlParam ID: %v", err)
	}
	return intID, nil
}

func GetAuthToken(msg *bearerstdapi.BearerStandardAPIdata) (token []byte, err error) {
	if token, err = hex.DecodeString(msg.AuthToken); err != nil {
		return []byte{}, fmt.Errorf("could not decode auth token: %v", err)
	}
	return token, nil
}
