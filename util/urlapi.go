package util

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/util"
)

func UnmarshalRequest(ctx *httprouter.HTTPContext) (req types.APIRequest, err error) {
	var bytes []byte
	bytes, err = ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, &req)
	if err != nil {
		return req, fmt.Errorf("could not decode request body: %v", err)
	}
	return
}

func GetOrganizationID(ctx *httprouter.HTTPContext) ([]byte, error) {
	organization := ctx.URLParam("entityID")
	organizationID, err := hex.DecodeString(util.TrimHex(organization))
	if err != nil {
		return nil, fmt.Errorf("could not parse urlParam EntityID")
	}
	return organizationID, nil
}

func GetID(ctx *httprouter.HTTPContext) (int, error) {
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
