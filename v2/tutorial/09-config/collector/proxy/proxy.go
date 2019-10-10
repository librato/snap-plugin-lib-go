package proxy

import (
	"fmt"
	"time"

	data2 "github.com/librato/snap-plugin-lib-go/v2/tutorial/09-config/collector/data"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

const defaultCPUMeasurementTime = 1 * time.Second

type Proxy interface {
	ProcessesInfo() ([]data2.ProcessInfo, error)
	TotalCpuUsage(time.Duration) (float64, error)
	TotalMemoryUsage() (float64, error)
}

type proxyCollector struct{}

func New() Proxy {
	return &proxyCollector{}
}

func (p proxyCollector) ProcessesInfo() ([]data2.ProcessInfo, error) {
	procInfo := []data2.ProcessInfo{}

	processesData, err := process.Processes()
	if err != nil {
		return procInfo, fmt.Errorf("can't obtain list of processes: %v", err)
	}

	for _, proc := range processesData {
		name, err := proc.Name()
		if err != nil {
			continue
		}

		cpuPerc, err := proc.CPUPercent()
		if err != nil {
			continue
		}

		memPerc, err := proc.MemoryPercent()
		if err != nil {
			continue
		}

		procInfo = append(procInfo, data2.ProcessInfo{
			ProcessName: name,
			CpuUsage:    cpuPerc,
			MemoryUsage: float64(memPerc),
			PID:         proc.Pid,
		})
	}

	return procInfo, nil
}

func (p proxyCollector) TotalCpuUsage(timeout time.Duration) (float64, error) {
	totalCpu, err := cpu.Percent(timeout, false)
	if err != nil {
		return 0, fmt.Errorf("can't obtain cpu information: %v", err)
	}
	if len(totalCpu) == 0 {
		return 0, fmt.Errorf("unexpected cpu information: %v", err)
	}

	return totalCpu[0], nil
}

func (p proxyCollector) TotalMemoryUsage() (float64, error) {
	memoryInfo, err := mem.VirtualMemory()
	if err != nil {
		return 0, fmt.Errorf("can't obtain memory information: %v", err)
	}

	return memoryInfo.UsedPercent, nil
}