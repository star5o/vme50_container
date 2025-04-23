package main

import (
	"fmt"
	"os"

	// Update import paths based on the new root structure and module path
	"github.com/star5o/vme50_container/command"
	"github.com/star5o/vme50_container/container"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vme50-container",
	Short: "VME50 Container - A simple container runtime for learning",
	Long:  `vme50-container is a simple container runtime for educational purposes.`,
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Internal command for container initialization, not exposed to the user directly.
var containerInitCmd = &cobra.Command{
	Use:    "container-init",
	Short:  "Internal - Initializes the container environment",
	Hidden: true, // Hide from help text
	Args:   cobra.MinimumNArgs(1), // Requires at least the '--' separator and command
	RunE: func(cmd *cobra.Command, args []string) error { // Use RunE for error return
		return container.ContainerInitProcess(args)
	},
}

func init() {
	// Check for root privileges early
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "Error: This program must be run as root.")
		os.Exit(1)
	}
	// Add run command
	rootCmd.AddCommand(command.GetRunCmd()) // Use the getter function
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Cobra prints errors by default
		os.Exit(1)
	}
}

func main() {
	// Check if the first argument indicates we are running the internal init command
	if len(os.Args) > 1 && os.Args[1] == containerInitCmd.Use {
		// Execute the internal init command directly
		if err := container.ContainerInitProcess(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "[Container Init Error] %v\n", err)
			os.Exit(1)
		}
	} else {
		// Execute the regular Cobra CLI
		Execute()
	}
} 