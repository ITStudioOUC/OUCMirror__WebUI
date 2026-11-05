# 镜像使用帮助

本目录保存补充帮助文章的 Markdown 原稿，配置示例使用本站公开地址 `https://mirrors.app.itstudio.club`。部署到其他域名时，需要替换示例中的站点地址，并确认对应仓库已经同步。

发布时，在管理后台的“镜像与任务”中选择同名镜像，将全文粘贴到“展示信息 → 使用说明（Markdown）”并保存。文章显示在 `/help/<源名>/`，所有界面语言共用一份中文内容。

线上文章保存在后端数据库中；本目录不会在部署时自动导入或覆盖后台编辑的内容。已有 Debian、Ubuntu 安装镜像文章继续保留在后台。

| 文件 | 内容 |
| --- | --- |
| [anaconda.md](anaconda.md) | Conda 频道和安装器 |
| [archlinux.md](archlinux.md) | Arch Linux 官方仓库 |
| [archlinuxarm.md](archlinuxarm.md) | Arch Linux ARM 仓库 |
| [archlinuxcn.md](archlinuxcn.md) | Arch Linux 中文社区仓库 |
| [debian-cd.md](debian-cd.md) | Debian 安装镜像和校验 |
| [debian-security.md](debian-security.md) | Debian 安全更新仓库 |
| [elrepo.md](elrepo.md) | Enterprise Linux 驱动和内核仓库 |
| [jenkins.md](jenkins.md) | Jenkins 安装包和 WAR |
| [kubernetes.md](kubernetes.md) | Kubernetes DEB、RPM 仓库 |
| [openwrt.md](openwrt.md) | OpenWrt 正式版软件仓库 |
| [proxmox.md](proxmox.md) | Proxmox VE 软件仓库 |
| [tailscale.md](tailscale.md) | Tailscale Debian、Ubuntu 仓库 |
| [ubuntu.md](ubuntu.md) | Ubuntu 软件仓库 |

文章参考 TUNA 帮助和各项目官方文档编写，原始资料链接见各篇末尾。目录、版本和同步范围以本站实际提供的内容为准。

## 交互式配置

后台可通过“插入配置生成器”添加控件，也可以直接在 Markdown 中插入：

```mirror-config
{"type":"ubuntu"}
```

目前注册的类型为 `debian`、`ubuntu`、`debian-security`、`archlinux`、`archlinuxarm`、`archlinuxcn`、`anaconda`、`kubernetes`、`tailscale`、`proxmox`、`openwrt`、`elrepo`。

不指定 `format` 时，支持多格式的控件会显示格式选择。指定格式可固定展示，例如 `{"type":"ubuntu","format":"deb822","options":{"release":"noble"}}`。默认选项只接受对应控件已定义的值，不支持脚本或任意模板。

后台保存文章与控件标记；版本列表和默认版本在“配置生成器”中维护，保存后帮助页会自动读取，无需重新编译。`debian-security` 共用 Debian 中的稳定版选项。移除版本前，需检查文章中是否通过 `options.release` 固定引用了它；失效的配置标记会显示为普通代码供作者修正。

前端的 `src/mirror-config.ts` 注册生成规则，`MirrorConfig.vue` 和 `ConfigCode.vue` 共用布局和复制逻辑。新增软件源或更改包管理器规则仍需修改代码；`backend/generator-defaults.json` 仅用于首次初始化版本目录。
