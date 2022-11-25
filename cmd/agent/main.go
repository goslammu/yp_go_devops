package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/goslammu/yp_go_devops/internal/pkg/agent"
	log "github.com/sirupsen/logrus"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	go func() {
		if err := http.ListenAndServe(":7070", nil); err != nil {
			panic(err)
		}
	}()

	log.Println("Build version: ", buildVersion)
	log.Println("Build date: ", buildDate)
	log.Println("Build commit: ", buildCommit)

	runtimeGauges := []string{
		"Alloc",
		"BuckHashSys",
		"Frees",
		"GCCPUFraction",
		"GCSys",
		"HeapAlloc",
		"HeapIdle",
		"HeapInuse",
		"HeapObjects",
		"HeapReleased",
		"HeapSys",
		"LastGC",
		"Lookups",
		"MCacheInuse",
		"MCacheSys",
		"MSpanInuse",
		"MSpanSys",
		"Mallocs",
		"NextGC",
		"NumForcedGC",
		"NumGC",
		"OtherSys",
		"PauseTotalNs",
		"StackInuse",
		"StackSys",
		"Sys",
		"TotalAlloc",
	}
	customGauges := []string{
		agent.RandomValue,
		agent.TotalMemory,
		agent.FreeMemory,
	}
	counters := []string{
		"PollCount",
	}
	for i := 1; i <= runtime.NumCPU(); i++ {
		customGauges = append(customGauges, agent.CPUutilization+fmt.Sprint(i))
	}

	config := agent.NewConfig(
		"127.0.0.1:8080",
		"key",
		"./../server/certs/cert.pem",
		agent.JSONCT,
		agent.SendBatchOn,
		agent.ModeHTTP,
		time.Second,
		3*time.Second,
	)

	if err := config.SetByExternal(); err != nil {
		log.Println(err)
	}

	agn := agent.NewAgent(config)

	agn.RuntimeGauges = runtimeGauges
	agn.Counters = counters
	agn.CustomGauges = customGauges

	if err := agn.Init(); err != nil {
		log.Println(err)
	}

	if err := agn.Run(); err != nil {
		log.Println(err)
	}
}
