package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func main() {
	// 检查是否是子进程的初始化调用
	if len(os.Args) > 1 && os.Args[1] == "child-init" {
		// 参数: child-init rootfs mem cpu cmd args...
		if len(os.Args) < 6 {
			log.Fatalf("子进程需要至少 6 个参数: child-init rootfs mem cpu cmd ...")
		}
		// 注意：这里的 args 包含了 child-init, rootfs, mem, cpu, cmd 及后续参数
		// runChildProcess 将需要解析这些
		runChildProcess(os.Args[1:])
		return // runChildProcess 内部会 exec 或退出
	}

	// 定义命令行参数
	rootfs := flag.String("rootfs", "", "容器根文件系统的路径 (必需)")
	mem := flag.String("mem", "", "内存限制 (例如 "100M", 可选)")
	cpu := flag.String("cpu", "", "CPU 限制 (例如 "0.5", 可选)")
	cmdStr := flag.String("cmd", "", "要在容器内执行的命令及其参数 (必需)")

	flag.Parse()

	// 验证参数
	if *rootfs == "" {
		log.Fatalf("错误: -rootfs 参数是必需的")
	}
	if *cmdStr == "" {
		log.Fatalf("错误: -cmd 参数是必需的")
	}

	// 验证 rootfs 路径存在且是目录
	info, err := os.Stat(*rootfs)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("错误: rootfs 路径 '%s' 不存在", *rootfs)
		}
		log.Fatalf("错误: 检查 rootfs 路径 '%s' 失败: %v", *rootfs, err)
	}
	if !info.IsDir() {
		log.Fatalf("错误: rootfs 路径 '%s' 不是一个目录", *rootfs)
	}

	// 解析命令字符串
	cmdParts := strings.Fields(*cmdStr)
	if len(cmdParts) == 0 {
		log.Fatalf("错误: -cmd 参数不能为空命令")
	}
	command := cmdParts[0]
	args := cmdParts[1:]

	// 准备传递给子进程的参数
	childArgs := []string{"child-init", *rootfs, *mem, *cpu, command}
	childArgs = append(childArgs, args...)

	// 准备 exec.Command
	// 使用 /proc/self/exe 来重新执行当前可执行文件，但传递 "child-init" 作为第一个参数
	cmd := exec.Command("/proc/self/exe", childArgs...)

	// 设置 Cloneflags 来创建新的命名空间
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS |   // New mount namespace
					syscall.CLONE_NEWPID |   // New PID namespace
					syscall.CLONE_NEWUTS, // New UTS namespace
		// 注意：没有 CLONE_NEWUSER (简化) 或 CLONE_NEWNET
	}

	// 连接标准输入输出错误流
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("父进程: 准备启动子进程执行命令: %s
", *cmdStr)

	// 运行子进程并等待
	if err := cmd.Run(); err != nil {
		log.Fatalf("错误: 运行容器进程失败: %v", err)
	}

	fmt.Println("父进程: 子进程执行完毕")
}

// runChildProcess 的声明（实际实现在 child.go）
// 这里只是为了让 main.go 能编译通过
func runChildProcess(args []string) {
	fmt.Println("错误: runChildProcess 应该在 child.go 中实现!")
	os.Exit(1)
} 