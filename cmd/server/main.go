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

	config := server.NewConfig(
		"127.0.0.1:8080",
		"", // "postgresql://postgres:1@127.0.0.1:5432",
		"./tmp/metricStorage.json",
		"key",
		time.Second,
		true,
		false,
	)

	if err := config.SetByExternal(); err != nil {
		panic(err)
	}

	srv := server.NewServer(config)

	if err := srv.Init(); err != nil {
		panic(err)
	}

	if err := srv.Run(); err != nil {
		panic(err)
	}

	if err := srv.Shutdown(); err != nil {
		panic(err)
	}
}
