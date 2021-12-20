package vocclient

import (
	"fmt"
	"strings"

	"go.vocdoni.io/dvote/api"
	"go.vocdoni.io/dvote/client"
	"go.vocdoni.io/dvote/log"
)

func DiscoverGateway(url string) (*Gateway, error) {
	log.Debugf("discovering gateway %s", url)
	if !strings.HasSuffix(url, "/dvote") {
		url = url + "/dvote"
	}
	client, err := client.New(url)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to gateway %s: %v", url, err)
	}
	resp, err := client.Request(api.APIrequest{Method: "getInfo"}, nil)
	if err != nil {
		return nil, fmt.Errorf("could not get info for %s: %v", client.Addr, err)
	}
	return &Gateway{
		client:        client,
		health:        resp.Health,
		supportedApis: resp.APIList,
	}, nil
}
