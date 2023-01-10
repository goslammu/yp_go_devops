package server

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/caarlos0/env"
)

const (
	initialDownloadFlag     = "r"
	serverAddressFlag       = "a"
	databaseAddressFlag     = "d"
	fileDestinationFlag     = "f"
	hashKeyFlag             = "k"
	certDestinationFlag     = "crypto-key"
	configFileDestFlag      = "config"
	configFileDestFlagShort = "c"
	storeIntervalFlag       = "i"
)

var (
	errConfigFilePathNotDefined = errors.New("config file path not defined")
)

const (
	InitialDownloadOn  = true
	InitialDownloadOff = false

	DropDatabaseOn  = true
	DropDatabaseOff = false

	ModeHTTPS = true
	ModeHTTP  = false
)

// Config struct contains all server settings required for run.
type serverConfig struct {
	// Destination of server.
	ServerAddress string `env:"ADDRESS" json:"address"`

	// Destination of database. If is not empty, database is chosen as the storage (for pgxstorage only).
	DatabaseAddress string `env:"DATABASE_DSN" json:"database_dsn"`

	// Destination of file for the file storage (for filestorage only).
	FileDestination string `env:"STORE_FILE" json:"store_file"`

	// Key for handling hashed requests.
	HashKey string `env:"KEY" json:"key"`

	// Destination of TLS certification data.
	CertDestination string `env:"CRYPTO_KEY" json:"crypto_key"`

	// Time interval between to-file storing actions (for filestorage only).
	// If not defined, storing will be made in sync way.
	StoreInterval time.Duration `env:"STORE_INTERVAL" json:"store_interval"`

	// Defines if needed to download storage on server init (for filestorage only).
	InitialDownload bool `env:"RESTORE" json:"restore"`

	// Defines if is needed to drop database on server init (for pgxstorage only).
	InitialDatabaseDrop bool `json:"-"`

	EnableHTTPS bool `json:"-"`
}

// Serverconfig constructor.
func NewConfig(serverAddress, databaseAddress, fileDestination, hashKey, certDestination string, storeInterval time.Duration, initDownload, initialDatabaseDrop, enableHTTPS bool) serverConfig {
	return serverConfig{
		ServerAddress:       serverAddress,
		DatabaseAddress:     databaseAddress,
		FileDestination:     fileDestination,
		HashKey:             hashKey,
		CertDestination:     certDestination,
		StoreInterval:       storeInterval,
		InitialDownload:     initDownload,
		InitialDatabaseDrop: initialDatabaseDrop,
		EnableHTTPS:         enableHTTPS,
	}
}

// Checks command-line flags availability and parses environment variables to fill server Config.
// Config hierarchy: environment variables > flags > struct.
func (cf *serverConfig) SetByExternal() error {
	var initialDownload bool

	var serverAddress,
		databaseAddress,
		fileDestination,
		hashKey,

		certDestination,
		configFilePath string

	var storeInterval time.Duration

	flag.BoolVar(&initialDownload, initialDownloadFlag, initialDownload, "initial download flag")

	flag.StringVar(&serverAddress, serverAddressFlag, serverAddress, "server address")
	flag.StringVar(&databaseAddress, databaseAddressFlag, databaseAddress, "database address")
	flag.StringVar(&fileDestination, fileDestinationFlag, fileDestination, "storage file destination")
	flag.StringVar(&hashKey, hashKeyFlag, hashKey, "hash key")
	flag.StringVar(&certDestination, certDestinationFlag, certDestination, "cert data destination")

	flag.StringVar(&configFilePath, configFileDestFlag, configFilePath, "config file destination")
	flag.StringVar(&configFilePath, configFileDestFlagShort, configFilePath, "config file destination")

	flag.DurationVar(&storeInterval, storeIntervalFlag, storeInterval, "store interval")

	flag.Parse()

	if err := cf.setFromFile(configFilePath); err != nil {
		log.Println(err)
	}

	if isFlagSet(initialDownloadFlag) {
		cf.InitialDownload = initialDownload
	}

	if isFlagSet(serverAddressFlag) {
		cf.ServerAddress = serverAddress
	}

	if isFlagSet(databaseAddressFlag) {
		cf.DatabaseAddress = databaseAddress
	}

	if isFlagSet(fileDestinationFlag) {
		cf.FileDestination = fileDestination
	}

	if isFlagSet(hashKeyFlag) {
		cf.HashKey = hashKey
	}

	if isFlagSet(certDestinationFlag) {
		cf.CertDestination = certDestination
	}

	if err := env.Parse(cf); err != nil {
		return err
	}

	return cf.Dump("lastConfig.json")
}

func (cf *serverConfig) Dump(path string) error {
	bj, err := json.Marshal(cf)
	if err != nil {
		return err
	}

	return os.WriteFile(path, bj, os.ModePerm)
}

func (cf *serverConfig) setFromFile(path string) error {
	if path == "" {
		return errConfigFilePathNotDefined
	}

	bj, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(bj, cf)
}

func isFlagSet(name string) (isSet bool) {
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			isSet = true
		}
	})

	return
}
