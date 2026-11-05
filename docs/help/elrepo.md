ELRepo 为 Enterprise Linux 系列系统提供额外的驱动、内核等软件包。使用前确认系统的主版本和架构：

```sh
cat /etc/os-release
uname -m
```

ELRepo 仓库版本必须与系统主版本匹配，不能将 EL9 软件包用于 EL10，也不要将其配置到 Debian、Ubuntu 等系统中。

## 安装官方仓库配置

先按 [ELRepo 官方说明](https://elrepo.org/)安装与当前系统匹配的 `elrepo-release` 软件包，并导入官方签名密钥。已经配置过 ELRepo 的系统可以跳过这一步。

备份仓库文件：

```sh
sudo cp /etc/yum.repos.d/elrepo.repo /etc/yum.repos.d/elrepo.repo.bak
```

## 替换下载地址

编辑 `/etc/yum.repos.d/elrepo.repo`：

1. 注释需要使用本站的仓库段中的 `mirrorlist=` 行。
2. 启用对应的 `baseurl=`，将其中 `http://elrepo.org/linux` 或 `https://elrepo.org/linux` 替换为 `https://mirrors.app.itstudio.club/mirrors/elrepo`。
3. 保留路径后面的系统版本、仓库名称及 `$basearch` 变量；保留官方配置中的签名检查和密钥地址。

也可以使用下方命令完成备份、启用直接下载地址、替换地址及刷新缓存。命令保留仓库配置中的系统版本、架构和签名校验设置。

```mirror-config
{"type":"elrepo"}
```

## 刷新缓存

```sh
sudo dnf makecache
dnf repolist
```

`elrepo-kernel` 等可选仓库保持原有启用状态；需要额外内核时，应先阅读 ELRepo 的内核说明，再明确启用对应仓库。换镜像本身不需要更换内核。

如果元数据下载失败，检查 [ELRepo 目录](/mirrors/elrepo/)是否包含当前系统版本，并恢复备份后重试。

## 参考资料

- [TUNA ELRepo 使用帮助](https://mirrors.tuna.tsinghua.edu.cn/help/elrepo/)
- [ELRepo 项目与安装说明](https://elrepo.org/)
