# 一个简单的教学版 Docker

这是一个使用 Go 语言编写的、用于教学目的的简化版 Docker 实现。
它利用了 Linux 内核的以下特性来创建隔离环境：

*   **Namespaces**: PID, UTS, Mount 命名空间 (`syscall.CLONE_NEWPID`, `CLONE_NEWUTS`, `CLONE_NEWNS`)
*   **Cgroups v2**: 用于设置内存 (`memory.max`) 和 CPU (`cpu.max`) 限制。
*   **Isolated Filesystem**: 通过 `chroot` 和独立的 Mount Namespace 实现。

sudo apt update && sudo apt install golang-go -y



## 编译

```bash
go build .
```

这将生成一个名为 `mydocker` 的可执行文件。

## 准备 Rootfs (根文件系统)

容器需要一个根文件系统目录，其中包含它需要运行的命令和库。一个简单的方法是使用 Docker 导出 `busybox` 镜像的文件系统：

```bash
# 1. 创建一个临时的 busybox 容器
CID=$(docker create busybox)

# 2. 将容器的文件系统导出到一个 tar 文件
docker export $CID > busybox.tar

# 3. 创建一个目录并解压 tar 文件
mkdir rootfs
tar -xf busybox.tar -C rootfs

# 4. （可选）清理临时容器和 tar 文件
docker rm $CID
rm busybox.tar
```

现在，`./rootfs` 目录就可以用作 `mydocker` 的根文件系统了。

## 运行

由于需要创建命名空间和配置 Cgroups，程序必须以 **root** 权限运行。

```bash
# 语法:
sudo ./mydocker -rootfs <path_to_rootfs> [-mem <limit>] [-cpu <limit>] -cmd "<command_and_args>"

# 示例 1: 在容器中运行 /bin/sh
sudo ./mydocker -rootfs ./rootfs -cmd "/bin/sh"

# 示例 2: 查看容器内的主机名 (应为 mydocker-container)
sudo ./mydocker -rootfs ./rootfs -cmd "hostname"

# 示例 3: 查看容器内的进程列表 (PID 应该从 1 开始)
sudo ./mydocker -rootfs ./rootfs -cmd "ps aux"

# 示例 4: 设置内存限制为 100MB，并运行一个简单的 shell 命令
sudo ./mydocker -rootfs ./rootfs -mem 100M -cmd "echo hello from container; sleep 5"

# 示例 5: 设置 CPU 限制为 50%，并运行命令
# 注意: CPU 限制的效果可能不明显，除非运行 CPU 密集型任务
sudo ./mydocker -rootfs ./rootfs -cpu 0.5 -cmd "/bin/sh"
```

## 命令行参数

*   `-rootfs` (string, **必需**): 指向容器根文件系统目录的路径。
*   `-cmd` (string, **必需**): 要在容器内执行的命令及其参数，用引号括起来。
*   `-mem` (string, 可选): 内存限制。支持纯数字（字节）或带 `M`/`m` (兆字节), `G`/`g` (吉字节) 后缀。例如: `100M`, `1G`。
*   `-cpu` (string, 可选): CPU 限制。一个大于 0 的小数，表示 CPU 时间片的比例。例如 `0.5` 表示容器最多使用 50% 的 CPU 时间。

## 实现细节

*   `main.go`: 程序入口，解析命令行参数，创建父进程并使用 `Cloneflags` 启动子进程。
*   `child.go`: 子进程（容器）的初始化逻辑，在此进程内完成命名空间、Cgroup、文件系统的设置，最后使用 `syscall.Exec` 执行用户命令。
*   `cgroups.go`: 辅助函数，用于与 Cgroup v2 文件系统 (`/sys/fs/cgroup`) 交互以设置资源限制。
*   `fs.go`: 辅助函数，用于设置挂载点传播、执行 `chroot`、挂载虚拟文件系统 (`/proc`, `/dev/pts`)。 