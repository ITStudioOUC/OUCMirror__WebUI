本站镜像 Kubernetes 的版本化软件包仓库，可用于安装 `kubelet`、`kubeadm` 和 `kubectl`。容器镜像、容器运行时及集群网络插件需要另行配置。

本文以 `v1.35` 仓库为例。它是示例版本，不代表最新版本；已有集群应根据官方升级流程及版本偏差策略选择次版本，并在所有仓库地址中保持一致。修改软件源不会自动完成集群升级。

## Debian / Ubuntu

安装所需工具：

```sh
sudo apt update
sudo apt install ca-certificates curl gpg
sudo install -d -m 0755 /etc/apt/keyrings
```

从 Kubernetes 官方仓库获取该通道的签名公钥。如果同名密钥文件已经存在，先核对、备份，再更新：

```sh
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.35/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
sudo chmod 0644 /etc/apt/keyrings/kubernetes-apt-keyring.gpg
```

备份已有 `/etc/apt/sources.list.d/kubernetes.list`，然后写入：

```text
deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://mirrors.app.itstudio.club/mirrors/kubernetes/core:/stable:/v1.35/deb/ /
```

更新索引并查看可安装版本：

```sh
sudo apt update
apt-cache madison kubeadm
```

新节点需要安装组件时，可执行：

```sh
sudo apt install kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
```

已有集群请按升级指南逐个处理节点，不要直接套用新节点安装步骤。

## RPM 系统

备份已有 `/etc/yum.repos.d/kubernetes.repo`，配置为：

```ini
[kubernetes]
name=Kubernetes
baseurl=https://mirrors.app.itstudio.club/mirrors/kubernetes/core:/stable:/v1.35/rpm/
enabled=1
gpgcheck=1
gpgkey=https://pkgs.k8s.io/core:/stable:/v1.35/rpm/repodata/repomd.xml.key
exclude=kubelet kubeadm kubectl cri-tools kubernetes-cni
```

刷新缓存并查看版本：

```sh
sudo dnf makecache
dnf --disableexcludes=kubernetes list --showduplicates kubeadm
```

新节点安装组件时使用 `sudo dnf install kubelet kubeadm kubectl --disableexcludes=kubernetes`。`exclude` 用于避免普通系统更新意外升级集群组件。

## 检查地址

本站使用 `core:/stable:/v1.xx/` 目录布局，不使用旧的 `apt.kubernetes.io` 地址。其他版本请先查看 [仓库目录](/mirrors/kubernetes/core:/stable:/)是否已同步，并确认该版本仍符合项目的支持策略。

## 参考资料

- [TUNA Kubernetes 使用帮助](https://mirrors.tuna.tsinghua.edu.cn/help/kubernetes/)
- [Kubernetes：安装 kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/)
- [Kubernetes 版本偏差策略](https://kubernetes.io/releases/version-skew-policy/)
