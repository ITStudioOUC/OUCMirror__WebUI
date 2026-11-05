本站提供 Ubuntu 的 x86 软件仓库。ARM 等架构使用的 `ubuntu-ports` 不在此仓库内；安装 ISO 请前往 [Ubuntu 安装镜像](/help/ubuntu-releases/)。

## 选择系统版本

运行下面的命令，确认当前系统代号和架构：

```sh
. /etc/os-release
printf '%s\n' "$VERSION_CODENAME"
dpkg --print-architecture
```

本文以 Ubuntu 24.04 LTS（`noble`）为例。Ubuntu 22.04 LTS 使用 `jammy`。只替换为当前系统的代号，换源不能代替发行版升级。

切换前检查 [noble 索引目录](/mirrors/ubuntu/dists/noble/)是否有 `InRelease` 或 `Release`。如果索引尚未同步完成，请继续使用官方源，稍后再切换。

## Ubuntu 24.04 及之后：DEB822 格式

先备份 `/etc/apt/sources.list.d/ubuntu.sources`，再将其中的 Ubuntu 仓库配置替换为：

```text
Types: deb
URIs: https://mirrors.app.itstudio.club/mirrors/ubuntu/
Suites: noble noble-updates noble-backports
Components: main restricted universe multiverse
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg

Types: deb
URIs: http://security.ubuntu.com/ubuntu/
Suites: noble-security
Components: main restricted universe multiverse
Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg
```

安全更新保留官方源，以减少镜像同步延迟的影响。不要同时保留另一份相同仓库配置。

## Ubuntu 22.04：传统格式

备份 `/etc/apt/sources.list` 后，将 Ubuntu 条目替换为：

```text
deb https://mirrors.app.itstudio.club/mirrors/ubuntu/ jammy main restricted universe multiverse
deb https://mirrors.app.itstudio.club/mirrors/ubuntu/ jammy-updates main restricted universe multiverse
deb https://mirrors.app.itstudio.club/mirrors/ubuntu/ jammy-backports main restricted universe multiverse
deb http://security.ubuntu.com/ubuntu/ jammy-security main restricted universe multiverse
```

保留其他独立软件源的配置。需要源码包时，可以在传统格式中增加对应 `deb-src` 行，或将 DEB822 的 `Types` 改为 `deb deb-src`。

## 更新索引

```sh
sudo apt update
apt-cache policy
```

检查输出中的仓库地址是否指向本站。若出现缺少 Release 文件、404 或签名错误，先恢复备份并检查系统代号、同步状态及系统时间，不要关闭签名校验。

## 参考资料

- [TUNA Ubuntu 使用帮助](https://mirrors.tuna.tsinghua.edu.cn/help/ubuntu/)
- [Ubuntu 软件仓库说明](https://documentation.ubuntu.com/server/how-to/software/package-management/)
