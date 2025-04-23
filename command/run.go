package command

import (
	"fmt"
	"os"
	//"os/exec" // Now handled in container package
	// Update import path based on the new root structure and module path
	"github.com/star5o/vme50_container/container" 
	"github.com/google/uuid" // For generating container IDs

	"github.com/spf13/cobra"
)

var (
	rootfsPath   string
	containerCmd []string
	hostname     string
	cpuShares    int
	memoryLimit  string
)

var runCmd = &cobra.Command{
	Use:   "run [flags] [command]",
	Short: "Run a command in a new container",
	Long:  `Creates and runs a new container with specified isolation settings.`,
	Args:  cobra.MinimumNArgs(1), // Ensure at least one argument (the command) is provided
	Run: func(cmd *cobra.Command, args []string) {
		containerCmd = args
		fmt.Printf("[Run Command] Starting container...\n")
		fmt.Printf("[Run Command]   RootFS: %s\n", rootfsPath)
		if hostname != "" {
			fmt.Printf("[Run Command]   Hostname: %s\n", hostname)
		}
		if cpuShares > 0 {
			fmt.Printf("[Run Command]   CPU Shares: %d\n", cpuShares)
		}
		if memoryLimit != "" {
			fmt.Printf("[Run Command]   Memory Limit: %s\n", memoryLimit)
		}
		fmt.Printf("[Run Command]   Command: %v\n", containerCmd)

		// 1. Validate rootfsPath exists and is a directory (Basic Check)
		fileInfo, err := os.Stat(rootfsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[Run Command] Error validating rootfs path '%s': %v\n", rootfsPath, err)
			os.Exit(1)
		}
		if !fileInfo.IsDir() {
			fmt.Fprintf(os.Stderr, "[Run Command] Error: rootfs path '%s' is not a directory\n", rootfsPath)
			os.Exit(1)
		}

		// 2. Generate container ID
		containerID := uuid.New().String()[:8] // Use first 8 chars for brevity
		fmt.Printf("[Run Command] Generated Container ID: %s\n", containerID)

		// 3. Setup Cgroups
		if err := container.SetupCgroups(containerID, cpuShares, memoryLimit); err != nil {
			fmt.Fprintf(os.Stderr, "[Run Command] Error setting up Cgroups: %v\n", err)
			os.Exit(1)
		}
		// Ensure cleanup happens even on errors later
		defer container.CleanupCgroups(containerID)

		// 4. Prepare and start container process (with namespaces, chroot)
		fmt.Println("[Run Command] Cgroups setup complete. Starting container process...")
		if err := container.StartContainerProcess(rootfsPath, hostname, containerCmd); err != nil {
			// Error already printed in StartContainerProcess
			fmt.Fprintf(os.Stderr, "[Run Command] Failed to start container process.\n")
			os.Exit(1) // Exit after cleanup deferred above
		}

		// 5. Wait for process (handled by StartContainerProcess -> cmd.Run())
		// 6. Cleanup Cgroups (handled by defer)
		fmt.Println("[Run Command] Container finished.")
	},
}

func init() {
	runCmd.Flags().StringVar(&rootfsPath, "rootfs", "", "Path to the container's root filesystem (required)")
	runCmd.MarkFlagRequired("rootfs") // Make --rootfs mandatory

	runCmd.Flags().StringVar(&hostname, "hostname", "", "Set container hostname (UTS Namespace)")
	runCmd.Flags().IntVar(&cpuShares, "cpu-shares", 0, "CPU shares (relative weight, Cgroup v2 cpu.weight)")
	runCmd.Flags().StringVar(&memoryLimit, "memory-limit", "", "Memory limit (e.g., '512M', '1G', Cgroup v2 memory.max)")

	// Hide completion command
	runCmd.CompletionOptions.DisableDefaultCmd = true
}

// GetRunCmd returns the run command instance
func GetRunCmd() *cobra.Command {
	return runCmd
} 