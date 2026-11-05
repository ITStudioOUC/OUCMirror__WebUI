本站提供 Arch Linux 官方软件仓库。本文适用于 Arch Linux 的 x86_64 系统；ARM 设备请参阅 [Arch Linux ARM](/help/archlinuxarm/)，中文社区软件包请参阅 [Arch Linux CN](/help/archlinuxcn/)。

## 配置镜像

先备份镜像列表：

```sh
sudo cp /etc/pacman.d/mirrorlist /etc/pacman.d/mirrorlist.bak
```

编辑 `/etc/pacman.d/mirrorlist`，将下面一行放在最前面：

```text
Server = https://mirrors.app.itstudio.club/mirrors/archlinux/$repo/os/$arch
```

`$repo` 和 `$arch` 是 pacman 使用的变量，请原样保留。`/etc/pacman.conf` 中相应仓库应继续通过 `Include = /etc/pacman.d/mirrorlist` 引用镜像列表。

## 刷新并完整升级

更换镜像后执行：

```sh
sudo pacman -Syyu
```

之后日常升级使用 `sudo pacman -Syu`。Arch Linux 不支持部分升级，不要仅运行 `pacman -Sy` 后单独更新某个普通软件包。

若遇到数据库或软件包 404，请检查 [本站同步状态](/status)并稍后重试。持续失败时可恢复备份，或把其他可用镜像移到前面；保持软件包签名校验开启。

## 安装镜像

安装 ISO 位于 [archlinux/iso/](/mirrors/archlinux/iso/)。选择发布目录中的 ISO 和签名文件，按官方安装指南验证签名。此仓库不提供用于任意历史版本回退的 Arch Linux Archive 服务。

## 参考资料

- [TUNA Arch Linux 使用帮助](https://mirrors.tuna.tsinghua.edu.cn/help/archlinux/)
- [ArchWiki：镜像](https://wiki.archlinux.org/title/Mirrors)
- [ArchWiki：系统维护](https://wiki.archlinux.org/title/System_maintenance)
