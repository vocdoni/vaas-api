package urlapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.vocdoni.io/api/config"
	"go.vocdoni.io/api/database"
	"go.vocdoni.io/api/database/transactions"
	"go.vocdoni.io/api/types"
	"go.vocdoni.io/api/vocclient"
	dvotedb "go.vocdoni.io/dvote/db"
	"go.vocdoni.io/dvote/httprouter"
	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/metrics"
)

const apiVersion = "v1"

var txTimeout time.Duration = time.Minute

type URLAPI struct {
	PrivateCalls uint64
	PublicCalls  uint64
	BaseRoute    string

	config                *config.API
	globalOrganizationKey []byte
	globalMetadataKey     []byte
	router                *httprouter.HTTProuter
	api                   *bearerstdapi.BearerStandardAPI
	metricsagent          *metrics.Agent
	db                    database.Database
	kv                    transactions.TxCacheDB
	vocClient             *vocclient.Client
}

func NewURLAPI(router *httprouter.HTTProuter,
	cfg *config.API, metricsAgent *metrics.Agent) (*URLAPI, error) {
	if router == nil {
		return nil, fmt.Errorf("httprouter is nil")
	}
	baseRoute := cfg.Route
	if len(baseRoute) == 0 || baseRoute[0] != '/' {
		return nil, fmt.Errorf("invalid base route (%s), it must start with /", baseRoute)
	}
	// Remove trailing slash
	if len(baseRoute) > 0 {
		baseRoute = strings.TrimSuffix(baseRoute, "/")
	}
	baseRoute += "/" + apiVersion
	urlapi := URLAPI{
		config:       cfg,
		BaseRoute:    baseRoute,
		router:       router,
		metricsagent: metricsAgent,
	}
	log.Infof("url api available with baseRoute %s", baseRoute)
	if len(cfg.GlobalEntityKey) > 0 {
		key, err := hex.DecodeString(cfg.GlobalEntityKey)
		if err != nil {
			log.Errorf("could not decode global encryption key: %v", err)
		} else {
			urlapi.globalOrganizationKey = key
		}
		log.Infof("global entity encryption key: %s", key)
	}
	if len(cfg.GlobalMetaKey) > 0 {
		key, err := hex.DecodeString(cfg.GlobalMetaKey)
		if err != nil {
			log.Fatalf("could not decode global metadata key: %v", err)
		}
		urlapi.globalMetadataKey = key
		log.Infof("global metadata encryption key: %x", urlapi.globalMetadataKey)
	}
	urlapi.registerMetrics()
	var err error
	urlapi.api, err = bearerstdapi.NewBearerStandardAPI(router, baseRoute)
	if err != nil {
		return nil, err
	}

	return &urlapi, nil
}

func (u *URLAPI) EnableVotingServiceHandlers(db database.Database,
	kv dvotedb.Database, client *vocclient.Client) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	if kv == nil {
		return fmt.Errorf("database is nil")
	}
	if client == nil {
		return fmt.Errorf("database is nil")
	}
	u.db = db
	u.kv = transactions.NewTxKv(kv, &sync.RWMutex{})
	u.vocClient = client

	// Register auth tokens from the DB
	err := u.syncAuthTokens()
	if err != nil {
		return fmt.Errorf("could not sync auth tokens with db: %v", err)
	}

	go u.monitorCachedTxs()

	if err := u.enableSuperadminHandlers(u.config.AdminToken); err != nil {
		return err
	}
	if err := u.enableEntityHandlers(); err != nil {
		return err
	}
	if err := u.enablePublicHandlers(); err != nil {
		return err
	}
	return nil
}

func (u *URLAPI) syncAuthTokens() error {
	integratorKeys, err := u.db.GetIntegratorApiKeysList()
	if err != nil {
		return err
	}
	for _, key := range integratorKeys {
		// Register integrator key to router
		log.Infof("register auth token from database %s", hex.EncodeToString(key))
		u.api.AddAuthToken(hex.EncodeToString(key), INTEGRATOR_MAX_REQUESTS)

		// Fetch integrator's organizations from the db
		orgs, err := u.db.ListOrganizations(key, &types.ListOptions{})
		if err != nil {
			return err
		}

		// Register each organization's api token to the router
		for _, org := range orgs {
			u.api.AddAuthToken(org.PublicAPIToken, int64(org.PublicAPIQuota))
		}
	}
	return nil
}

func (u *URLAPI) RegisterToken(token string, requests int64) {
	log.Infof("register auth token %s", token)
	u.api.AddAuthToken(token, requests)
}

func (u *URLAPI) RevokeToken(token string) {
	log.Infof("revoke auth token %s", token)
	u.api.DelAuthToken(token)
}

func (u *URLAPI) monitorCachedTxs() {
	callback := func(key, value []byte) bool {
		// unmarshal value to serializableTx
		var serializableTx transactions.SerializableTx
		if err := json.Unmarshal(value, &serializableTx); err != nil {
			log.Errorf("could not get query from tx cache: %v", err)
			return true
		}
		// MOCK MINED LOGIC
		// if tx not mined, check if it's old
		// Lock KvMutex so entries aren't deleted while operating on them
		u.kv.Lock()
		defer u.kv.Unlock()
		if !serializableTx.CreationTime.Add(15 * time.Second).Before(time.Now()) {
			if serializableTx.CreationTime.After(time.Now().Add(txTimeout)) {
				if err := u.kv.DeleteTx(key); err != nil {
					log.Errorf("could not delete timed-out tx: %v", err)
				}
			}
			return true
		}

		// commit that tx to the database if mined
		if err := serializableTx.Commit(u.db); err != nil {
			log.Errorf("could not commit query tx: %v", err)
		}
		// If query has been committed, delete from kv
		if err := u.kv.DeleteTx(key); err != nil {
			log.Errorf("could not delete query tx: %v", err)
			return true
		}
		return true
	}
	for {
		err := u.kv.DB.Iterate([]byte(transactions.TxPrefix), callback)
		if err != nil {
			log.Error(err)
		}
		time.Sleep(5 * time.Second)
	}
}

func sendResponse(response interface{}, ctx *httprouter.HTTPContext) error {
	data, err := json.Marshal(response)
	if err != nil {
		log.Errorf("error marshaling JSON: %v", err)
		return fmt.Errorf("error marshaling JSON: %w", err)
	}
	if err = ctx.Send(data, 200); err != nil {
		log.Error(err)
		return err
	}
	return nil
}
