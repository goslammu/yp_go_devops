package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"time"

	"github.com/go-chi/chi"
	"github.com/goslammu/yp_go_devops/internal/pkg/compresser"
	"github.com/goslammu/yp_go_devops/internal/pkg/filestorage"
	"github.com/goslammu/yp_go_devops/internal/pkg/metric"
	"github.com/goslammu/yp_go_devops/internal/pkg/pgxstorage"
	_ "github.com/jackc/pgx/v4/stdlib"
)

var (
	errShutdownBySystemCall = errors.New("shutdown by system call")
	errStorageNotDefined    = errors.New("server storage is not defined")
	errNotInitialized       = errors.New("server is not initialized")
	errAlreadyInitialized   = errors.New("server is already initialized")
	errNotTurnedOn          = errors.New("server is not turned on")
	errTurnedOn             = errors.New("server is turned on")
)

const (
	certFileName = "cert.pem"
	keyFileName  = "key.pem"
)

// server struct implements full value server for metric storaging and getting them from clients.
type server struct {
	storage               metric.MetricStorage
	uploadSig, shutdown   chan struct{}
	server                *http.Server
	config                serverConfig
	initialized, turnedOn bool
}

// Server constructor.
func NewServer(config serverConfig) *server {
	return &server{
		config: config,
	}
}

// Init initializes server components.
func (srv *server) Init() error {
	if srv.initialized {
		return errAlreadyInitialized
	}

	if srv.turnedOn {
		return errTurnedOn
	}

	if err := srv.initStorage(); err != nil {
		return err
	}

	if err := srv.initRouter(); err != nil {
		return err
	}
	/*
		if err := srv.UpdateCert(); err != nil {
			return err
		}
	*/
	srv.initialized = true

	return nil
}

// Run immediatly turns server on.
func (srv *server) Run() error {
	if !srv.initialized {
		return errNotInitialized

	}

	if srv.turnedOn {
		return errTurnedOn
	}

	srv.shutdown = make(chan struct{})

	if srv.config.EnableHTTPS {
		go srv.runHTTPS()
	} else {
		go srv.runHTTP()
	}

	srv.turnedOn = true

	return srv.shutdownHandler()
}

func (srv *server) Shutdown() error {
	if !srv.initialized {
		return errNotInitialized
	}

	if !srv.turnedOn {
		return errNotTurnedOn
	}

	if srv.uploadSig != nil {
		close(srv.uploadSig)
	}

	if err := srv.storage.Close(); err != nil {
		return err
	}

	if err := srv.server.Shutdown(context.Background()); err != nil {
		return err
	}

	srv.turnedOn = false

	return nil
}

// initStorage initializes storage according to server configuration.
func (srv *server) initStorage() error {
	switch {
	case srv.config.DatabaseAddress != "":
		return srv.initDatabaseStorage()
	case srv.config.FileDestination != "":
		return srv.initFileStorage()
	default:
		return errStorageNotDefined
	}
}

func (srv *server) initDatabaseStorage() error {
	dbstorage, err := pgxstorage.New(srv.config.DatabaseAddress, srv.config.InitialDatabaseDrop)
	if err != nil {
		return err
	}

	srv.storage = dbstorage

	srv.config.StoreInterval = -1

	return nil
}

func (srv *server) initFileStorage() error {
	filestorage := filestorage.New(srv.config.FileDestination)

	if srv.config.InitialDownload {
		if err := filestorage.DownloadStorage(); err != nil {
			log.Println(err)
		}
	}

	if srv.config.StoreInterval != 0 {
		go func() {
			uploadTimer := time.NewTicker(srv.config.StoreInterval)

			for {
				select {
				case <-uploadTimer.C:
					if err := filestorage.UploadStorage(); err != nil {
						log.Println(err)
					}
				case <-srv.shutdown:
					return
				}
			}
		}()
	} else {
		srv.uploadSig = make(chan struct{})

		go func() {
			for {
				select {
				case <-srv.uploadSig:
					if err := filestorage.UploadStorage(); err != nil {
						log.Println(err)
					}
				case <-srv.shutdown:
					return
				}
			}
		}()

	}

	srv.storage = filestorage

	return nil
}

// initRouter initializes server main http-router.
func (srv *server) initRouter() error {
	mainRouter := chi.NewRouter()

	mainRouter.Use(compresser.Compresser)

	mainRouter.Route("/", func(r chi.Router) {
		r.Get("/", srv.handlerGetAll)
	})
	mainRouter.Route("/value", func(r chi.Router) {
		r.Post("/", srv.handlerGetMetricJSON)
		r.Get("/{type}/{name}", srv.handlerGetMetric)
	})
	mainRouter.Route("/update", func(r chi.Router) {
		r.Post("/", srv.handlerUpdateJSON)
		r.Post("/{type}/{name}/{val}", srv.handlerUpdateDirect)
	})
	mainRouter.Route("/updates", func(r chi.Router) {
		r.Post("/", srv.handlerUpdateBatch)
	})
	mainRouter.Route("/ping", func(r chi.Router) {
		r.Get("/", srv.handlerCheckConnection)
	})

	srv.server = &http.Server{
		Addr:    srv.config.ServerAddress,
		Handler: mainRouter,
	}

	return nil
}

func (srv *server) UpdateCert() error {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),

		Subject: pkix.Name{
			Organization: []string{"orgName"},
			Country:      []string{"ru"},
		},

		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(10, 0, 0),

		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},

		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	certBytes, err := x509.CreateCertificate(
		rand.Reader,
		cert,
		cert,
		&privateKey.PublicKey,
		privateKey)
	if err != nil {
		return err
	}

	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err := os.WriteFile(srv.config.CertDestination+certFileName, certPem, os.ModePerm); err != nil {
		log.Println(srv.config.CertDestination + certFileName)
		return err
	}

	privateKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err := os.WriteFile(srv.config.CertDestination+"/"+keyFileName, privateKeyPem, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func (srv *server) shutdownHandler() error {
	sysCall := make(chan os.Signal, 1)
	signal.Notify(sysCall, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-sysCall:
		close(sysCall)

		if err := srv.Shutdown(); err != nil {
			return err
		}

		return errShutdownBySystemCall
	case <-srv.shutdown:
		close(sysCall)

		return nil
	}
}

func (srv *server) runHTTPS() {
	if err := srv.server.ListenAndServeTLS(
		srv.config.CertDestination+certFileName,
		srv.config.CertDestination+keyFileName); err != nil {
		log.Println(err)

		if srv.turnedOn {
			if err := srv.Shutdown(); err != nil {
				log.Println(err)
			}
		}
	}
}

func (srv *server) runHTTP() {
	if err := srv.server.ListenAndServe(); err != nil {
		log.Println(err)

		if srv.turnedOn {
			if err := srv.Shutdown(); err != nil {
				log.Println(err)
			}
		}
	}
}
