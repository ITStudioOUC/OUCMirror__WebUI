本站提供 Jenkins 安装文件，包括 DEB、RPM、WAR 和插件文件。当前镜像没有 APT 的 `Release`、`Packages` 或 RPM 仓库元数据，因此不能将这些下载目录直接作为 APT、DNF 软件源。

## 选择发行通道

- [LTS DEB 安装包](/mirrors/jenkins/debian-stable/)
- [LTS RPM 安装包](/mirrors/jenkins/rpm-stable/)
- [LTS WAR](/mirrors/jenkins/war-stable/)
- [其他文件与每周版本](/mirrors/jenkins/)

生产环境通常选择 LTS。具体版本以目录为准，升级已有实例前请备份 `JENKINS_HOME`，并查阅对应版本的升级指南。

## Debian / Ubuntu 安装示例

先安装受目标 Jenkins 版本支持的 Java。以下示例使用 OpenJDK 21，需系统仓库中提供该软件包：

```sh
sudo apt update
sudo apt install fontconfig openjdk-21-jre
java -version
```

从 LTS DEB 目录中选择版本并下载。以下 `2.580.2` 是文件名示例，不代表最新版本：

```sh
curl -fLO https://mirrors.app.itstudio.club/mirrors/jenkins/debian-stable/jenkins_2.580.2_all.deb
sudo apt install ./jenkins_2.580.2_all.deb
sudo systemctl enable --now jenkins
```

访问服务器的 `8080` 端口，按页面提示完成首次初始化。管理员初始密码按 Jenkins 页面给出的本机文件位置获取，不要发布该文件内容。

此方式安装的是单个软件包。后续升级需再次下载并安装新版本；需要包管理器自动更新时，请按 Jenkins 官方说明配置其官方软件仓库。

## 使用 WAR 运行

需要临时运行或自行管理 Java 进程时，可以下载 LTS WAR：

```sh
curl -fL https://mirrors.app.itstudio.club/mirrors/jenkins/war-stable/latest/jenkins.war -o jenkins.war
java -jar jenkins.war --httpPort=8080
```

`latest` 会随同步变化，运行前请根据所下载版本的 Java 支持表确认运行环境。长期部署应配置适当的进程管理、访问控制和备份。

## 校验与插件

安装前核对 Jenkins 官方发布的版本及校验值；RPM 安装还应按官方说明导入并核对签名密钥。插件目录本身不是更新中心配置，不能把它当作 `update-center.json` 地址填写。

## 参考资料

- [Jenkins Linux 安装说明](https://www.jenkins.io/doc/book/installing/linux/)
- [Jenkins WAR 安装说明](https://www.jenkins.io/doc/book/installing/war-file/)
- [Jenkins Java 支持策略](https://www.jenkins.io/doc/book/platform-information/support-policy-java/)
- [Jenkins 升级指南](https://www.jenkins.io/doc/upgrade-guide/)
