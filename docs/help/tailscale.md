本站提供 Tailscale 稳定通道的 Debian、Ubuntu APT 软件包，分别位于 `tailscale/debian/` 和 `tailscale/ubuntu/`。镜像地址中不含上游的 `stable/` 路径；本站未提供对应的 RPM 仓库。

先运行 `cat /etc/os-release` 确认发行版和代号。下面两套配置择一使用，不能在 Ubuntu 上使用 Debian 仓库，反之亦然。

## 准备工具

```sh
sudo apt update
sudo apt install ca-certificates curl
sudo install -d -m 0755 /usr/share/keyrings
```

已有 `/etc/apt/sources.list.d/tailscale.list` 或同名密钥环时，先备份并检查，避免重复配置。密钥从 Tailscale 官方站点获取。

## Debian 示例

以 Debian 12（`bookworm`）为例，下载官方公钥：

```sh
curl -fsSL https://pkgs.tailscale.com/stable/debian/bookworm.noarmor.gpg | sudo tee /usr/share/keyrings/tailscale-archive-keyring.gpg >/dev/null
sudo chmod 0644 /usr/share/keyrings/tailscale-archive-keyring.gpg
```

将 `/etc/apt/sources.list.d/tailscale.list` 设置为：

```text
deb [signed-by=/usr/share/keyrings/tailscale-archive-keyring.gpg] https://mirrors.app.itstudio.club/mirrors/tailscale/debian bookworm main
```

Debian 13 使用 `trixie`，同时替换官方密钥下载地址和源配置中的代号。其他版本请先确认 [Debian 目录](/mirrors/tailscale/debian/dists/)中存在对应套件。

## Ubuntu 示例

以 Ubuntu 24.04 LTS（`noble`）为例：

```sh
curl -fsSL https://pkgs.tailscale.com/stable/ubuntu/noble.noarmor.gpg | sudo tee /usr/share/keyrings/tailscale-archive-keyring.gpg >/dev/null
sudo chmod 0644 /usr/share/keyrings/tailscale-archive-keyring.gpg
```

将 `/etc/apt/sources.list.d/tailscale.list` 设置为：

```text
deb [signed-by=/usr/share/keyrings/tailscale-archive-keyring.gpg] https://mirrors.app.itstudio.club/mirrors/tailscale/ubuntu noble main
```

其他 Ubuntu 版本需同时修改密钥地址和套件代号，并确认 [Ubuntu 目录](/mirrors/tailscale/ubuntu/dists/)中已提供该套件。

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
