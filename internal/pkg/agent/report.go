package agent

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// report() runs metrics sending to server according to Report Interval from Config.
// Sends metrics batch if SendBatch from Connfig is checked.
// Otherwise sends every metric individually.
func (agn *agent) report() {
	var reportFunc func()

	if agn.config.SendByBatch {
		reportFunc = func() {
			if err := agn.sendBatchAsJSON(); err != nil {
				log.Println(err)
			}
		}
	} else {
		reportFunc = func() {
			agn.reportMetrics(agn.RuntimeGauges)
			agn.reportMetrics(agn.CustomGauges)
			agn.reportMetrics(agn.Counters)
		}
	}

	reportTimer := time.NewTicker(agn.config.ReportInterval)

	for {
		select {
		case <-reportTimer.C:
			reportFunc()
		case <-agn.shutdown:
			return
		}
	}
}

// reportMetrics() sends listed metrics individually.
func (agn *agent) reportMetrics(names []string) {
	for _, v := range names {
		go func(name string) {
			if err := agn.sendMetric(name); err != nil {
				log.Println(err)
			}
		}(v)
	}
}
