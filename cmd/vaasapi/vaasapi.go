package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.vocdoni.io/api/config"
	"go.vocdoni.io/api/database"
	"go.vocdoni.io/api/database/pgsql"
	"go.vocdoni.io/api/urlapi"
	"go.vocdoni.io/api/vocclient"
	"go.vocdoni.io/dvote/crypto/ethereum"
	"go.vocdoni.io/dvote/httprouter"
	log "go.vocdoni.io/dvote/log"
	"go.vocdoni.io/dvote/metrics"
)

func newConfig() (*config.Vaas, config.Error) {
	var err error
	var cfgError config.Error
	cfg := config.NewVaasConfig()
	home, err := os.UserHomeDir()
	if err != nil {
		cfgError = config.Error{
			Critical: true,
			Message:  fmt.Sprintf("cannot get user home directory with error: %s", err),
		}
		return nil, cfgError
	}
	// flags
	flag.StringVar(&cfg.DataDir, "dataDir", home+"/.vaasapi", "directory where data is stored")
	cfg.LogLevel = *flag.String("logLevel", "info", "Log level (debug, info, warn, error, fatal)")
	cfg.LogOutput = *flag.String("logOutput", "stdout", "Log output (stdout, stderr or filepath)")
	cfg.LogErrorFile = *flag.String("logErrorFile", "", "Log errors and warnings to a file")
	cfg.SaveConfig = *flag.Bool("saveConfig", false,
		"overwrites an existing config file with the CLI provided flags")
	cfg.SigningKey = *flag.String("signingKey", "",
		"signing private Keys (if not specified, a new "+
			"one will be created), the first one is the oracle public key")
	cfg.API.AdminToken = *flag.String("adminToken", "", "hexString token for admin api calls")
	cfg.API.ExplorerVoteUrl = *flag.String("explorerVoteUrl",
		"https://vaas.explorer.vote/envelope/", "explorer url for vote envelope pages")
	cfg.API.GlobalEntityKey = *flag.String("globalEntityKey", "",
		"encryption key for organization private keys in the db. Leave empty for no encryption")
	cfg.API.GatewayUrl = *flag.String("gatewayUrl",
		"https://api-dev.vocdoni.net", "url to use as gateway api endpoint")
	cfg.API.MaxCensusSize = *flag.Uint64("maxCensusSize", 2<<32, "maximum size of a voter census")
	cfg.API.Route = *flag.String("apiRoute", "/", "dvote API route")
	cfg.API.ListenHost = *flag.String("listenHost", "0.0.0.0", "API endpoint listen address")
	cfg.API.ListenPort = *flag.Int("listenPort", 8000, "API endpoint http port")
	cfg.API.Ssl.Domain = *flag.String("sslDomain", "",
		"enable TLS secure domain with LetsEncrypt auto-generated certificate")
	cfg.DB.Host = *flag.String("dbHost", "127.0.0.1", "DB server address")
	cfg.DB.Port = *flag.Int("dbPort", 5432, "DB server port")
	cfg.DB.User = *flag.String("dbUser", "user", "DB Username")
	cfg.DB.Password = *flag.String("dbPassword", "password", "DB password")
	cfg.DB.Dbname = *flag.String("dbName", "database", "DB database name")
	cfg.DB.Sslmode = *flag.String("dbSslmode", "prefer", "DB postgres sslmode")
	cfg.DefaultPlan.MaxCensusSize = *flag.Int("defaultPlanCensusSize",
		500, "Default census size (500)")
	cfg.DefaultPlan.MaxProccessCount = *flag.Int("defaultPlanProccessCount",
		10, "Default process count (10)")
	cfg.Migrate.Action = *flag.String("migrateAction", "", "Migration action (up,down,status)")
	// metrics
	cfg.Metrics.Enabled = *flag.Bool("metricsEnabled", true, "enable prometheus metrics")
	cfg.Metrics.RefreshInterval =
		*flag.Int("metricsRefreshInterval", 10, "metrics refresh interval in seconds")

	// parse flags
	flag.Parse()

	// setting up viper
	viper := viper.New()
	viper.AddConfigPath(cfg.DataDir)
	viper.SetConfigName("vaasapi")
	viper.SetConfigType("yml")
	viper.SetEnvPrefix("VAASAPI")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// binding flags to viper

	// global
	viper.BindPFlag("dataDir", flag.Lookup("dataDir"))
	viper.BindPFlag("logLevel", flag.Lookup("logLevel"))
	viper.BindPFlag("logErrorFile", flag.Lookup("logErrorFile"))
	viper.BindPFlag("logOutput", flag.Lookup("logOutput"))
	viper.BindPFlag("signingKey", flag.Lookup("signingKey"))
	viper.BindPFlag("api.adminToken", flag.Lookup("adminToken"))
	viper.BindPFlag("api.maxCensusSize", flag.Lookup("maxCensusSize"))
	viper.BindPFlag("api.explorerVoteUrl", flag.Lookup("explorerVoteUrl"))
	viper.BindPFlag("api.globalEntityKey", flag.Lookup("globalEntityKey"))
	viper.BindPFlag("api.gatewayUrl", flag.Lookup("gatewayUrl"))
	viper.BindPFlag("api.route", flag.Lookup("apiRoute"))
	viper.BindPFlag("api.listenHost", flag.Lookup("listenHost"))
	viper.BindPFlag("api.listenPort", flag.Lookup("listenPort"))
	viper.Set("api.ssl.dirCert", cfg.DataDir+"/tls")
	viper.BindPFlag("api.ssl.domain", flag.Lookup("sslDomain"))
	viper.BindPFlag("db.host", flag.Lookup("dbHost"))
	viper.BindPFlag("db.port", flag.Lookup("dbPort"))
	viper.BindPFlag("db.user", flag.Lookup("dbUser"))
	viper.BindPFlag("db.password", flag.Lookup("dbPassword"))
	viper.BindPFlag("db.dbName", flag.Lookup("dbName"))
	viper.BindPFlag("db.sslMode", flag.Lookup("dbSslmode"))
	viper.BindPFlag("defaultPlan.censusSize", flag.Lookup("defaultPlanCensusSize"))
	viper.BindPFlag("defaultPlan.ProcessCount", flag.Lookup("defaultPlanProcessCount"))
	viper.BindPFlag("migrate.action", flag.Lookup("migrateAction"))
	// metrics
	viper.BindPFlag("metrics.enabled", flag.Lookup("metricsEnabled"))
	viper.BindPFlag("metrics.refreshInterval", flag.Lookup("metricsRefreshInterval"))

	// check if config file exists
	_, err = os.Stat(cfg.DataDir + "/vaasapi.yml")
	if os.IsNotExist(err) {
		cfgError = config.Error{
			Message: fmt.Sprintf("creating new config file in %s", cfg.DataDir),
		}
		// creting config folder if not exists
		err = os.MkdirAll(cfg.DataDir, os.ModePerm)
		if err != nil {
			cfgError = config.Error{
				Message: fmt.Sprintf("cannot create data directory: %s", err),
			}
		}
		// create config file if not exists
		if err := viper.SafeWriteConfig(); err != nil {
			cfgError = config.Error{
				Message: fmt.Sprintf("cannot write config file into config dir: %s", err),
			}
		}
	} else {
		// read config file
		err = viper.ReadInConfig()
		if err != nil {
			cfgError = config.Error{
				Message: fmt.Sprintf("cannot read loaded config file in %s: %s", cfg.DataDir, err),
			}
		}
	}
	err = viper.Unmarshal(&cfg)
	if err != nil {
		cfgError = config.Error{
			Message: fmt.Sprintf("cannot unmarshal loaded config file: %s", err),
		}
	}

	// Generate and save signing key if nos specified
	if len(cfg.SigningKey) == 0 {
		fmt.Println("no signing keys, generating one...")
		signer := ethereum.NewSignKeys()
		signer.Generate()
		if err != nil {
			cfgError = config.Error{
				Message: fmt.Sprintf("cannot generate signing key: %s", err),
			}
			return cfg, cfgError
		}
		_, priv := signer.HexString()
		viper.Set("signingkey", priv)
		cfg.SigningKey = priv
		cfg.SaveConfig = true
	}

	if cfg.SaveConfig {
		viper.Set("saveConfig", false)
		if err := viper.WriteConfig(); err != nil {
			cfgError = config.Error{
				Message: fmt.Sprintf("cannot overwrite config file into config dir: %s", err),
			}
		}
	}
	return cfg, cfgError
}

