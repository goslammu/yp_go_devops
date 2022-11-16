package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dcaiman/YP_GO/internal/pkg/agent"
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
		agent.JSONCT,
		true,
		1000*time.Millisecond,
		3000*time.Millisecond,
	)

	if err := config.GetExternalConfig(); err != nil {
		panic(err)
	}

	agn := agent.NewAgent(config)

	agn.RuntimeGauges = runtimeGauges
	agn.Counters = counters
	agn.CustomGauges = customGauges

	if err := agn.Run(); err != nil {
		panic(err)
	}
}
