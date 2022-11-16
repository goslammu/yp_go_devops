package server

import (
	"context"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"

	"time"

	"github.com/dcaiman/YP_GO/internal/pkg/compresser"
	"github.com/dcaiman/YP_GO/internal/pkg/filestorage"
	"github.com/dcaiman/YP_GO/internal/pkg/metric"
	"github.com/dcaiman/YP_GO/internal/pkg/pgxstorage"
	"github.com/go-chi/chi"
	_ "github.com/jackc/pgx/v4/stdlib"
)

var (
	errStorageNotDefined = errors.New("server storage is not defined")
	errNotInitialized    = errors.New("server is not initialized")
)

// server struct implements full value server for metric storaging and getting them from clients.
type server struct {
	storage     metric.MetricStorage
	uploadSig   chan struct{}
	s           *http.Server
	config      serverConfig
	initialized bool
}

// Server constructor.
func NewServer(config serverConfig) *server {
	return &server{
		config: config,
	}
}

// Init initializes server components.
func (srv *server) Init() error {
	if err := srv.initStorage(); err != nil {
		return err
	}

	if err := srv.initRouter(); err != nil {
		return err
	}

	srv.initialized = true

	return nil
}

// Run immediatly turns server on.
func (srv *server) Run() error {
	if srv.initialized {
		return srv.s.ListenAndServe()
	}

	return errNotInitialized
}

func (srv *server) Shutdown() error {
	if srv.uploadSig != nil {
		close(srv.uploadSig)
	}

	if err := srv.storage.Close(); err != nil {
		return err
	}

	return srv.s.Shutdown(context.Background())
}

// initStorage initializes storage according to server configuration.
func (srv *server) initStorage() error {
	if srv.config.DatabaseAddress != "" {
		dbstorage, err := pgxstorage.New(srv.config.DatabaseAddress, srv.config.InitialDatabaseDrop)
		if err != nil {
			return err
		}

		srv.storage = dbstorage

		srv.config.StoreInterval = -1

		return nil
	}

	if srv.config.FileDestination != "" {
		srv.uploadSig = make(chan struct{})

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
					case <-srv.uploadSig:
						return
					}
				}
			}()
		} else {
			go func(syncUploadSig chan struct{}) {
				for {
					_, ok := <-syncUploadSig
					if !ok {
						return
					}

					if err := filestorage.UploadStorage(); err != nil {
						log.Println(err)
					}
				}
			}(srv.uploadSig)
		}

		srv.storage = filestorage

		return nil
	}

	return errStorageNotDefined
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

	srv.s = &http.Server{
		Addr:    srv.config.ServerAddress,
		Handler: mainRouter,
	}

	return nil
}
