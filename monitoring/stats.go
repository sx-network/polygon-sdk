package monitoring

import (
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shirou/gopsutil/mem"
)

type Stats struct {
	Logger                hclog.Logger
	IsMemStressTestEnable bool
	DelayInSecondsStats   uint64
	Threshold             float64
}

func (stats *Stats) TrackMemoryUsage() {
	if stats.IsMemStressTestEnable {
		go func() {
			heapMemoryStressTest(stats.Logger)
		}()
	}

	for {
		// Get the virtual memory stats
		vm, err := mem.VirtualMemory()
		if err != nil {
			stats.Logger.Error("Error getting VirtualMemory", err)
			time.Sleep(time.Second * time.Duration(stats.DelayInSecondsStats)) // Wait for the specified interval
			continue
		}

		// Calculate the memory usage percentage
		memUsage := float64(vm.Used) / float64(vm.Total)
		stats.Logger.Info(fmt.Sprintf("Memory usage: %.2f%% (%v bytes), Total Memory: %v bytes, Threshold: %.2f%%", memUsage*100, vm.Used, vm.Total, stats.Threshold*100))

		// Check if memory usage exceeds the threshold
		if memUsage > stats.Threshold {
			stats.Logger.Info("Memory usage exceeds threshold. Performing graceful shutdown...")
			gracefulShutdown(stats.Logger)
			return
		}

		time.Sleep(time.Second * time.Duration(stats.DelayInSecondsStats)) // Wait for the specified interval
	}
}

func gracefulShutdown(logger hclog.Logger) {
	logger.Warn("Graceful shutdown completed")
	os.Exit(0)
}

func heapMemoryStressTest(logger hclog.Logger) {
	var memorySlice [][]byte

	// Loop 1000 times to force memory allocations
	for i := 0; i < 1000; i++ {
		// Allocate a large slice of bytes (100 MB)
		memory := make([]byte, 1024*1024*1000) // 100 MB
		memorySlice = append(memorySlice, memory)

		logger.Info(fmt.Sprintf("Iteration %d - Allocated %d MB", i+1, len(memorySlice)*100))
		time.Sleep(time.Second * 5)
	}
}
