本站提供 Proxmox 软件仓库与部分安装镜像。本文说明 Proxmox VE 的非订阅仓库配置，Debian 基础软件源请另行参阅 [Debian 使用帮助](/help/debian/)。

## 确认系统版本

```sh
pveversion
cat /etc/os-release
```

Proxmox VE 9 基于 Debian 13（`trixie`），Proxmox VE 8 基于 Debian 12（`bookworm`）。仓库套件必须与当前系统一致，不要通过修改代号直接跨版本升级。

以下操作在节点的 root 终端中执行。操作前备份 `/etc/apt/sources.list` 及 `/etc/apt/sources.list.d/` 中将要修改的文件。

## 生成仓库配置

选择当前 Proxmox VE 版本和源配置格式，将生成的内容保存到所标注的文件中。PVE 9 通常使用 DEB822，PVE 8 通常使用传统格式。

```mirror-config
{"type":"proxmox"}
```

Proxmox 官方安装的软件包负责提供签名密钥；PVE 9 使用 `/usr/share/keyrings/proxmox-archive-keyring.gpg`，PVE 8 的官方密钥文件为 `/etc/apt/trusted.gpg.d/proxmox-release-bookworm.gpg`。文件缺失时先按对应版本的官方说明修复，不要移除签名校验。传统格式继续使用系统已有的官方签名密钥配置。

两种格式选择其一，不要重复启用相同仓库。

## 检查仓库并更新

在 Proxmox 网页端的节点“更新 → 软件仓库”中检查配置。没有订阅时，可禁用仍然启用的 enterprise 条目，避免无权限的仓库导致更新报错；有订阅的用户可继续使用官方 enterprise 仓库。

`pve-no-subscription` 面向无需订阅的用户，其验证程度与 enterprise 仓库不同，生产环境请按 Proxmox 官方建议选择。

```sh
apt update
apt-cache policy pve-manager
```

软件源切换成功后，再按自己的维护计划更新系统。Ceph 等独立仓库需要匹配相应版本，不要直接套用 PVE 仓库的路径和组件名。

## 安装镜像

ISO 可从 [Proxmox 安装镜像目录](/mirrors/proxmox/iso/)下载。安装前请核对官方公布的校验值和硬件要求。

## 参考资料

- [TUNA Proxmox 使用帮助](https://mirrors.tuna.tsinghua.edu.cn/help/proxmox/)
- [Proxmox VE 软件仓库说明](https://pve.proxmox.com/pve-docs/pve-admin-guide.html#sysadmin_package_repositories)
