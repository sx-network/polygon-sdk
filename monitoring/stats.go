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
	Logger hclog.Logger
}

func (stats *Stats) TrackMemoryUsage() {
	threshold := 0.8 // 80% threshold for memory usage

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

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
