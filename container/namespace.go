package container

import "syscall"

// GetCloneFlags returns the syscall clone flags for creating basic container namespaces.
// Currently includes PID, UTS, IPC, and Mount namespaces.
func GetCloneFlags() uintptr {
	return syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNS
} 