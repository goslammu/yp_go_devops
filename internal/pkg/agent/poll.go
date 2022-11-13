package agent

import (
	"log"
	"runtime"
	"time"

	"github.com/dcaiman/YP_GO/internal/pkg/metric"
)

// General run of all metric collecting processes according to Poll Interval from Config.
func (agn *Agent) poll() {
	pollTimer := time.NewTicker(agn.Cfg.PollInterval)
	for {
		<-pollTimer.C
		agn.pollRuntimeGauges()
		agn.pollCustomGauges()
		agn.pollCounters()
	}
}

// Runs runtime metrics collecting processes.
func (agn *Agent) pollRuntimeGauges() {
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)

	for i := range agn.RuntimeGauges {
		go func(name string, mem *runtime.MemStats) {
			val := agn.getRuntimeMetricValue(name, mem)

			if err := agn.Storage.UpdateMetric(&metric.Metric{
				ID:    name,
				MType: Gauge,
				Value: &val,
			}); err != nil {
				log.Println(err)
				return
			}
		}(agn.RuntimeGauges[i], memStats)
	}
}

// Runs custom metrics collecting processes.
func (agn *Agent) pollCustomGauges() {
	for i := range agn.CustomGauges {
		go func(name string) {
			val, err := agn.getCustomMetricValue(name)
			if err != nil {
				log.Println(err)
				return
			}

			if err := agn.Storage.UpdateMetric(&metric.Metric{
				ID:    name,
				MType: Gauge,
				Value: &val,
			}); err != nil {
				log.Println(err)
				return
			}
		}(agn.CustomGauges[i])
	}
}

// Runs counters processes.
func (agn *Agent) pollCounters() {
	for i := range agn.Counters {
		go func(name string) {
			var del int64 = 1

			if err := agn.Storage.UpdateMetric(&metric.Metric{
				ID:    name,
				MType: Counter,
				Delta: &del,
			}); err != nil {
				log.Println(err)
				return
			}
		}(agn.Counters[i])
	}
}
