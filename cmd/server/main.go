package main

import (
	"net/http"

	"github.com/goslammu/yp_go_devops/internal/pkg/server"
	log "github.com/sirupsen/logrus"

	_ "net/http/pprof"
	"time"
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

	config := server.NewConfig(
		"127.0.0.1:8080",
		"", //"postgresql://postgres:1@127.0.0.1:5432",
		"./tmp/metricStorage.json",
		"key",
		"./certs/",
		time.Second,
		server.InitialDownloadOn,
		server.DropDatabaseOff,
		server.ModeHTTP,
	)

	if err := config.SetByExternal(); err != nil {
		panic(err)
	}

	srv := server.NewServer(config)

	if err := srv.Init(); err != nil {
		log.Println(err)
	}

	if err := srv.Run(); err != nil {
		log.Println(err)
	}
}
