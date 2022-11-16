package agent

import (
	"encoding/json"
	"errors"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/dcaiman/YP_GO/internal/pkg/metric"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

const (
	// Randomises it's value from 0 to 100.
	RandomValue = "RandomValue"

	// Shows total amount of RAM on this system got from mem-packet.
	TotalMemory = "TotalMemory"

	// Shows the kernel's notion of free memory got from mem-packet.
	FreeMemory = "FreeMemory"

	// Shows CPU cores individual usage in percents got from cpu-packet.
	// Needed to add the number of CPU in the end to get "CPUutilization1", "CPUutilization2" (according to cores number).
	CPUutilization = "CPUutilization"
)

// Collects runtime metric by it's name.
func (agn *agent) getRuntimeMetricValue(name string, mem *runtime.MemStats) float64 {
	return reflect.Indirect(reflect.ValueOf(mem)).FieldByName(name).Convert(reflect.TypeOf(0.0)).Float()
}

// Collects custom metric by it's name. Implements individual algorithms for custom metrics.
func (agn *agent) getCustomMetricValue(name string) (float64, error) {
	switch name {
	case RandomValue:
		return 100 * rand.Float64(), nil
	case TotalMemory:
		vm, err := mem.VirtualMemory()
		if err != nil {
			return 0, err
		}
		return float64(vm.Total), nil
	case FreeMemory:
		vm, err := mem.VirtualMemory()
		if err != nil {
			return 0, err
		}
		return float64(vm.Free), nil
	}
	if strings.Contains(name, CPUutilization) {
		procUsage, err := cpu.Percent(0, true)
		if err != nil {
			return 0, err
		}
		num, err := strconv.ParseInt(strings.TrimPrefix(name, CPUutilization), 10, 64)
		if err != nil {
			return 0, err
		}
		if int(num) > len(procUsage) {
			return 0, errors.New("cannot get <" + name + ">: core number error")
		}
		return procUsage[num-1], nil
	}
	return 0, errors.New("cannot get: unsupported metric <" + name + ">")
}

// Gives a batch of all storaged metrics in json format.
func (agn *agent) getStorageBatch() ([]byte, error) {
	allMetrics, err := agn.storage.GetBatch()
	if err != nil {
		return nil, err
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
	if reflect.DeepEqual(mj, []byte("[]")) {
		return nil, errors.New("cannot get storage batch: storage is empty")
	}
	return mj, nil
}

// Resets all counters in agent storage.
func (agn *agent) resetCounters() error {
	for _, name := range agn.Counters {
		m := metric.Metric{
			ID:    name,
			MType: Counter,
		}
		if err := agn.storage.UpdateMetric(&m); err != nil {
			return err
		}
	}
	return nil
}

// Resets individual counter.
func (agn *agent) resetCounter(name string) error {
	if err := agn.storage.UpdateMetric(&metric.Metric{
		ID:    name,
		MType: Counter,
	}); err != nil {
		return err
	}
	return nil
}
