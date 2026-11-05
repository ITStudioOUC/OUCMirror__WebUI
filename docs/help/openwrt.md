本站提供 OpenWrt 正式发行版的固件和软件包，目录位于 [openwrt/releases/](/mirrors/openwrt/releases/)，不包含 `snapshots`。

## 确认版本和包管理器

登录路由器后，先查看 `/etc/openwrt_release` 和现有的软件源配置。若配置中含有 `snapshots`，请保留原地址，本文的正式版镜像不能替代它。

OpenWrt 25.12 起使用 APK；24.10 及更早版本使用 OPKG。第三方固件可能更改包管理器或仓库，应以实际系统为准。下面的命令在路由器的 root 会话中执行。

## 生成换源命令

根据当前系统选择 APK 或 OPKG。下面的命令先备份配置，再替换仓库地址并刷新索引。

```mirror-config
{"type":"openwrt"}
```

两种方式仅替换官方地址的前缀，保留原来的版本、target、subtarget 和软件包架构路径。自定义仓库保持原样。不要复制其他路由器的整份源配置，否则可能安装不兼容的软件包。

## 更新失败时

先检查系统时间、HTTPS 证书支持和本站同步状态。缺少证书组件时，使用原来的可用仓库补齐所需组件后再切换，不要关闭证书或软件包签名验证。

404 常见于版本或架构不匹配，也可能是该发行版尚未同步到本站。可以恢复对应 `.bak` 配置后重新更新索引。

`apk update`、`opkg update` 只刷新索引，不执行固件升级；不要把更换软件源当成跨版本升级，也不要为此批量升级系统核心软件包。

## 下载固件

先在 OpenWrt 官方设备页面或 Firmware Selector 中确定型号和硬件版本，再在本站同版本的 `targets/` 目录中下载。刷入前核对文件名、设备适配和官方校验值。

## 参考资料

- [TUNA OpenWrt 使用帮助](https://mirrors.tuna.tsinghua.edu.cn/help/openwrt/)
- [OpenWrt 软件包管理](https://openwrt.org/docs/guide-user/additional-software/start)
- [OpenWrt Firmware Selector](https://firmware-selector.openwrt.org/)
