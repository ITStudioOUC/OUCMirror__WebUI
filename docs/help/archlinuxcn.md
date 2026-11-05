Arch Linux CN 是 Arch Linux 中文社区维护的非官方软件仓库，收录部分常用软件、工具和字体。它不替代 Arch Linux 官方仓库，也不包含 AUR 的全部软件包。

## 添加仓库

备份 `/etc/pacman.conf`，在文件末尾添加：

```mirror-config
{"type":"archlinuxcn"}
```

`$arch` 保持原样。本站提供的架构可在 [仓库目录](/mirrors/archlinuxcn/)中查看。已有 `[archlinuxcn]` 配置时，修改原来的 `Server`，不要重复添加仓库段。

## 导入签名密钥并升级

按社区仓库的安装流程，先安装密钥环，随后立即完整升级系统：

```sh
sudo pacman -Sy archlinuxcn-keyring
sudo pacman -Syu
```

第一条命令用于引导安装社区签名密钥环，之后不要停留在仅刷新数据库的状态。日常更新使用 `sudo pacman -Syu`。

如果提示密钥不受信任或签名无效，请先确认系统时间，并根据中文社区的官方说明处理密钥环问题。不要通过 `SigLevel = Never` 绕过验证，也不要从不明来源导入密钥。

## 安装软件

完成上述步骤后，可以像使用其他 pacman 仓库一样安装软件：

```sh
pacman -Sl archlinuxcn
sudo pacman -S archlinuxcn/软件包名
```

将 `软件包名` 替换为列表中实际存在的名称。若要使用官方仓库的软件包镜像，请另行配置 [Arch Linux 镜像](/help/archlinux/)。

## 参考资料

- [TUNA Arch Linux CN 使用帮助](https://mirrors.tuna.tsinghua.edu.cn/help/archlinuxcn/)
- [Arch Linux 中文社区仓库说明](https://www.archlinuxcn.org/archlinux-cn-repo-and-mirror/)
