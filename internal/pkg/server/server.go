package server

import (
	"errors"
	"log"
	"net/http"

	"time"

	"github.com/dcaiman/YP_GO/internal/pkg/compresser"
	"github.com/dcaiman/YP_GO/internal/pkg/filestorage"
	"github.com/dcaiman/YP_GO/internal/pkg/metric"
	"github.com/dcaiman/YP_GO/internal/pkg/pgxstorage"
	"github.com/go-chi/chi"
	_ "github.com/jackc/pgx/v4/stdlib"
)

// server struct implements full value server for metric storaging and getting them from clients.
type server struct {
	syncUpload chan struct{}
	storage    metric.MetricStorage
	config     serverConfig
}

// Server constructor.
func NewServer(config serverConfig) *server {
	return &server{
		config: config,
	}
}

// Immediatly turns server on.
func (srv *server) Run() error {
	if srv.config.DatabaseAddress != "" {
		dbstorage, err := pgxstorage.New(srv.config.DatabaseAddress, srv.config.InitialDatabaseDrop)
		if err != nil {
			return err
		}

		defer func() {
			if er := dbstorage.Close(); er != nil {
				log.Println(er.Error())
			}
		}()

		srv.storage = dbstorage

	} else if srv.config.FileDestination != "" {
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
					<-uploadTimer.C

					if err := filestorage.UploadStorage(); err != nil {
						log.Println(err)
					}
				}
			}()
		} else {
			srv.syncUpload = make(chan struct{})

			go func(c chan struct{}) {
				for {
					<-c

					if err := filestorage.UploadStorage(); err != nil {
						log.Println(err)
					}
				}
			}(srv.syncUpload)
		}

		srv.storage = filestorage

	} else {
		return errors.New("server storage IS NOT DEFINED")
	}

	log.Println("server CONFIG: ", srv.config)

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

	return http.ListenAndServe(srv.config.ServerAddress, mainRouter)
}
