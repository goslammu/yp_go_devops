package agent

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/dcaiman/YP_GO/internal/pkg/filestorage"
	"github.com/dcaiman/YP_GO/internal/pkg/metric"
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

// Agent struct implements full value client for metric collecting, and sending them to server.
type agent struct {
	// List of metric names, which will be collected from MemStats.
	RuntimeGauges []string

	// List of metric names, which will be collected according to individual algorithm.
	CustomGauges []string

	// List of counters metric names.
	Counters []string

	// Implementation of local agent metrics storage.
	storage metric.MetricStorage

	// Implementation of Config for Agent.
	config agentConfig
}

func NewAgent(config agentConfig) *agent {
	return &agent{
		config: config,
	}
}

// Immediatly turns Agent on.
func (agn *agent) Run() error {
	agn.storage = filestorage.New("")

	go agn.poll()
	go agn.report()

	syscallCh := make(chan os.Signal, 1)

	signal.Notify(syscallCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-syscallCh

	return errShutdownBySystemCall
}
