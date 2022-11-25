package agent

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/goslammu/yp_go_devops/internal/pkg/filestorage"
	"github.com/goslammu/yp_go_devops/internal/pkg/metric"
)

var (
	errShutdownBySystemCall = errors.New("shutdown by system call")
	errNotInitialized       = errors.New("agent is not initialized")
	errAlreadyInitialized   = errors.New("agent is already initialized")
	errNotTurnedOn          = errors.New("agent is not turned on")
	errTurnedOn             = errors.New("agent is turned on")
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

	shutdown chan struct{}

	initialized, turnedOn bool

	client http.Client
}

// Agent constructor.
func NewAgent(config agentConfig) *agent {
	return &agent{
		config: config,
	}
}

func (agn *agent) Init() error {
	if agn.initialized {
		return errAlreadyInitialized
	}

	if agn.turnedOn {
		return errTurnedOn
	}

	agn.client = *http.DefaultClient

	if agn.config.EnableHTTPS {
		if err := agn.initHTTPSclient(); err != nil {
			return err
		}
	}

	agn.storage = filestorage.New("")

	agn.initialized = true

	return nil
}

// Immediatly turns Agent on.
func (agn *agent) Run() error {
	if !agn.initialized {
		return errNotInitialized
	}

	if agn.turnedOn {
		return errTurnedOn
	}

	agn.shutdown = make(chan struct{})

	go agn.poll()
	go agn.report()

	agn.turnedOn = true

	return agn.shutdownHandler()
}

func (agn *agent) Shutdown() error {
	if !agn.initialized {
		return errNotInitialized
	}

	if !agn.turnedOn {
		return errNotTurnedOn
	}

	close(agn.shutdown)

	agn.turnedOn = false

	return nil
}

func (agn *agent) shutdownHandler() error {
	sysCall := make(chan os.Signal, 1)
	signal.Notify(sysCall, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-sysCall:
		close(sysCall)

		if err := agn.Shutdown(); err != nil {
			return err
		}

		return errShutdownBySystemCall
	case <-agn.shutdown:
		close(sysCall)

		return nil
	}
}

func (agn *agent) initHTTPSclient() error {
	cert, err := os.ReadFile(agn.config.CertDestination)
	if err != nil {
		return err
	}

	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(cert[:])

	agn.client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: rootCAs}}

	return nil
}
