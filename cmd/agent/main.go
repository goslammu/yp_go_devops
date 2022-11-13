package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/dcaiman/YP_GO/internal/pkg/agent"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
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

	agn := agent.Agent{
		Cfg: agent.Config{
			CType:          agent.JSONCT,
			PollInterval:   1000 * time.Millisecond,
			ReportInterval: 3000 * time.Millisecond,
			SrvAddr:        "127.0.0.1:8080",
			HashKey:        "key",
			SendBatch:      true,
		},
		RuntimeGauges: runtimeGauges,
		CustomGauges:  customGauges,
		Counters:      counters,
	}

	if err := agn.GetExternalConfig(); err != nil {
		panic(err)
	}

	go func() {
		if err := http.ListenAndServe(":7070", nil); err != nil {
			panic(err)
		}
	}()

	if err := agn.Run(); err != nil {
		panic(err)
	}
}
