本站镜像 Kubernetes 的版本化软件包仓库，可用于安装 `kubelet`、`kubeadm` 和 `kubectl`。容器镜像、容器运行时及集群网络插件需要另行配置。

## 选择版本和软件包格式

先根据安装或升级计划确定 Kubernetes 次版本。已有集群需要遵守官方版本偏差策略；修改软件源不会自动完成集群升级。控件中的版本是本站已提供的仓库分支，不代表所有版本都仍处于官方支持期。

Debian、Ubuntu 选择 APT，RPM 系统选择 RPM。下面的公钥地址和软件源会一起随版本变化。

## 准备工具

APT 系统先安装工具：

```sh
sudo apt update
sudo apt install ca-certificates curl gpg
```

备份已有 Kubernetes 源文件和同名签名密钥文件，再按控件中的步骤配置。APT 显示的第一段命令导入官方公钥，第二段是需要保存的源文件内容；RPM 配置通过 `gpgkey` 指定对应的官方公钥。

```mirror-config
{"type":"kubernetes"}
```

## 刷新索引并安装

APT 系统查看可安装版本：

```sh
sudo apt update
apt-cache madison kubeadm
```

新节点需要安装组件时：

```sh
sudo apt install kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
```

RPM 系统查看可安装版本：

```sh
sudo dnf makecache
dnf --disableexcludes=kubernetes list --showduplicates kubeadm
```

新节点安装使用 `sudo dnf install kubelet kubeadm kubectl --disableexcludes=kubernetes`。配置中的 `exclude` 避免普通系统更新意外升级集群组件。已有集群应按升级指南逐个处理节点。

## 检查地址

本站使用 `core:/stable:/v1.xx/` 目录布局，不使用旧的 `apt.kubernetes.io` 地址。其他版本请查看 [仓库目录](/mirrors/kubernetes/core:/stable:/)，并确认对应分支已完成同步。

## 参考资料

- [TUNA Kubernetes 使用帮助](https://mirrors.tuna.tsinghua.edu.cn/help/kubernetes/)
- [Kubernetes：安装 kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/)
- [Kubernetes 版本偏差策略](https://kubernetes.io/releases/version-skew-policy/)
