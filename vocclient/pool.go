package vocclient

import (
	"fmt"

	"go.vocdoni.io/dvote/api"
	"go.vocdoni.io/dvote/client"
	"go.vocdoni.io/dvote/crypto/ethereum"
)

// gateway client wrapper
type Gateway struct {
	client        *client.Client
	health        int32
	supportedApis []string
}

// list of clients, enables sorting by health
type GatewayPool []Gateway

func (pool GatewayPool) Len() int           { return len(pool) }
func (pool GatewayPool) Less(i, j int) bool { return pool[i].health > pool[j].health }
func (pool GatewayPool) Swap(i, j int)      { pool[i], pool[j] = pool[j], pool[i] }

func (pool GatewayPool) activeGateway() (Gateway, error) {
	if len(pool) == 0 || pool[0].client == nil {
		return Gateway{}, fmt.Errorf("no gateways available")
	}
	return (pool)[0], nil
}

func (pool *GatewayPool) shift() {
	if pool == nil || len(*pool) < 2 {
		return
	}
	*pool = append((*pool)[1:], (*pool)[0])
}

func (pool *GatewayPool) Request(req api.APIrequest, signer *ethereum.SignKeys) (resp *api.APIresponse, err error) {
	gw, err := pool.activeGateway()
	if err != nil {
		return nil, fmt.Errorf("could not make request %s: %v", req.Method, err)
	}
	return gw.client.Request(req, signer)
}
