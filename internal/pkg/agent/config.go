package agent

import (
	"flag"
	"time"

	"github.com/caarlos0/env"
)

// Config struct contains all agent settings required for run.
type agentConfig struct {
	// Destination of the metrics storaging server.
	ServerAddr string `env:"ADDRESS"`

	// Key for hashing report packets. Nothing will be hashed if HashKey is empty.
	HashKey string `env:"KEY"`

	// Defines http content-type of report packet.
	ContentType string

	// Defines if to send metrics in batch or individually.
	SendByBatch bool

	// Time interval between polling actions.
	PollInterval time.Duration `env:"POLL_INTERVAL"`

	// Time interval between sendings metrics to the server.
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
}

// Agentconfig constructor.
func NewConfig(serverAddress, hashKey, contentType string, sendByBatch bool, pollInterval, reportInterval time.Duration) agentConfig {
	return agentConfig{
		ServerAddr:     serverAddress,
		HashKey:        hashKey,
		ContentType:    contentType,
		SendByBatch:    sendByBatch,
		PollInterval:   pollInterval,
		ReportInterval: reportInterval,
	}
}

// Checks command-line flags availability and parses environment variables to fill Agent Config.
// Config hierarchy: environment variables > flags > struct.
func (cf *agentConfig) GetExternalConfig() error {
	flag.StringVar(&cf.ServerAddr, "a", cf.ServerAddr, "server address")
	flag.DurationVar(&cf.ReportInterval, "r", cf.ReportInterval, "report interval")
	flag.DurationVar(&cf.PollInterval, "p", cf.PollInterval, "poll interval")
	flag.StringVar(&cf.HashKey, "k", cf.HashKey, "hash key")
	flag.Parse()

	return env.Parse(cf)
}
