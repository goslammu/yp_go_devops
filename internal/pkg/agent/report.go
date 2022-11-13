package agent

import (
	"log"
	"time"
)

// report() runs metrics sending to server according to Report Interval from Config.
// Sends metrics batch if SendBatch from Connfig is checked.
// Otherwise sends every metric individually.
func (agn *agent) report() {
	reportTimer := time.NewTicker(agn.config.ReportInterval)

	for {
		<-reportTimer.C
		if agn.config.SendByBatch {
			if err := agn.sendBatch(); err != nil {
				log.Println(err)
			}
		} else {
			agn.reportMetrics(agn.RuntimeGauges)
			agn.reportMetrics(agn.CustomGauges)
			agn.reportMetrics(agn.Counters)
		}
	}
}

// reportMetrics() sends listed metrics individually.
func (agn *agent) reportMetrics(names []string) {
	for i := range names {
		go func(i int) {
			if err := agn.sendMetric(names[i]); err != nil {
				log.Println(err)
			}
		}(i)
	}
}
