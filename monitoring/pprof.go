package monitoring

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/hashicorp/go-hclog"
)

type Profile struct {
	Logger    hclog.Logger
	Goroutine *pprof.Profile
	Heap      *pprof.Profile
}

func (profile *Profile) SetupPprofProfiles() {
	// Get the directory of the executable
	exeDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		profile.Logger.Error("Error getting executable directory", "error", err)
		return
	}

	// Directory to store profiling files
	profileDir := filepath.Join(exeDir, "../pprof/heap")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		profile.Logger.Error("Error creating profile directory", "error", err)
		return
	}

	go collectHeapProfile(profile.Heap, profileDir, profile.Logger)
}

// collectGoroutineProfile periodically collects goroutine profiles
func collectHeapProfile(heapProfile *pprof.Profile, profileDir string, logger hclog.Logger) {
	for {
		// Capture heap profile
		heapFileName := fmt.Sprintf(filepath.Join(profileDir, "heap_%s.prof"), time.Now().Format("20060102-1504"))
		heapFile, err := os.Create(heapFileName)
		if err != nil {
			logger.Error("Error creating heap profile file:", err)
			return
		}
		if err := pprof.WriteHeapProfile(heapFile); err != nil {
			logger.Error("Error writing heap profile:", err)
		}
		heapFile.Close()
		logger.Info("Heap profile file created:", heapFileName)

		time.Sleep(time.Minute)
	}
}
