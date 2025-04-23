# VME50 Container - 一个教学目的的简易 Docker

本项目使用 Go 语言编写，旨在演示容器化技术（如 Linux 命名空间、Cgroups 和 Chroot）的基本原理。这是一个简化的实现，主要用于学习和理解容器是如何工作的。

**仓库地址**: [https://github.com/star5o/vme50_container](https://github.com/star5o/vme50_container)

## 功能特性

*   使用 Linux 命名空间 (PID, UTS, IPC, Mount) 提供进程隔离。
*   使用 Cgroups v2 限制容器的资源使用（CPU 权重、内存上限）。
*   使用 `chroot` 提供基础的文件系统隔离。
*   提供简单的命令行接口 (`run` 命令) 来启动容器。

## 环境要求

*   **操作系统**: Linux (已在 Ubuntu 24.04 测试，理论上支持 Cgroups v2 和必要命名空间的较新内核均可)。
*   **Go 环境**: 需要安装 Go 语言开发环境 (建议 1.18 或更高版本)。如果您尚未安装，请访问 [Go 官方网站](https://golang.org/doc/install) 查看安装指南。
    *   通过包管理器安装（如 `sudo apt install golang-go`）通常也是可以的。建议通过运行 `go version` 检查安装的版本是否满足要求。
*   **Root 权限**: 本程序需要以 `root` 用户身份运行，因为它需要执行创建命名空间、配置 Cgroups 和 `chroot` 等特权操作。

## 编译指南

假设您已经克隆了本仓库，并且位于项目根目录 (`~/software/vme50_container`) 下。

1.  **初始化 Go Module (如果尚未完成)**:
    如果项目还没有 `go.mod` 文件，您可能需要初始化它（通常仓库里应该包含了）。如果需要手动初始化，可以在项目根目录运行：
    ```bash
    # 确认模块路径与 go.mod 文件中的一致 (例如 github.com/star5o/vme50_container)
    # 如果 go.mod 文件已存在，通常只需要运行 go mod tidy
    # go mod init github.com/star5o/vme50_container # 仅在没有 go.mod 时运行
    go mod tidy # 下载或更新依赖 (如 cobra, uuid)
    ```
    *注意*: 即使通过 `apt` 安装，Go 1.11 及以后版本默认启用 Go Modules 支持。推荐使用此模式。

2.  **编译程序**:
    在项目根目录下运行以下命令进行编译：
    ```bash
    # -o 指定输出的可执行文件名
    go build -o vme50-container .
    ```
    编译成功后，您会在当前目录 (`~/software/vme50_container`) 下看到一个名为 `vme50-container` 的可执行文件。

## 准备容器根文件系统 (Rootfs)

为了运行容器，您需要一个包含基本 Linux 环境的文件系统目录。一个简单的方法是使用 `busybox`。

1.  **使用 Docker 导出 Busybox (推荐)**:
    如果您安装了 Docker，这是最简单的方式：
    ```bash
    # 拉取 busybox 镜像
    docker pull busybox:latest

    # 创建一个 busybox 容器但不启动
    docker create --name temp-busybox busybox:latest

    # 从容器中导出文件系统到 ~/rootfs 目录
    # 确保 ~/rootfs 目录事先不存在或为空
    mkdir ~/rootfs
    docker export temp-busybox | tar -C ~/rootfs -xf -

    # 删除临时容器
    docker rm temp-busybox

    # 检查导出的文件系统
    ls ~/rootfs
    ```
    现在 `~/rootfs` 目录就包含了运行基本命令所需的文件。

2.  **手动准备 (备选)**:
    您也可以手动下载 Busybox 二进制文件，并创建一个包含必要目录（如 `/bin`, `/proc`, `/dev`, `/etc`, `/tmp` 等）的目录结构。确保将 Busybox 可执行文件放在 `/bin` 下，并创建指向它的常用命令符号链接（如 `sh`, `ls`, `echo` 等）。`/proc` 目录必须存在，容器启动时会挂载它。

## 运行容器

**必须使用 `sudo` 或以 `root` 用户身份运行！**

假设：
*   您位于项目根目录 `~/software/vme50_container`。
*   `vme50-container` 可执行文件在此目录中。
*   您的根文件系统位于 `~/rootfs`。

**基本运行示例 (启动一个 shell)**:

```bash
sudo ./vme50-container run --rootfs ~/rootfs /bin/sh
```

这将在一个新的容器环境中启动一个交互式 shell。您可以在这个 shell 中运行 Busybox 提供的命令（如 `ls /`, `pwd`, `echo hello`, `ps aux`）。输入 `exit` 退出容器。

**带资源限制和主机名运行**:

```bash
# 限制内存为 128MB，CPU 权重为 500，设置主机名为 my-container
sudo ./vme50-container run \
  --rootfs ~/rootfs \
  --memory-limit "128M" \
  --cpu-shares 500 \
  --hostname my-container \
  /bin/sh -c "echo 'Inside container!' ; hostname ; echo 'My PID is $$' ; ps aux ; sleep 10"
```

**命令行参数说明**:

*   `run`: 子命令，表示要运行一个新容器。
*   `--rootfs <path>`: **必需**。指定容器使用的根文件系统目录路径。
*   `--memory-limit <limit>`: (可选) 设置容器的内存上限 (e.g., "64M", "1G")。
*   `--cpu-shares <value>`: (可选) 设置容器的 CPU 权重 (Cgroup v2 `cpu.weight`，范围 1-10000，相对于其他容器)。
*   `--hostname <name>`: (可选) 设置容器的主机名。
*   `[command] [args...]`: **必需**。在容器内要执行的命令及其参数。**重要提示**: 当前实现要求命令使用绝对路径 (如 `/bin/sh`) 或相对于根文件系统根目录的路径 (如 `bin/sh`)。

## 重要提示与限制

*   **必须以 Root 身份运行**: 这是 Linux 内核功能的要求。
*   **Cgroups v2**: 当前实现主要针对 Cgroups v2。如果您的系统使用 Cgroups v1，Cgroup 相关功能可能无法正常工作。
*   **网络隔离**: 默认情况下，容器共享主机的网络栈。尚未实现网络命名空间隔离。
*   **用户隔离**: 容器内的进程以 root 用户运行（除非基础镜像配置了非 root 用户）。尚未实现用户命名空间隔离。
*   **文件系统**: 使用基础的 `chroot`。未实现 OverlayFS 等分层文件系统。
*   **命令路径**: 容器内执行命令时，目前没有完整的 PATH 查找，需要提供相对根目录或绝对路径。
*   **挂载点**: 只挂载了 `/proc`。`/dev`, `/sys` 等可能需要手动挂载（未来可改进）。

