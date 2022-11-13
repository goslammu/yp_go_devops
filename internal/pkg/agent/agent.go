package agent

import (
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dcaiman/YP_GO/internal/pkg/filestorage"
	"github.com/dcaiman/YP_GO/internal/pkg/metric"

	"github.com/caarlos0/env"
)

var (
	errShutdownBySystemCall = errors.New("shutdown by system call")
)

// Supported metric types list.
const (
	// Gauge implements any floating point value.
	Gauge = "gauge"

	// Counter implements integer which increments on every agent poll.
	Counter = "counter"
)

// Config struct contains all agent settings required for run.
type Config struct {
	// Destination of the metrics storaging server.
	SrvAddr string `env:"ADDRESS"`

	// Key for hashing report packets. Nothing will be hashed if HashKey is empty.
	HashKey string `env:"KEY"`

	// Defines http content-type of report packet.
	CType string

	// Defines if to send metrics in batch or individually.
	SendBatch bool

	// Time interval between polling actions.
	PollInterval time.Duration `env:"POLL_INTERVAL"`

	// Time interval between sendings metrics to the server.
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
}

// Agent struct implements full value client for metric collecting, and sending them to server.
type Agent struct {
	// List of metric names, which will be collected from MemStats.
	RuntimeGauges []string

	// List of metric names, which will be collected according to individual algorithm.
	CustomGauges []string

	// List of counters metric names.
	Counters []string

	// Implementation of local agent metrics storage.
	Storage metric.MStorage

	// Implementation of Config for Agent.
	Cfg Config
}

// Immediatly turns Agent on.
func (agn *Agent) Run() error {
	agn.Storage = filestorage.New("")

	go agn.poll()
	go agn.report()

	syscallCh := make(chan os.Signal, 1)

	signal.Notify(syscallCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-syscallCh

	return errShutdownBySystemCall
}

// Checks command-line flags availability and parses environment variables to fill Agent Config.
// Config hierarchy: environment variables > flags > struct.
func (agn *Agent) GetExternalConfig() error {
	flag.StringVar(&agn.Cfg.SrvAddr, "a", agn.Cfg.SrvAddr, "server address")
	flag.DurationVar(&agn.Cfg.ReportInterval, "r", agn.Cfg.ReportInterval, "report interval")
	flag.DurationVar(&agn.Cfg.PollInterval, "p", agn.Cfg.PollInterval, "poll interval")
	flag.StringVar(&agn.Cfg.HashKey, "k", agn.Cfg.HashKey, "hash key")
	flag.Parse()

	if err := env.Parse(&agn.Cfg); err != nil {
		return err
	}

	return nil
}
