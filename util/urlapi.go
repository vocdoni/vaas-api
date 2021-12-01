package util

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/httprouter"
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

func GetEntityID(ctx *httprouter.HTTPContext) ([]byte, error) {
	entity := ctx.URLParam("entityID")
	entityID, err := hex.DecodeString(util.TrimHex(entity))
	if err != nil {
		return nil, fmt.Errorf("could not parse urlParam EntityID")
	}
	return entityID, nil
}

func GetID(ctx *httprouter.HTTPContext) (int, error) {
	id := ctx.URLParam("id")
	intID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("could not parse urlParam ID: %v", err)
	}
	return intID, nil
}

func SendResponse(response types.APIResponse, ctx *httprouter.HTTPContext) error {
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}
	if err = ctx.Send(data); err != nil {
		return err
	}
	return nil
}
