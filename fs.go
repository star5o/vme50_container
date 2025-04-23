package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
)

// 此文件将包含文件系统隔离 (chroot, mount) 相关的辅助函数 

// setupFilesystem 配置容器的文件系统环境
func setupFilesystem(rootfsPath string) {
	fmt.Printf("文件系统: 配置 rootfs -> %s
", rootfsPath)

	// 1. 设置挂载传播为私有 (MS_PRIVATE)
	// 这是为了防止容器内的挂载事件泄露到主机，反之亦然。
	// MS_REC 确保递归应用到所有子挂载点。
	fmt.Println("文件系统: 设置根挂载点传播为 MS_PRIVATE | MS_REC")
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		log.Fatalf("错误: 设置根挂载点为 MS_PRIVATE 失败: %v", err)
	}

	// 2. 执行 Chroot
	fmt.Printf("文件系统: Chroot 到 %s
", rootfsPath)
	if err := syscall.Chroot(rootfsPath); err != nil {
		log.Fatalf("错误: Chroot 到 %s 失败: %v", rootfsPath, err)
	}
	fmt.Println("文件系统: Chdir 到 / (新的根目录)")
	if err := syscall.Chdir("/"); err != nil {
		log.Fatalf("错误: Chdir 到 / 失败: %v", err)
	}

	// 3. 挂载虚拟文件系统
	fmt.Println("文件系统: 挂载 /proc")
	// 创建 /proc 目录（如果不存在）
	if err := os.MkdirAll("/proc", 0755); err != nil && !os.IsExist(err) {
		log.Fatalf("错误: 在 chroot 环境中创建 /proc 目录失败: %v", err)
	}
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		log.Fatalf("错误: 挂载 /proc 失败: %v", err)
	}

	fmt.Println("文件系统: 挂载 /dev/pts")
	// 创建 /dev/pts 目录（如果不存在）
	if err := os.MkdirAll("/dev/pts", 0755); err != nil && !os.IsExist(err) {
		log.Fatalf("错误: 在 chroot 环境中创建 /dev/pts 目录失败: %v", err)
	}
	if err := syscall.Mount("devpts", "/dev/pts", "devpts", 0, ""); err != nil {
		log.Fatalf("错误: 挂载 /dev/pts 失败: %v", err)
	}

	fmt.Println("文件系统: 配置完成")
}

// 后续将添加 mountProc 和 mountDevPts 函数 