package config

import (
	"fmt"
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
}

func (v *Vaas) String() string {
	return fmt.Sprintf("API: %+v,  DB: %+v, LogLevel: %s, LogOutput: %s, LogErrorFile: %s,  Metrics: %+v, DataDir: %s, SaveConfig: %v, SigningKey: %s, GatewayUrls: %v, Migrate: %+v",
		*v.API, *v.DB, v.LogLevel, v.LogOutput, v.LogErrorFile, *v.Metrics, v.DataDir, v.SaveConfig, v.SigningKeys, v.GatewayUrls, *v.Migrate)
}

// NewVaasConfig initializes the fields in the config stuct
func NewVaasConfig() *Vaas {
	return &Vaas{
		API:     new(API),
		DB:      new(DB),
		Migrate: new(Migrate),
		Metrics: new(MetricsCfg),
	}
}

type Migrate struct {
	// Action defines the migration action to be taken (up, down, status)
	Action string
}
