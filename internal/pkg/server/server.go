package server

import (
	"flag"
	"log"
	"net/http"

	"time"

	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/dcaiman/YP_GO/internal/pkg/compresser"
	"github.com/dcaiman/YP_GO/internal/pkg/filestorage"
	"github.com/dcaiman/YP_GO/internal/pkg/metric"
	"github.com/dcaiman/YP_GO/internal/pkg/pgxstorage"

	"github.com/caarlos0/env"
	"github.com/go-chi/chi/v5"
)

// Config struct contains all server settings required for run.
type Config struct {
	// Destination of server.
	Address string `env:"ADDRESS"`

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
	InitDownload bool `env:"RESTORE"`

	// Defines if is needed to drop database on server init (for pgxstorage only).
	DropDB bool
}

// Server struct implements full value server for metric storaging and getting them from clients.
type Server struct {
	SyncUpload chan struct{}
	Storage    metric.MStorage
	Cfg        Config
}

// Immediatly turns Server on.
func (srv *Server) Run() error {
	if srv.Cfg.DatabaseAddress != "" {
		dbStorage, err := pgxstorage.New(srv.Cfg.DatabaseAddress, srv.Cfg.DropDB)
		if err != nil {
			return err
		}
		defer func() {
			if er := dbStorage.Close(); er != nil {
				log.Println(er.Error())
			}
		}()
		srv.Storage = dbStorage

	} else if srv.Cfg.FileDestination != "" {
		fileStorage := filestorage.New(srv.Cfg.FileDestination)

		if srv.Cfg.InitDownload {
			if err := fileStorage.DownloadStorage(); err != nil {
				log.Println(err)
			}
		}
		if srv.Cfg.StoreInterval != 0 {
			go func() {
				uploadTimer := time.NewTicker(srv.Cfg.StoreInterval)
				for {
					<-uploadTimer.C
					if err := fileStorage.UploadStorage(); err != nil {
						log.Println(err)
					}
				}
			}()
		} else {
			srv.SyncUpload = make(chan struct{})
			go func(c chan struct{}) {
				for {
					<-c
					if err := fileStorage.UploadStorage(); err != nil {
						log.Println(err)
					}
				}
			}(srv.SyncUpload)
		}
		srv.Storage = fileStorage
	} else {
		panic("SERVER STORAGE IS NOT DEFINED")
	}

	log.Println("SERVER CONFIG: ", srv.Cfg)

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

	return http.ListenAndServe(srv.Cfg.Address, mainRouter)
}

// Checks command-line flags availability and parses environment variables to fill Server Config.
// Config hierarchy: environment variables > flags > struct.
func (srv *Server) GetExternalConfig() error {
	flag.BoolVar(&srv.Cfg.InitDownload, "r", srv.Cfg.InitDownload, "initial download flag")
	flag.StringVar(&srv.Cfg.FileDestination, "f", srv.Cfg.FileDestination, "storage file destination")
	flag.StringVar(&srv.Cfg.Address, "a", srv.Cfg.Address, "server address")
	flag.DurationVar(&srv.Cfg.StoreInterval, "i", srv.Cfg.StoreInterval, "store interval")
	flag.StringVar(&srv.Cfg.HashKey, "k", srv.Cfg.HashKey, "hash key")
	flag.StringVar(&srv.Cfg.DatabaseAddress, "d", srv.Cfg.DatabaseAddress, "database address")
	flag.Parse()

	if err := env.Parse(&srv.Cfg); err != nil {
		return err
	}

	return nil
}