func main() {
	var err error
	// setup config
	// creating config and init logger
	cfg, cfgerr := newConfig()
	if cfgerr.Critical {
		panic(cfgerr.Message)
	}
	if cfg == nil {
		panic("cannot read configuration")
	}
	log.Init(cfg.LogLevel, cfg.LogOutput)
	if path := cfg.LogErrorFile; path != "" {
		if err := log.SetFileErrorLog(path); err != nil {
			log.Fatal(err)
		}
	}
	log.Debugf("initializing config: %s", cfg.String())

	// Signer
	signer := ethereum.NewSignKeys()
	if err := signer.AddHexKey(cfg.SigningKey); err != nil {
		log.Fatal(err)
	}
	pub, _ := signer.HexString()
	log.Infof("my public key: %s", pub)
	log.Infof("my address: %s", signer.AddressString())

	client, err := vocclient.New(cfg.API.GatewayUrl, signer)
	if err != nil {
		log.Fatal(err)
	}
	blockHeight, err := client.GetCurrentBlock()
	if err != nil {
		log.Error(err)
	}
	log.Infof("Connected to %s at block height %d", client.ActiveEndpoint(), blockHeight)

	// Database Interface
	var db database.Database

	// Postgres with sqlx
	db, err = pgsql.New(cfg.DB)
	if err != nil {
		log.Fatal(err)
	}

	// Standalone Migrations
	if cfg.Migrate.Action != "" {
		if err := pgsql.Migrator(cfg.Migrate.Action, db); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Check that all migrations are applied before proceeding
	// and if not apply them
	if err := pgsql.Migrator("upSync", db); err != nil {
		log.Fatal(err)
	}

	// Router
	var httpRouter httprouter.HTTProuter
	httpRouter.TLSdomain = cfg.API.Ssl.Domain
	httpRouter.TLSdirCert = cfg.API.Ssl.DirCert
	if err = httpRouter.Init(cfg.API.ListenHost, cfg.API.ListenPort); err != nil {
		log.Fatal(err)
	}

	var metricsAgent *metrics.Agent
	// Enable metrics via proxy
	if cfg.Metrics.Enabled {
		metricsAgent = metrics.NewAgent("/metrics",
			time.Duration(cfg.Metrics.RefreshInterval)*time.Second, &httpRouter)
	}

	// Rest api
	urlApi, err := urlapi.NewURLAPI(&httpRouter, cfg.API, metricsAgent)
	if err != nil {
		log.Fatal(err)
	}

	// Vaas api
	log.Infof("enabling VaaS API methods")
	if err := urlApi.EnableVotingServiceHandlers(db, client); err != nil {
		log.Fatal(err)
	}

	// Start token notifier
	integratorTokenNotifier, err := pgsql.NewNotifier(cfg.DB, "integrator_tokens_update")
	if err != nil {
		log.Fatal(err)
	}
	go integratorTokenNotifier.FetchNewTokens(urlApi)

	log.Info("startup complete")
	// close if interrupt received
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Warnf("received SIGTERM, exiting at %s", time.Now().Format(time.RFC850))
	os.Exit(0)
}
