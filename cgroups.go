package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// 此文件将包含 Cgroup v2 相关操作辅助函数

const (
	cgroupRoot = "/sys/fs/cgroup"
)

// setupCgroups 配置 cgroup v2
// containerID 用于创建唯一的 cgroup 目录
func setupCgroups(containerID string, memLimit string, cpuLimit string) {
	fmt.Printf("Cgroups: 配置 Cgroup (ID: %s, Mem: %s, CPU: %s)
", containerID, memLimit, cpuLimit)

	// 1. 构造 Cgroup 路径 (使用 "mydocker" 作为父目录)
	cgroupPath := filepath.Join(cgroupRoot, "mydocker", containerID)
	fmt.Printf("Cgroups: 路径为 %s
", cgroupPath)

	// 2. 创建 Cgroup 目录
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		log.Fatalf("错误: 创建 cgroup 目录 %s 失败: %v", cgroupPath, err)
	}
	// 注意：理想情况下，容器退出时应删除此目录 (os.RemoveAll)
	// 但为了简单起见，教学版省略清理步骤。

	// 3. 将当前进程 PID 加入 Cgroup
	pid := os.Getpid()
	procsFile := filepath.Join(cgroupPath, "cgroup.procs")
	fmt.Printf("Cgroups: 将 PID %d 添加到 %s
", pid, procsFile)
	if err := os.WriteFile(procsFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		// 注意：这里可能会因为权限问题或 cgroup 配置问题失败
		// 例如，如果父 cgroup 没有启用某些 controller，子 cgroup 可能无法写入。
		// 在 systemd 管理的系统上，直接操作 /sys/fs/cgroup 可能有限制。
		log.Fatalf("错误: 写入 PID %d 到 %s 失败: %v", pid, procsFile, err)
	}

	// 4. 设置资源限制
	if memLimit != "" {
		setMemoryLimit(cgroupPath, memLimit)
	} else {
		fmt.Println("Cgroups: 未提供内存限制，跳过设置")
	}

	// 设置 CPU 限制
	if cpuLimit != "" {
		setCPULimit(cgroupPath, cpuLimit)
	} else {
		fmt.Println("Cgroups: 未提供 CPU 限制，跳过设置")
	}

	fmt.Println("Cgroups: 配置完成")
}

// setMemoryLimit 设置 Cgroup v2 内存限制
func setMemoryLimit(cgroupPath string, memLimit string) {
	memFile := filepath.Join(cgroupPath, "memory.max")
	fmt.Printf("Cgroups: 设置内存限制 %s 到 %s
", memLimit, memFile)

	// TODO: 更健壮的解析，支持 K, M, G 单位，这里简化处理，假定是纯数字字节或带 M/G 后缀
	var limitInBytes string
	if strings.HasSuffix(memLimit, "M") || strings.HasSuffix(memLimit, "m") {
		val, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimSuffix(memLimit, "M"), "m"), 10, 64)
		if err != nil {
			log.Printf("警告: 解析内存限制 '%s' 失败: %v, 跳过设置", memLimit, err)
			return
		}
		limitInBytes = strconv.FormatInt(val*1024*1024, 10)
	} else if strings.HasSuffix(memLimit, "G") || strings.HasSuffix(memLimit, "g") {
		val, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimSuffix(memLimit, "G"), "g"), 10, 64)
		if err != nil {
			log.Printf("警告: 解析内存限制 '%s' 失败: %v, 跳过设置", memLimit, err)
			return
		}
		limitInBytes = strconv.FormatInt(val*1024*1024*1024, 10)
	} else {
		// 假定是纯数字字节
		_, err := strconv.ParseInt(memLimit, 10, 64)
		if err != nil {
			log.Printf("警告: 内存限制 '%s' 格式无法识别 (应为纯数字字节, 或带 M/G 后缀): %v, 跳过设置", memLimit, err)
			return
		}
		limitInBytes = memLimit
	}

	if err := os.WriteFile(memFile, []byte(limitInBytes), 0644); err != nil {
		// 写 memory.max 失败可能是因为 memory controller 未在父 cgroup 启用
		log.Printf("警告: 写入内存限制到 %s 失败: %v. 请检查 cgroup memory controller 是否启用", memFile, err)
		// 不中止，允许继续执行，只是内存限制可能无效
	}
}

// setCPULimit 设置 Cgroup v2 CPU 限制
func setCPULimit(cgroupPath string, cpuLimit string) {
	cpuFile := filepath.Join(cgroupPath, "cpu.max")
	fmt.Printf("Cgroups: 设置 CPU 限制 %s 到 %s
", cpuLimit, cpuFile)

	// 解析 cpuLimit (例如 "0.5" 表示 50%)
	cpuQuotaFloat, err := strconv.ParseFloat(cpuLimit, 64)
	if err != nil || cpuQuotaFloat <= 0 {
		log.Printf("警告: 解析 CPU 限制 '%s' 失败 (应为 > 0 的小数): %v, 跳过设置", cpuLimit, err)
		return
	}

	// Cgroup v2 cpu.max 格式为 "quota period"
	// period 通常是 100000 微秒 (100ms)
	period := 100000
	quota := int(cpuQuotaFloat * float64(period))

	if quota <= 0 {
		log.Printf("警告: 计算得到的 CPU quota (%d) 无效, 来自限制 '%s', 跳过设置", quota, cpuLimit)
		return
	}

	cpuMax := fmt.Sprintf("%d %d", quota, period)
	fmt.Printf("Cgroups: 写入 cpu.max 内容: '%s'
", cpuMax)

	if err := os.WriteFile(cpuFile, []byte(cpuMax), 0644); err != nil {
		// 写 cpu.max 失败可能是因为 cpu controller 未在父 cgroup 启用
		log.Printf("警告: 写入 CPU 限制到 %s 失败: %v. 请检查 cgroup cpu controller 是否启用", cpuFile, err)
		// 不中止
	}
}

// 后续将添加 setCPULimit 函数 