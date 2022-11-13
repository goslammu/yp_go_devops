package server

import (
	"flag"
	"time"

	"github.com/caarlos0/env"
)

// Config struct contains all server settings required for run.
type serverConfig struct {
	// Destination of server.
	ServerAddress string `env:"ADDRESS"`

	// Destination of database. If is not empty, database is chosen as the storage (for pgxstorage only).
	DatabaseAddress string `env:"DATABASE_DSN"`

	// Destination of file for the file storage (for filestorage only).
	FileDestination string `env:"STORE_FILE"`

	// Key for handling hashed requests.
	HashKey string `env:"KEY"`

	// Time interval between to-file storing actions (for filestorage only).
	// If not defined, storing will be made in sync way.
	StoreInterval time.Duration `env:"STORE_INTERVAL"`

	// Defines if needed to download storage on server init (for filestorage only).
	InitialDownload bool `env:"RESTORE"`

	// Defines if is needed to drop database on server init (for pgxstorage only).
	InitialDatabaseDrop bool
}

// Serverconfig constructor.
func NewConfig(serverAddress, databaseAddress, fileDestination, hashKey string, storeInterval time.Duration, initDownload, initialDatabaseDrop bool) serverConfig {
	return serverConfig{
		ServerAddress:   serverAddress,
		DatabaseAddress: databaseAddress,
		FileDestination: fileDestination,
		HashKey:         hashKey,
		StoreInterval:   storeInterval,
		InitialDownload: initDownload,
	}
}

// Checks command-line flags availability and parses environment variables to fill server Config.
// Config hierarchy: environment variables > flags > struct.
func (cf *serverConfig) SetByExternal() error {
	if flag.Lookup("r") == nil {
		flag.BoolVar(&cf.InitialDownload, "r", cf.InitialDownload, "initial download flag")
	}
	if flag.Lookup("f") == nil {
		flag.StringVar(&cf.FileDestination, "f", cf.FileDestination, "storage file destination")
	}
	if flag.Lookup("a") == nil {
		flag.StringVar(&cf.ServerAddress, "a", cf.ServerAddress, "server address")
	}
	if flag.Lookup("i") == nil {
		flag.DurationVar(&cf.StoreInterval, "i", cf.StoreInterval, "store interval")
	}
	if flag.Lookup("k") == nil {
		flag.StringVar(&cf.HashKey, "k", cf.HashKey, "hash key")
	}
	if flag.Lookup("d") == nil {
		flag.StringVar(&cf.DatabaseAddress, "d", cf.DatabaseAddress, "database address")
	}
	flag.Parse()

	if err := env.Parse(cf); err != nil {
		return err
	}

	return nil
}
