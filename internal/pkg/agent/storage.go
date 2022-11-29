package agent

import "encoding/json"

// Gives a batch of all storaged metrics in json format.
func (agn *agent) getStorageBatch() ([]byte, error) {
	allMetrics, err := agn.storage.GetBatch()
	if err != nil {
		return nil, err
	}

	if len(allMetrics) == 0 {
		return nil, errStorageIsEmpty
	}

	for i := range allMetrics {
		if errUpdateHash := allMetrics[i].UpdateHash(agn.config.HashKey); errUpdateHash != nil {
			return nil, errUpdateHash
		}
	}

	mj, err := json.Marshal(allMetrics)
	if err != nil {
		return nil, err
	}

	return mj, nil
}
