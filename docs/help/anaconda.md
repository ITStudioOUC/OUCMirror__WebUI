本站提供 Conda 软件包频道及部分安装器。频道和平台的同步范围可在 [Anaconda 目录](/mirrors/anaconda/)中查看。

## 配置文件

配置位于用户目录下的 `.condarc`：Linux 和 macOS 使用 `~/.condarc`，Windows 使用 `%USERPROFILE%\.condarc`。先备份已有文件；没有该文件时可运行：

```sh
conda config --set show_channel_urls yes
```

切换前确认所需频道的平台目录已有 `repodata.json`，例如 [conda-forge 的 linux-64 索引](/mirrors/anaconda/cloud/conda-forge/linux-64/)。索引尚未同步完成时，先保留原配置。

## 使用 conda-forge

将 `.condarc` 中相关配置改为以下内容。已有的 `channels` 应一并检查，避免意外混用不同频道的软件包：

```yaml
channels:
  - conda-forge
show_channel_urls: true
channel_priority: strict
custom_channels:
  conda-forge: https://mirrors.app.itstudio.club/mirrors/anaconda/cloud
  bioconda: https://mirrors.app.itstudio.club/mirrors/anaconda/cloud
```

这里的 `custom_channels` 地址不包含频道名，Conda 会自动补上 `conda-forge` 或 `bioconda`。

刷新索引，并在新环境中验证：

```sh
conda clean --index-cache
conda create -n mirror-test --override-channels -c conda-forge python numpy
conda list -n mirror-test --show-channel-urls
```

`--override-channels` 让此次创建只使用指定频道。使用 Bioconda 时，可按下面的顺序指定频道：

```sh
conda create -n bio-test --override-channels -c conda-forge -c bioconda samtools
```

## 使用 defaults

如果项目需要 Anaconda 的 `defaults`，可改用下面这套配置。它是上一套配置的替代方案：

```yaml
channels:
  - defaults
show_channel_urls: true
default_channels:
  - https://mirrors.app.itstudio.club/mirrors/anaconda/pkgs/main
  - https://mirrors.app.itstudio.club/mirrors/anaconda/pkgs/r
  - https://mirrors.app.itstudio.club/mirrors/anaconda/pkgs/msys2
```

使用前请查阅 Anaconda 对相应仓库的使用条款。自定义环境文件中的完整 URL、`-c` 参数和其他配置文件可能覆盖 `.condarc`；可通过 `conda config --show-sources` 排查。

## 下载安装器

- [Miniconda 安装器](/mirrors/anaconda/miniconda/)
- [Anaconda Distribution 安装器](/mirrors/anaconda/archive/)

选择与系统、CPU 架构匹配的文件，安装前与官方发布的 SHA256 校验值比较。本站尚未同步到的文件请从官方获取。

## 参考资料

- [TUNA Anaconda 使用帮助](https://mirrors.tuna.tsinghua.edu.cn/help/anaconda/)
- [Conda 频道配置](https://docs.conda.io/projects/conda/en/stable/user-guide/configuration/use-condarc.html)
- [Bioconda 频道配置](https://bioconda.github.io/user/install.html)
