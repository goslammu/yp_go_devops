package agent

import (
	"errors"
	"log"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/goslammu/yp_go_devops/internal/pkg/metric"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

var (
	errStorageIsEmpty    = errors.New("storage is empty")
	errUnsupportedMetric = errors.New("unsupported metric")
)

var randomMaxValue float64 = 100

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
func getRuntimeMetricValue(name string, mem *runtime.MemStats) float64 {
	return reflect.Indirect(reflect.ValueOf(mem)).FieldByName(name).Convert(reflect.TypeOf(0.0)).Float()
}

// Collects custom metric by it's name. Implements individual algorithms for custom metrics.
func getCustomMetricValue(name string) (float64, error) {
	switch {
	case name == RandomValue:
		return getRandomValue()
	case name == TotalMemory:
		return getTotalMemory()
	case name == FreeMemory:
		return getFreeMemory()
	case strings.Contains(name, CPUutilization):
		return getCPUutilization(name)
	default:
		return 0, errUnsupportedMetric
	}
}

func getRandomValue() (float64, error) {
	return randomMaxValue * rand.Float64(), nil
}

func getTotalMemory() (float64, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}

	return float64(vm.Total), nil
}

func getFreeMemory() (float64, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return float64(vm.Free), nil
}

func getCPUutilization(name string) (float64, error) {
	num, err := strconv.ParseInt(strings.TrimPrefix(name, CPUutilization), 10, 64)
	if err != nil {
		return 0, err
	}

	procUsage, err := cpu.Percent(0, true)
	if err != nil {
		return 0, err
	}

	if int(num) > len(procUsage) {
		return 0, errUnsupportedMetric
	}

	return procUsage[num-1], nil
}

// Resets all counters in agent storage.
func (agn *agent) resetCounters() error {
	for _, v := range agn.Counters {
		if err := agn.resetCounter(v); err != nil {
			log.Println(err)
		}
	}

	return nil
}

// Resets individual counter.
func (agn *agent) resetCounter(name string) error {
	return agn.storage.UpdateMetric(&metric.Metric{
		ID:    name,
		MType: Counter,
	})
}
