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

	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
)

const (
	monitorInterval  = 30 * time.Second
	profileRetention = 10
)

type MemoryMonitor struct {
	profile *config.Profile
}

func NewMemoryMonitor(profile *config.Profile) *MemoryMonitor {
	return &MemoryMonitor{
		profile: profile,
	}
}

func (mm *MemoryMonitor) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(monitorInterval)
	defer ticker.Stop()
	defer wg.Done()
	slog.Debug(fmt.Sprintf("memory monitor: started and will run every %v", monitorInterval))
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			threshold := mm.profile.RuntimeMemoryProfileThreshold.Load()
			mm.checkMemory(threshold)
		}
	}
}

func (mm *MemoryMonitor) checkMemory(memoryThreshold uint64) {
	// memoryThreshold == 0 means no need to monitor
	if memoryThreshold == 0 {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if m.Sys >= memoryThreshold {
		now := time.Now().Unix()

		// Dump heap profile
		heapFileName := filepath.Join(mm.profile.DataDir, fmt.Sprintf("memory_dump_%d.prof", now))
		slog.Info("memory monitor: memory allocated exceeds memory threshold. will dump pprof memory and goroutine profile", "memoryUsage", m.Sys, "memoryThreshold", memoryThreshold)
		slog.Info("memory monitor: dumping pprof memory profile", "fileName", heapFileName)

		if err := dumpHeapProfile(heapFileName); err != nil {
			slog.Info("memory monitor: could not dump memory profile", "fileName", heapFileName, log.BBError(err))
			// Continue to attempt goroutine dump even if heap dump fails
		} else {
			if err := retainMostRecentFiles(mm.profile.DataDir, profileRetention); err != nil {
				slog.Info("memory monitor: failed to cleanup old dump files after heap dump", log.BBError(err))
			}
		}

		// Dump goroutine profile
		goroutineFileName := filepath.Join(mm.profile.DataDir, fmt.Sprintf("goroutine_dump_%d.prof", now))
		slog.Info("memory monitor: dumping pprof goroutine profile", "fileName", goroutineFileName)

		if err := dumpGoroutineProfile(goroutineFileName); err != nil {
			slog.Info("memory monitor: could not dump goroutine profile", "fileName", goroutineFileName, log.BBError(err))
		} else {
			if err := retainMostRecentFiles(mm.profile.DataDir, profileRetention); err != nil {
				slog.Info("memory monitor: failed to cleanup old dump files after goroutine dump", log.BBError(err))
			}
		}
	}
}

func retainMostRecentFiles(dir string, retainCount int) error {
	// Handle memory dumps
	if err := retainMostRecentFilesByPattern(dir, "memory_dump_*.prof", retainCount); err != nil {
		return errors.Errorf("failed to retain memory dumps: %v", err)
	}

	// Handle goroutine dumps
	if err := retainMostRecentFilesByPattern(dir, "goroutine_dump_*.prof", retainCount); err != nil {
		return errors.Errorf("failed to retain goroutine dumps: %v", err)
	}

	return nil
}

func dumpHeapProfile(fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return errors.Errorf("could not create memory profile file: %v", err)
	}
	defer f.Close()

	return pprof.WriteHeapProfile(f)
}

func dumpGoroutineProfile(fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return errors.Errorf("could not create goroutine profile file: %v", err)
	}
	defer f.Close()

	goroutineProfile := pprof.Lookup("goroutine")
	if goroutineProfile == nil {
		return errors.New("could not look up goroutine profile")
	}

	return goroutineProfile.WriteTo(f, 0)
}

func retainMostRecentFilesByPattern(dir, pattern string, retainCount int) error {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return err
	}

	type fileInfo struct {
		path      string
		timestamp int64
	}

	files := make([]fileInfo, 0, len(matches))
	for _, path := range matches {
		// Extract timestamp from filename
		base := filepath.Base(path)
		// Find the last underscore and extract everything after it until .prof
		if idx := strings.LastIndex(base, "_"); idx != -1 {
			timestampStr := strings.TrimSuffix(base[idx+1:], ".prof")
			timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				slog.Info("memory monitor: failed to parse timestamp from filename", "file", base, log.BBError(err))
				continue
			}
			files = append(files, fileInfo{path: path, timestamp: timestamp})
		}
	}

	// Sort files by timestamp in descending order (most recent first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].timestamp > files[j].timestamp
	})

	// Remove older files beyond retention count
	for i := retainCount; i < len(files); i++ {
		slog.Info("memory monitor: removing old dump file", "file", files[i].path)
		if err := os.Remove(files[i].path); err != nil {
			slog.Info("memory monitor: failed to remove old dump file", "file", files[i].path, log.BBError(err))
		}
	}

	return nil
}
