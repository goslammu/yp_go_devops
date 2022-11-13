package main

import (
	"log"
	"net/http"

	_ "net/http/pprof"
	"time"

	"github.com/dcaiman/YP_GO/internal/pkg/server"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	go func() {
		if err := http.ListenAndServe(":6060", nil); err != nil {
			panic(err)
		}
	}()

	log.Println("Build version: ", buildVersion)
	log.Println("Build date: ", buildDate)
	log.Println("Build commit: ", buildCommit)

	srv := server.Server{
		Cfg: server.Config{
			Address: "127.0.0.1:8080",
			//DatabaseAddress: "postgresql://postgres:1@127.0.0.1:5432",
			StoreInterval:   0 * time.Second,
			FileDestination: "./tmp/metricStorage.json",
			HashKey:         "key",
			InitDownload:    true,

			DropDB: false,
		},
	}

	if err := srv.GetExternalConfig(); err != nil {
		panic(err)
	}

	if err := srv.Run(); err != nil {
		panic(err)
	}
}
