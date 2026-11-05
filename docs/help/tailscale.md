本站提供 Tailscale 稳定通道的 Debian、Ubuntu APT 软件包，分别位于 `tailscale/debian/` 和 `tailscale/ubuntu/`。镜像地址中不含上游的 `stable/` 路径；本站未提供对应的 RPM 仓库。

## 确认系统

运行 `cat /etc/os-release` 确认发行版和代号。不能在 Ubuntu 上选择 Debian 仓库，反之亦然。

安装工具：

```sh
sudo apt update
sudo apt install ca-certificates curl
```

已有 `/etc/apt/sources.list.d/tailscale.list`、`tailscale.sources` 或同名密钥环时，先备份并检查，避免重复配置。

## 生成配置

选择当前系统版本。第一段命令从 Tailscale 官方获取对应公钥，第二段是需要保存的源文件内容，两者会一起随版本变化。传统格式和 DEB822 选择一种即可。

```mirror-config
{"type":"tailscale"}
```

未列出的版本请先检查 [Debian 目录](/mirrors/tailscale/debian/dists/)或 [Ubuntu 目录](/mirrors/tailscale/ubuntu/dists/)是否有对应套件，并参照官方说明配置。

## 安装和登录

```sh
sudo apt update
sudo apt install tailscale
sudo systemctl enable --now tailscaled
sudo tailscale up
tailscale status
```

根据 `tailscale up` 输出的登录地址完成设备授权。镜像仅负责安装包下载，账号登录、控制服务和组网仍由 Tailscale 提供。

## 参考资料

- [Tailscale 稳定版软件仓库](https://pkgs.tailscale.com/stable/)
- [Tailscale Linux 安装说明](https://tailscale.com/kb/1031/install-linux)
