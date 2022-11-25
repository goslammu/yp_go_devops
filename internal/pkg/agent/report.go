package agent

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// report() runs metrics sending to server according to Report Interval from Config.
// Sends metrics batch if SendBatch from Connfig is checked.
// Otherwise sends every metric individually.
func (agn *agent) report() {
	reportTimer := time.NewTicker(agn.config.ReportInterval)

	for {
		select {
		case <-reportTimer.C:

			switch agn.config.SendByBatch {
			case true:
				if err := agn.sendBatch(); err != nil {
					log.Println(err)
				}
			default:
				agn.reportMetrics(agn.RuntimeGauges)
				agn.reportMetrics(agn.CustomGauges)
				agn.reportMetrics(agn.Counters)
			}
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
