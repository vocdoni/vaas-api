package vocclient

import (
	"fmt"
	"strings"

	"go.vocdoni.io/dvote/client"
	"go.vocdoni.io/dvote/log"
)

func DiscoverGateway(url string) (*client.Client, error) {
	log.Debugf("discovering gateway %s", url)
	if !strings.HasSuffix(url, "/dvote") {
		url = url + "/dvote"
	}
	client, err := client.New(url)
	if err != nil {
		return nil, fmt.Errorf("could not connect to gateway %s: %v", url, err)
	}
	return client, nil
}
