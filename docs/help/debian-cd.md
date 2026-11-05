此镜像提供 Debian 安装文件。已经安装 Debian、需要修改软件包源时，请参阅 [Debian 使用帮助](/help/debian/)。

## 选择安装文件

- [当前正式版](/mirrors/debian-cd/current/)：按 CPU 架构选择安装镜像。
- [amd64 网络安装镜像](/mirrors/debian-cd/current/amd64/iso-cd/)：适用于 Intel、AMD 的 64 位 x86 电脑；安装时需要联网下载所选软件。
- [当前 Live 镜像](/mirrors/debian-cd/current-live/)：可以先启动桌面环境进行体验，再按需安装。

文件名中的版本号以目录实际内容为准。`amd64` 不适用于 ARM 设备，请按设备架构选择对应目录。Live 镜像可用的架构和桌面环境也以目录为准。

## 下载与完整性校验

在所选 ISO 的同一目录中，同时下载 `SHA512SUMS` 和 `SHA512SUMS.sign`，与 ISO 放在同一个本地目录。

Linux 下运行：

```sh
sha512sum --ignore-missing -c SHA512SUMS
```

确认所下载的 ISO 显示 `OK`。Windows PowerShell 可计算文件哈希后，与清单中同名文件的值逐位比较：

```powershell
Get-FileHash -Algorithm SHA512 -LiteralPath .\下载的镜像.iso
```

将示例文件名替换为实际文件名。哈希一致说明文件与校验清单一致；清单本身的来源需要通过签名确认。

## 验证签名

按 [Debian 官方验证说明](https://www.debian.org/CD/verify)获取 Debian CD 发布密钥，并通过官方资料核对指纹。然后验证校验清单：

```sh
gpg --verify SHA512SUMS.sign SHA512SUMS
```

确认签名有效且签名密钥属于 Debian 发布团队，再进行 ISO 哈希校验。不要仅因为 GPG 显示某个签名有效，就信任尚未核对身份的密钥。

## 安装

校验后可将 ISO 挂载到虚拟机光驱，或用启动盘工具写入 USB 设备。实机安装会涉及磁盘分区，请先备份数据，并参考对应架构的安装手册。

## 参考资料

- [Debian 安装镜像](https://www.debian.org/CD/)
- [Debian 镜像验证](https://www.debian.org/CD/verify)
- [Debian 安装手册](https://www.debian.org/releases/stable/installmanual)
