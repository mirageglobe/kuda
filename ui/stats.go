package ui

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
)

type statsInfo struct {
	memMB     float64
	cpuPct    float64
	prevCPUNs int64
}

func (s statsInfo) update() statsInfo {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	nowNs := ru.Utime.Sec*1e9 + int64(ru.Utime.Usec)*1e3 +
		ru.Stime.Sec*1e9 + int64(ru.Stime.Usec)*1e3

	var cpuPct float64
	if s.prevCPUNs > 0 && nowNs > s.prevCPUNs {
		cpuPct = float64(nowNs-s.prevCPUNs) * 100 / float64(time.Second)
		if cpuPct > 100 {
			cpuPct = 100
		}
	}

	return statsInfo{
		memMB:     float64(ms.Alloc) / 1024 / 1024,
		cpuPct:    cpuPct,
		prevCPUNs: nowNs,
	}
}

func (s statsInfo) String() string {
	return fmt.Sprintf("cpu %.1f%%  mem %.1f MB", s.cpuPct, s.memMB)
}
