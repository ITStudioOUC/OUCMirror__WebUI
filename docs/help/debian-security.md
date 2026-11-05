`debian-security` 提供 Debian 安全更新。它需要与 [Debian 主仓库](/help/debian/)配合使用，不能代替主仓库。

安全更新对时效要求较高，通常建议保留官方 `https://security.debian.org/debian-security`。需要使用本站时，请先检查 [同步状态](/status)，再按下面的方法替换安全更新条目。

## 选择发行版

运行 `cat /etc/os-release` 确认当前系统代号。常见对应关系：

| Debian 版本 | 代号 | 安全更新套件 |
| --- | --- | --- |
| Debian 13 | trixie | trixie-security |
| Debian 12 | bookworm | bookworm-security |
| Debian 11 | bullseye | bullseye-security |

目录存在不代表该版本仍处于官方安全支持期，请同时查阅 Debian 的安全支持和 LTS 公告。

## 生成安全更新配置

备份 `/etc/apt/sources.list` 或 `/etc/apt/sources.list.d/debian.sources`，在下面选择当前系统版本和格式，只替换原有的安全更新段。保留主仓库、updates 等其他配置。

```mirror-config
{"type":"debian-security"}
```

组件会按发行版调整：Debian 11 不加入 `non-free-firmware`。传统格式和 DEB822 选择一种即可，不要重复添加安全更新源。这里生成的内容会使用本站安全更新镜像；希望保留官方安全更新时，请使用 [Debian 帮助页](/help/debian/)中的完整配置生成器。

## 更新索引

```sh
sudo apt update
apt-cache policy
```

若本站同步延迟、索引缺失或更新失败，将安全更新 URI 改回 `https://security.debian.org/debian-security` 后重试。不要关闭 APT 签名校验。

## 参考资料

- [TUNA Debian Security 使用帮助](https://mirrors.tuna.tsinghua.edu.cn/help/debian-security/)
- [Debian 安全信息](https://www.debian.org/security/)
- [Debian LTS](https://wiki.debian.org/LTS)
