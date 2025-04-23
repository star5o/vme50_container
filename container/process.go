package container

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// StartContainerProcess prepares and starts the container's init process.
// It sets up the command with necessary namespaces, chroot, and arguments.
func StartContainerProcess(rootfsPath, containerHostname string, userCmdArgs []string) error {
	logPrefix := "[Parent Process]"

	// Path to the currently running executable
	selfPath := os.Args[0]

	// Arguments for the init process
	// Format: my-docker container-init <hostname> <user-command> <user-args...>
	initArgs := []string{"container-init"}
	if containerHostname != "" {
		initArgs = append(initArgs, "--hostname", containerHostname) // Pass hostname as a flag
	}
	initArgs = append(initArgs, "--") // Separator for user command
	initArgs = append(initArgs, userCmdArgs...)

	fmt.Printf("%s Preparing init command: %s %v\n", logPrefix, selfPath, initArgs)

	cmd := exec.Command(selfPath, initArgs...)

	// Set namespaces and chroot
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: GetCloneFlags(), // Get flags from namespace.go
		Chroot:     rootfsPath,
		// TODO: Add user namespace mappings if needed (requires more setup)
	}

	// Inherit standard streams
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("%s Starting container init process...\n", logPrefix)
	err := cmd.Run() // Start and wait for the process
	if err != nil {
		// cmd.Run() returns ExitError which includes stderr output,
		// so printing the error usually gives enough context.
		fmt.Fprintf(os.Stderr, "%s Container init process failed: %v\n", logPrefix, err)
		return fmt.Errorf("container process failed: %w", err)
	}

	fmt.Printf("%s Container process finished successfully.\n", logPrefix)
	return nil
}

// ContainerInitProcess is the entry point for the code running inside the container.
// It performs mounts, sets hostname, and executes the user's command.
func ContainerInitProcess(initArgs []string) error {
	logPrefix := "[Container Init]"
	fmt.Printf("%s Starting initialization inside container...\n", logPrefix)

	// --- Argument Parsing for Init ---
	var containerHostname string
	var userCmdAndArgs []string

	hostnameFlagFound := false
	separatorFound := false
	for i, arg := range initArgs {
		if separatorFound {
			userCmdAndArgs = append(userCmdAndArgs, arg)
			continue
		}
		if hostnameFlagFound {
			containerHostname = arg
			hostnameFlagFound = false // Reset flag
			continue
		}
		if arg == "--hostname" {
			hostnameFlagFound = true
			continue
		}
		if arg == "--" {
			separatorFound = true
			continue
		}
		// Ignore unexpected args before separator
	}

	if !separatorFound || len(userCmdAndArgs) == 0 {
		return fmt.Errorf("%s invalid arguments for init process: missing separator '--' or user command", logPrefix)
	}
	userCommand := userCmdAndArgs[0]
	userArgs := userCmdAndArgs // syscall.Exec expects command as args[0]

	fmt.Printf("%s   User command: %v\n", logPrefix, userArgs)
	if containerHostname != "" {
		fmt.Printf("%s   Setting hostname to: %s\n", logPrefix, containerHostname)
	}

	// --- Mounts ---
	// Mount /proc (essential)
	// Note: The target directory /proc must exist in the rootfs provided by the user.
	fmt.Printf("%s Mounting /proc...\n", logPrefix)
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("%s failed to mount /proc: %w. Ensure /proc directory exists in rootfs", logPrefix, err)
	}
	defer func() {
		fmt.Printf("%s Unmounting /proc...\n", logPrefix)
		if err := syscall.Unmount("/proc", 0); err != nil {
			fmt.Fprintf(os.Stderr, "%s Warning: failed to unmount /proc: %v\n", logPrefix, err)
		}
	}()

	// TODO: Mount /dev (tmpfs), /sys (read-only bind?) for better compatibility

	// --- Hostname ---
	if containerHostname != "" {
		fmt.Printf("%s Setting hostname...\n", logPrefix)
		if err := syscall.Sethostname([]byte(containerHostname)); err != nil {
			return fmt.Errorf("%s failed to set hostname: %w", logPrefix, err)
		}
	}

	// --- Chdir ---
	// Chroot is handled by SysProcAttr.Chroot in the parent.
	// We need to change directory to the new root.
	fmt.Printf("%s Changing directory to /...\n", logPrefix)
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("%s failed to chdir to /: %w", logPrefix, err)
	}

	// --- Execute User Command ---
	fmt.Printf("%s Looking up executable path for: %s\n", logPrefix, userCommand)
	// Simple lookup: try absolute path first, then relative to root.
	// A real implementation would search PATH.
	execPath := userCommand
	if !strings.HasPrefix(userCommand, "/") {
		// Attempt simple lookup relative to root
		// TODO: Implement proper PATH lookup inside the chroot environment
		potentialPath, err := exec.LookPath(userCommand) // This uses host PATH before chroot, so might not be correct.
		if err == nil {
			fmt.Printf("%s Found path via host LookPath (may be incorrect): %s\n", logPrefix, potentialPath)
			// For now, let's stick to the user input, assuming it's relative to rootfs root or absolute
			// execPath = potentialPath
		} else {
			fmt.Printf("%s Warning: Could not find %s in host PATH, assuming relative to root or absolute path.\n", logPrefix, userCommand)
		}
		// If user gives "bin/sh", execPath remains "bin/sh". If "/bin/sh", it remains "/bin/sh".
	} else {
		fmt.Printf("%s Command is absolute path: %s\n", logPrefix, execPath)
	}


	fmt.Printf("%s Executing command: %s with args %v\n", logPrefix, execPath, userArgs)

	// Prepare environment variables (inherit from parent for simplicity now)
	env := os.Environ()

	// Use syscall.Exec to replace the current process
	// Args[0] should be the command path itself.
	err := syscall.Exec(execPath, userArgs, env)
	if err != nil {
		// If Exec fails, the current process continues.
		return fmt.Errorf("%s failed to exec command '%s': %w", logPrefix, execPath, err)
	}

	// Unreachable code if Exec succeeds
	return nil
} 