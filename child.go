package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// 此文件将包含子进程（容器）初始化逻辑

// runChildProcess 在容器的初始命名空间内运行
func runChildProcess(args []string) {
	fmt.Printf("子进程: 启动初始化，PID: %d
", os.Getpid())

	// 解析参数: args = ["child-init", rootfs, mem, cpu, command, commandArgs...]
	if len(args) < 5 {
		log.Fatalf("子进程内部错误：参数不足，需要至少 5 个，得到 %d", len(args))
	}
	rootfsPath := args[1]
	memLimit := args[2]
	cpuLimit := args[3]
	command := args[4]
	commandArgs := args[4:] // syscall.Exec 需要命令本身作为第一个参数

	fmt.Printf("子进程: rootfs=%s, mem=%s, cpu=%s, cmd=%s, args=%v
", rootfsPath, memLimit, cpuLimit, command, commandArgs)

	// 1. 设置主机名 (UTS Namespace)
	fmt.Println("子进程: 设置主机名 -> mydocker-container")
	if err := syscall.Sethostname([]byte("mydocker-container")); err != nil {
		log.Fatalf("错误: 子进程设置主机名失败: %v", err)
	}

	// 2. 设置 Cgroups
	// 使用 PID 作为简单的容器 ID
	containerID := strconv.Itoa(os.Getpid())
	fmt.Printf("子进程: 准备设置 Cgroups (ID: %s)
", containerID)
	setupCgroups(containerID, memLimit, cpuLimit)
	fmt.Println("子进程: Cgroups 设置调用完成")

	// 3. 设置文件系统 (Mount Namespace & Chroot)
	fmt.Printf("子进程: 准备设置文件系统 (rootfs: %s)
", rootfsPath)
	setupFilesystem(rootfsPath)
	fmt.Println("子进程: 文件系统设置调用完成")

	// 4. 执行用户命令
	fmt.Printf("子进程: 准备执行命令: %v
", commandArgs)

	// 在 chroot 后的新环境中查找命令路径
	// 注意：这依赖于新 rootfs 中的 $PATH 环境变量以及命令是否存在
	commandPath, err := exec.LookPath(command) // command 是 args[4]
	if err != nil {
		// 如果 LookPath 失败，尝试直接使用 command (可能用户提供了绝对路径)
		log.Printf("警告: 在 $PATH 中查找命令 '%s' 失败: %v. 尝试直接执行.", command, err)
		commandPath = command
	}
	fmt.Printf("子进程: 解析到的命令路径: %s
", commandPath)

	// syscall.Exec 会用新进程替换当前 Go 进程
	// 第一个参数是可执行文件路径
	// 第二个参数是 argv (包含命令本身作为 argv[0])
	// 第三个参数是环境变量
	if err := syscall.Exec(commandPath, commandArgs, os.Environ()); err != nil {
		// 如果 Exec 调用本身失败 (例如权限问题，或路径确实无效)
		log.Fatalf("错误: 子进程执行命令 '%s' ('%s') 失败: %v", command, commandPath, err)
	}

	// 如果 Exec 成功，代码不会执行到这里
	log.Println("子进程: syscall.Exec 调用完成 (理论上不应到达)")
}

// cgroups 和 fs 的辅助函数声明 (实际在其他文件)
func setupCgroups(containerID string, memLimit string, cpuLimit string) { /* 实现见 cgroups.go */ }
func setupFilesystem(rootfsPath string) { /* 实现见 fs.go */ } 