package agent

import (
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/goslammu/yp_go_devops/internal/pkg/metric"
)

// General run of all metric collecting processes according to Poll Interval from Config.
func (agn *agent) poll() {
	pollTimer := time.NewTicker(agn.config.PollInterval)

	for {
		select {
		case <-pollTimer.C:
			agn.pollRuntimeGauges()
			agn.pollCustomGauges()
			agn.pollCounters()
		case <-agn.shutdown:
			return
		}
	}
}

// Runs runtime metrics collecting processes.
func (agn *agent) pollRuntimeGauges() {
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)

	for _, v := range agn.RuntimeGauges {
		go func(name string) {
			val := getRuntimeMetricValue(name, memStats)

			if err := agn.storage.UpdateMetric(
				&metric.Metric{
					ID:    name,
					MType: Gauge,
					Value: &val,
				}); err != nil {
				log.Println(err)

				return
			}
		}(v)
	}
}

// Runs custom metrics collecting processes.
func (agn *agent) pollCustomGauges() {
	for _, v := range agn.CustomGauges {
		go func(name string) {
			val, err := getCustomMetricValue(name)
			if err != nil {
				log.Println(err)

				return
			}

			if err := agn.storage.UpdateMetric(
				&metric.Metric{
					ID:    name,
					MType: Gauge,
					Value: &val,
				}); err != nil {
				log.Println(err)

				return
			}
		}(v)
	}
}

// Runs counters processes.
func (agn *agent) pollCounters() {
	for _, v := range agn.Counters {
		go func(name string) {
			var del int64 = 1

			if err := agn.storage.UpdateMetric(
				&metric.Metric{
					ID:    name,
					MType: Counter,
					Delta: &del,
				}); err != nil {
				log.Println(err)

				return
			}
		}(v)
	}
}
