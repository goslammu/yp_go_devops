package agent

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/caarlos0/env"
	log "github.com/sirupsen/logrus"
)

const (
	serverAddressFlag       = "a"
	hashKeyFlag             = "k"
	certDestinationFlag     = "crypto-key"
	pollIntervalFlag        = "p"
	reportIntervalFlag      = "r"
	configFileDestFlag      = "config"
	configFileDestFlagShort = "c"
)

var (
	errConfigFilePathNotDefined = errors.New("config file path not defined")
)

const (
	SendBatchOn  = true
	SendBatchOff = false

	ModeHTTPS = true
	ModeHTTP  = false
)

// Config struct contains all agent settings required for run.
type agentConfig struct {
	// Destination of the metrics storaging server.
	ServerAddress string `env:"ADDRESS" json:"address"`

	// Key for hashing report packets. Nothing will be hashed if HashKey is empty.
	HashKey string `env:"KEY" json:"key"`

	// Destination of TLS certification data.
	CertDestination string `env:"CRYPTO_KEY" json:"crypto_key"`

	// Defines http content-type of report packet.
	ContentType string

	// Defines if to send metrics in batch or individually.
	SendByBatch bool

	// Time interval between polling actions.
	PollInterval time.Duration `env:"POLL_INTERVAL" json:"poll_interval"`

	// Time interval between sendings metrics to the server.
	ReportInterval time.Duration `env:"REPORT_INTERVAL" json:"report_interval"`

	EnableHTTPS bool
}

// Agentconfig constructor.
func NewConfig(serverAddress, hashKey, certDestination, contentType string, sendByBatch, enableHTTPS bool, pollInterval, reportInterval time.Duration) agentConfig {
	return agentConfig{
		ServerAddress:   serverAddress,
		HashKey:         hashKey,
		CertDestination: certDestination,
		ContentType:     contentType,
		SendByBatch:     sendByBatch,
		PollInterval:    pollInterval,
		ReportInterval:  reportInterval,
		EnableHTTPS:     enableHTTPS,
	}
}

// Checks command-line flags availability and parses environment variables to fill Agent Config.
// Config hierarchy: environment variables > flags > struct.
func (cf *agentConfig) SetByExternal() error {
	var serverAddress,
		hashKey,
		certDestination,
		configFilePath string

	var pollInterval,
		reportInterval time.Duration

	flag.StringVar(&serverAddress, "a", serverAddress, "server address")
	flag.StringVar(&hashKey, "k", hashKey, "hash key")
	flag.StringVar(&certDestination, "crypto-key", certDestination, "cert data dest")

	flag.StringVar(&configFilePath, configFileDestFlag, configFilePath, "config file destination")
	flag.StringVar(&configFilePath, configFileDestFlagShort, configFilePath, "config file destination")

	flag.DurationVar(&pollInterval, "p", pollInterval, "poll interval")
	flag.DurationVar(&reportInterval, "r", reportInterval, "report interval")

	flag.Parse()

	if err := cf.setFromFile(configFilePath); err != nil {
		log.Println(err)
	}

	if isFlagSet(serverAddressFlag) {
		cf.ServerAddress = serverAddress
	}

	if isFlagSet(hashKeyFlag) {
		cf.HashKey = hashKey
	}

	if isFlagSet(certDestinationFlag) {
		cf.CertDestination = certDestination
	}

	if isFlagSet(pollIntervalFlag) {
		cf.PollInterval = pollInterval
	}

	if isFlagSet(reportIntervalFlag) {
		cf.ReportInterval = reportInterval
	}

	if err := env.Parse(cf); err != nil {
		return err
	}

	return cf.Dump("lastConfig.json")
}

func (cf *agentConfig) Dump(path string) error {
	bj, err := json.Marshal(cf)
	if err != nil {
		return err
	}

	return os.WriteFile(path, bj, os.ModePerm)
}

func (cf *agentConfig) setFromFile(path string) error {
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
