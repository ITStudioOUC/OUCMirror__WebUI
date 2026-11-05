Arch Linux ARM 为 ARM 设备提供 Arch Linux 软件包。它与 x86_64 的 Arch Linux 仓库目录不同，请勿混用两者的镜像地址。

## 配置镜像

在已经安装好的 Arch Linux ARM 系统中，先备份：

```sh
sudo cp /etc/pacman.d/mirrorlist /etc/pacman.d/mirrorlist.bak
```

将下面一行放到 `/etc/pacman.d/mirrorlist` 的最前面：

```mirror-config
{"type":"archlinuxarm"}
```

保留 `$arch` 和 `$repo` 原样。这里没有 Arch Linux 官方仓库路径中的 `os/`。

本站目录包含 `aarch64`、`armv7h` 等架构，具体以 [仓库目录](/mirrors/archlinuxarm/)为准。不要为了使用某个目录而修改系统的架构设置。

## 更新系统

```sh
sudo pacman -Syyu
```

后续使用 `sudo pacman -Syu` 完整升级。遇到软件包缺失时，先检查同步状态和架构；不要关闭签名校验，也不要用 x86_64 仓库补充 ARM 软件包。

## 安装文件

根文件系统等安装文件位于 [archlinuxarm/os/](/mirrors/archlinuxarm/os/)。不同板卡的分区、引导程序和安装流程各不相同，请先在官方平台列表中找到自己的设备，再按对应指南选择文件。

## 参考资料

- [TUNA Arch Linux ARM 使用帮助](https://mirrors.tuna.tsinghua.edu.cn/help/archlinuxarm/)
- [Arch Linux ARM 镜像说明](https://archlinuxarm.org/about/mirrors)
- [Arch Linux ARM 支持的平台](https://archlinuxarm.org/platforms)
