package agent

import (
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dcaiman/YP_GO/internal/pkg/metric"
)

// General run of all metric collecting processes according to Poll Interval from Config.
func (agn *agent) poll() {
	pollTimer := time.NewTicker(agn.config.PollInterval)
	for {
		<-pollTimer.C
		agn.pollRuntimeGauges()
		agn.pollCustomGauges()
		agn.pollCounters()
	}
}

// Runs runtime metrics collecting processes.
func (agn *agent) pollRuntimeGauges() {
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)

	for i := range agn.RuntimeGauges {
		go func(name string, mem *runtime.MemStats) {
			val := getRuntimeMetricValue(name, mem)

			if err := agn.storage.UpdateMetric(&metric.Metric{
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
func (agn *agent) pollCustomGauges() {
	for i := range agn.CustomGauges {
		go func(name string) {
			val, err := getCustomMetricValue(name)
			if err != nil {
				log.Println(err)
				return
			}

			if err := agn.storage.UpdateMetric(&metric.Metric{
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
func (agn *agent) pollCounters() {
	for i := range agn.Counters {
		go func(name string) {
			var del int64 = 1

			if err := agn.storage.UpdateMetric(&metric.Metric{
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
