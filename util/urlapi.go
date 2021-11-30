package util

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"go.vocdoni.io/api/types"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/util"
)

func UnmarshalRequest(ctx *httprouter.HTTPContext) (req *types.APIRequest, err error) {
	var bytes []byte
	bytes, err = ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, req)
	return
}

func GetEntityID(ctx *httprouter.HTTPContext) ([]byte, error) {
	entity := ctx.URLParam("entityID")
	return hex.DecodeString(util.TrimHex(entity))
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
