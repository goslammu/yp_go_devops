package agent

import (
	"log"
	"time"
)

// report() runs metrics sending to server according to Report Interval from Config.
// Sends metrics batch if SendBatch from Connfig is checked.
// Otherwise sends every metric individually.
func (agn *Agent) report() {
	reportTimer := time.NewTicker(agn.Cfg.ReportInterval)

	for {
		<-reportTimer.C
		if agn.Cfg.SendBatch {
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
func (agn *Agent) reportMetrics(names []string) {
	for i := range names {
		go func(i int) {
			if err := agn.sendMetric(names[i]); err != nil {
				log.Println(err)
			}
		}(i)
	}
}
