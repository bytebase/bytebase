package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
)

const (
	monitorInterval = 30 * time.Second
)

type MemoryMonitor struct {
	profile         *config.Profile
	memoryThreshold uint64
}

func NewMemoryMonitor(profile *config.Profile, memoryThreshold uint64) *MemoryMonitor {
	return &MemoryMonitor{
		profile:         profile,
		memoryThreshold: memoryThreshold,
	}
}

func (m *MemoryMonitor) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(monitorInterval)
	defer ticker.Stop()
	defer wg.Done()
	slog.Debug(fmt.Sprintf("memory monitor: started and will run every %v", monitorInterval))
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkMemory(m.memoryThreshold)
		}
	}
}

func (r *MemoryMonitor) checkMemory(memoryThreshold uint64) {
	// memoryThreshold == 0 means no need to monitor
	if memoryThreshold == 0 {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if m.Sys >= memoryThreshold {
		now := time.Now().UnixMicro()
		heapFileName := filepath.Join(r.profile.DataDir, fmt.Sprintf("memory_dump_%d.prof", now))
		slog.Info("memory monitor: memory allocated exceeds memory threshold. will dump pprof memory and goroutine profile", "memoryUsage", m.Sys, "memoryThreshold", memoryThreshold)
		slog.Info("memory monitor: dumping pprof memory profile", "fileName", heapFileName)
		fHeap, err := os.Create(heapFileName)
		if err != nil {
			slog.Info("memory monitor: could not create memory profile file", "fileName", heapFileName, log.BBError(err))
			// Continue to attempt goroutine dump even if heap dump fails
		} else {
			defer fHeap.Close() // Close the file promptly
			// Write the heap profile
			if err := pprof.WriteHeapProfile(fHeap); err != nil {
				slog.Info("memory monitor: could not write memory profile", "fileName", heapFileName, log.BBError(err))
			} else {
				slog.Info("memory monitor: successfully wrote memory profile", "fileName", heapFileName)
			}
		}

		// It's often useful to also dump goroutine state
		goroutineFileName := filepath.Join(r.profile.DataDir, fmt.Sprintf("goroutine_dump_%d.prof", now))
		slog.Info("memory monitor: dumping pprof goroutine profile", "fileName", goroutineFileName)
		fGoroutine, err := os.Create(goroutineFileName)
		if err != nil {
			slog.Info("memory monitor: could not create goroutine profile file", "fileName", goroutineFileName, log.BBError(err))
			return // Return if we cannot create the goroutine file either
		}
		defer fGoroutine.Close()

		goroutineProfile := pprof.Lookup("goroutine")
		if goroutineProfile == nil {
			slog.Info("memory monitor: could not look up goroutine profile")
			return
		}
		if err := goroutineProfile.WriteTo(fGoroutine, 0); err != nil {
			slog.Info("memory monitor: could not write goroutine profile", "fileName", goroutineFileName, log.BBError(err))
		} else {
			slog.Info("memory monitor: successfully wrote goroutine profile", "fileName", goroutineFileName)
		}
	}
}
