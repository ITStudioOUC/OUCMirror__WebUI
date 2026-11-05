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

## 生成软件源配置

先备份系统现有的软件源配置。Ubuntu 24.04 及之后通常使用 `/etc/apt/sources.list.d/ubuntu.sources`，Ubuntu 22.04 通常使用 `/etc/apt/sources.list`。

在下面选择当前系统的版本和配置格式，将生成的内容替换到对应文件中。配置中的域名随当前访问的镜像站地址变化，右上角按钮可复制完整配置。

```mirror-config
{"type":"ubuntu"}
```

安全更新默认保留官方源，以减少镜像同步延迟的影响。`proposed` 默认关闭，普通使用无需开启。源码开关控制是否获取源码包索引。

保留其他独立软件源的配置。传统格式和 DEB822 格式选择一种即可，不要重复启用相同的 Ubuntu 仓库。选择 HTTP 或 HTTPS 时，请确认当前镜像站入口支持对应协议。

## 更新索引

```sh
sudo apt update
apt-cache policy
```

检查输出中的仓库地址是否指向本站。若出现缺少 Release 文件、404 或签名错误，先恢复备份并检查系统代号、同步状态及系统时间，不要关闭签名校验。

## 参考资料

- [TUNA Ubuntu 使用帮助](https://mirrors.tuna.tsinghua.edu.cn/help/ubuntu/)
- [Ubuntu 软件仓库说明](https://documentation.ubuntu.com/server/how-to/software/package-management/)
