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
	Logger         hclog.Logger
	IsEnable       bool
	DelayInSeconds uint64
	Goroutine      *pprof.Profile
	Heap           *pprof.Profile
}

/*
	SetupPprofProfiles configures the pprof profiles for monitoring different aspects of the program.
	It creates a directory to store the profile files and starts collecting
	the specified profiles in separate goroutines
*/
func (profile *Profile) SetupPprofProfiles() {
	// Get the directory of the executable
	exeDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		profile.Logger.Error("Error getting executable directory", "error", err)
		return
	}

	// Directory to store profiling files
	profileDir := filepath.Join(exeDir, "../pprof")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		profile.Logger.Error("Error creating profile directory", "error", err)
		return
	}

	// Start collecting profiles
	if profile.IsEnable {
		go collectProfile(profile.Heap, filepath.Join(profileDir, "heap"), profile.Logger, profile.DelayInSeconds)
		go collectProfile(profile.Goroutine, filepath.Join(profileDir, "goroutine"), profile.Logger, profile.DelayInSeconds)
	}
}

/*
	Collects the specified pprof profile data at regular intervals
	and writes it to a new file in the specified profile directory
*/
func collectProfile(profile *pprof.Profile, profileDir string, logger hclog.Logger, delayInSeconds uint64) {
	for {
		// Generate a unique filename for the profile
		profileFileName := fmt.Sprintf(filepath.Join(profileDir, "%s_%s.prof"), profile.Name(), time.Now().Format("20060102-1504"))

		// Create the profile directory if it does not exist
		if err := os.MkdirAll(profileDir, 0755); err != nil {
			logger.Error("Error creating profile directory:", err)
			return
		}

		// Create a new file to write the profile data
		profileFile, err := os.Create(profileFileName)
		if err != nil {
			logger.Error("Error creating profile file:", err)
			return
		}

		// Write the profile data to the file
		if err := profile.WriteTo(profileFile, 0); err != nil {
			logger.Error("Error writing profile data:", err)
		}

		profileFile.Close()
		logger.Info(fmt.Sprintf("Profile file created: %s", profileFileName))
		time.Sleep(time.Second * time.Duration(delayInSeconds))
	}
}
