package monitoring

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shirou/gopsutil/mem"
)

type Stats struct {
	Logger                hclog.Logger
	IsMemStressTestEnable bool
}

func (stats *Stats) TrackMemoryUsage() {
	threshold := 0.8 // 80% threshold for memory usage

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	if stats.IsMemStressTestEnable {
		go func() {
			heapMemoryStressTest(stats.Logger)
		}()
	}

	for {
		<-ticker.C

		// Get the virtual memory stats
		vm, err := mem.VirtualMemory()
		if err != nil {
			stats.Logger.Error("Error setting VirtualMemory", err)
			continue
		}

		// Calculate the memory usage percentage
		memUsage := float64(vm.Used) / float64(vm.Total)
		stats.Logger.Info(fmt.Sprintf("Memory usage: %.2f%% (%v bytes), Total Memory: %v bytes, Threshold: %.2f%%", memUsage*100, vm.Used, vm.Total, threshold*100))

		// Check if memory usage exceeds the threshold
		if memUsage > threshold {
			stats.Logger.Info("Memory usage exceeds threshold. Performing graceful shutdown...")
			restart(stats.Logger)
			return
		}
	}
}

func restart(logger hclog.Logger) {
	// Create a command to restart the sxnode service using systemctl
	restartCmd := exec.Command("sudo", "systemctl", "restart", "sxnode.service")
	restartCmd.Stdout = os.Stdout
	restartCmd.Stderr = os.Stderr

	if err := restartCmd.Start(); err != nil {
		logger.Error("Error restarting service:", err)
		return
	}

	logger.Info("Service restarted successfully")
}

func heapMemoryStressTest(logger hclog.Logger) {
	var memorySlice [][]byte

	// Loop 1000 times to force memory allocations
	for i := 0; i < 1000; i++ {
		// Allocate a large slice of bytes (100 MB)
		memory := make([]byte, 1024*1024*1000) // 100 MB
		memorySlice = append(memorySlice, memory)

		logger.Info(fmt.Sprintf("Iteration %d - Allocated MB %d", i+1, len(memorySlice)*100))
		time.Sleep(time.Second * 5)
	}
}
