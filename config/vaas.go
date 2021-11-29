package config

import (
	"fmt"
	"time"
)

type DB struct {
	Host     string
	Port     int
	User     string
	Password string
	Dbname   string
	Sslmode  string
}

type API struct {
	// Route is the URL router where the API will be served
	Route string
	// ListenPort port where the API server will listen on
	ListenPort int
	// ListenHost host where the API server will listen on
	ListenHost string
	// Ssl tls related config options
	Ssl struct {
		Domain  string
		DirCert string
	}
}

type Error struct {
	// Critical indicates if the error encountered is critical and the app must be stopped
	Critical bool
	// Message error message
	Message string
}

// MetricsCfg initializes the metrics config
type MetricsCfg struct {
	Enabled         bool
	RefreshInterval int
}
type Vaas struct {
	// API api config options
	API *API
	// Database connection options
	DB *DB
	// LogLevel logging level
	LogLevel string
	// LogOutput logging output
	LogOutput string
	// ErrorLogFile for logging warning, error and fatal messages
	LogErrorFile string
	// Metrics config options
	Metrics *MetricsCfg
	// DataDir path where the gateway files will be stored
	DataDir string
	// SaveConfig overwrites the config file with the CLI provided flags
	SaveConfig bool
	// SigningKey is the ECDSA hexString private key for signing messages
	SigningKeys []string
	// Urls to use for gateway api
	GatewayUrls []string
	// Migration options
	Migrate *Migrate
	// Web3 connection options
	EthNetwork *EthNetwork
}

func (v *Vaas) String() string {
	return fmt.Sprintf("API: %+v,  DB: %+v, LogLevel: %s, LogOutput: %s, LogErrorFile: %s,  Metrics: %+v, DataDir: %s, SaveConfig: %v, SigningKey: %s, GatewayUrls: %v, Migrate: %+v, Eth: %v",
		*v.API, *v.DB, v.LogLevel, v.LogOutput, v.LogErrorFile, *v.Metrics, v.DataDir, v.SaveConfig, v.SigningKeys, v.GatewayUrls, *v.Migrate, *v.EthNetwork)
}

// NewVaasConfig initializes the fields in the config stuct
func NewVaasConfig() *Vaas {
	return &Vaas{
		API:        new(API),
		DB:         new(DB),
		Migrate:    new(Migrate),
		Metrics:    new(MetricsCfg),
		EthNetwork: new(EthNetwork),
	}
}

type Migrate struct {
	// Action defines the migration action to be taken (up, down, status)
	Action string
}

type EthNetwork struct {
	// NetworkName is the Ethereum Network Name
	// currently supported: "mainnet", "sokol", goerli", "xdai",
	// more info in:
	// https://github.com/vocdoni/vocdoni-node/blob/8b5a1fbc161603b96831fed7b0748190afff0bff/chain/blockchains.go
	Name string
	// Provider is the Ethereum gateway host
	Provider string
	// GasLimit is the deafult gas limit for sending an EVM transaction
	GasLimit uint64
	// FaucetAmount is the default amount of xdai/gas to send to entities
	// 1 XDAI/ETH (as xDAI is the native token for xDAI chain)
	FaucetAmount int
	// Timeout applied to the ethereum transactions
	Timeout time.Duration
}
