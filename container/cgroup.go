package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall" // Required for syscall.Rmdir
)

const (
	cgroupRoot      = "/sys/fs/cgroup"
	myDockerCgroup  = "my-docker"
	cgroupProcs     = "cgroup.procs"
	cpuWeightFile   = "cpu.weight"
	memoryMaxFile   = "memory.max"
	defaultCPUWeight = 100 // Default value for cpu.weight in cgroup v2
)

// parseMemoryLimit converts memory strings like "512M", "1G" into bytes.
// Returns bytes and error.
func parseMemoryLimit(limit string) (int64, error) {
	limit = strings.TrimSpace(limit)
	if limit == "" {
		return 0, nil // No limit
	}

	unit := strings.ToLower(string(limit[len(limit)-1]))
	valueStr := limit[:len(limit)-1]

	var multiplier int64 = 1
	switch unit {
	case "b":
		valueStr = limit // "1024b" case
		multiplier = 1
	case "k":
		multiplier = 1024
	case "m":
		multiplier = 1024 * 1024
	case "g":
		multiplier = 1024 * 1024 * 1024
	default:
		// Assume bytes if no unit
		valueStr = limit
		multiplier = 1
	}

	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory limit format: %s", limit)
	}

	return value * multiplier, nil
}

// getContainerCgroupPath returns the full path for the container's cgroup directory.
func getContainerCgroupPath(containerID string) string {
	return filepath.Join(cgroupRoot, myDockerCgroup, containerID)
}

// SetupCgroups configures cgroups v2 limits for the container.
// It creates the necessary cgroup directory and applies CPU and memory limits.
func SetupCgroups(containerID string, cpuShares int, memoryLimit string) error {
	containerCgroupPath := getContainerCgroupPath(containerID)
	logPrefix := fmt.Sprintf("[Cgroup Setup %s]", containerID)

	fmt.Printf("%s Creating cgroup directory: %s\n", logPrefix, containerCgroupPath)
	if err := os.MkdirAll(containerCgroupPath, 0755); err != nil {
		return fmt.Errorf("%s failed to create cgroup directory %s: %w", logPrefix, containerCgroupPath, err)
	}

	// Add parent process to the cgroup to ensure child inherits it
	pid := os.Getpid()
	procsPath := filepath.Join(containerCgroupPath, cgroupProcs)
	fmt.Printf("%s Adding PID %d to %s\n", logPrefix, pid, procsPath)
	if err := os.WriteFile(procsPath, []byte(strconv.Itoa(pid)), 0644); err != nil {
		// Attempt cleanup before returning error
		_ = syscall.Rmdir(containerCgroupPath)
		return fmt.Errorf("%s failed to write PID to %s: %w", logPrefix, procsPath, err)
	}

	// Set CPU weight
	if cpuShares > 0 {
		// Cgroup v2 cpu.weight is 1-10000, default 100.
		// We directly use the provided shares value here, assuming it's intended for cpu.weight.
		// Add validation if specific range mapping is needed.
		if cpuShares < 1 || cpuShares > 10000 {
			fmt.Printf("%s Warning: cpuShares %d is outside the recommended range [1, 10000] for cpu.weight\n", logPrefix, cpuShares)
			// Clamp or return error based on desired behavior. Sticking to user value for now.
		}
		weightPath := filepath.Join(containerCgroupPath, cpuWeightFile)
		weightStr := strconv.Itoa(cpuShares)
		fmt.Printf("%s Setting %s to %s\n", logPrefix, weightPath, weightStr)
		if err := os.WriteFile(weightPath, []byte(weightStr), 0644); err != nil {
			// Attempt cleanup
			_ = syscall.Rmdir(containerCgroupPath)
			return fmt.Errorf("%s failed to write %s: %w", logPrefix, weightPath, err)
		}
	} else {
        // Even if no shares are specified, ensure default is set if file exists or needed
        // For simplicity now, we only write if user specified > 0
         fmt.Printf("%s No CPU shares specified, using cgroup default.\n", logPrefix)
	}


	// Set Memory limit
	if memoryLimit != "" {
		memBytes, err := parseMemoryLimit(memoryLimit)
		if err != nil {
			// Attempt cleanup
			_ = syscall.Rmdir(containerCgroupPath)
			return fmt.Errorf("%s failed to parse memory limit '%s': %w", logPrefix, memoryLimit, err)
		}

		if memBytes > 0 {
			memMaxPath := filepath.Join(containerCgroupPath, memoryMaxFile)
			memMaxStr := strconv.FormatInt(memBytes, 10)
			fmt.Printf("%s Setting %s to %s bytes\n", logPrefix, memMaxPath, memMaxStr)
			if err := os.WriteFile(memMaxPath, []byte(memMaxStr), 0644); err != nil {
				// Attempt cleanup
				_ = syscall.Rmdir(containerCgroupPath)
				return fmt.Errorf("%s failed to write %s: %w", logPrefix, memMaxPath, err)
			}
		} else {
             fmt.Printf("%s Parsed memory limit resulted in 0 bytes, no limit applied.\n", logPrefix)
        }
	} else {
         fmt.Printf("%s No memory limit specified.\n", logPrefix)
    }


	fmt.Printf("%s Cgroup setup complete for %s\n", logPrefix, containerID)
	return nil
}

// CleanupCgroups removes the cgroup directory created for the container.
func CleanupCgroups(containerID string) error {
	containerCgroupPath := getContainerCgroupPath(containerID)
	logPrefix := fmt.Sprintf("[Cgroup Cleanup %s]", containerID)
	fmt.Printf("%s Removing cgroup directory: %s\n", logPrefix, containerCgroupPath)

	// Attempt to remove the directory.
	// Note: Cgroup directories can sometimes be tricky to remove if processes are still listed.
	// In a robust implementation, one might need to move processes out first.
	// For this educational tool, a simple Rmdir is attempted.
	if err := syscall.Rmdir(containerCgroupPath); err != nil {
		// Check if the error is because the file/dir doesn't exist (already cleaned up?)
        if os.IsNotExist(err) {
             fmt.Printf("%s Cgroup directory %s does not exist, cleanup skipped.\n", logPrefix, containerCgroupPath)
             return nil // Not an error in this case
        }
		fmt.Fprintf(os.Stderr, "%s Warning: failed to remove cgroup directory %s: %v\n", logPrefix, containerCgroupPath, err)
		return fmt.Errorf("%s failed to remove cgroup directory %s: %w", logPrefix, containerCgroupPath, err)
	}
	fmt.Printf("%s Cgroup directory removed.\n", logPrefix, containerCgroupPath)
	return nil
} 