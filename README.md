# MirrorOUC

爱特软件镜像源的前端与管理服务，由[中国海洋大学爱特工作室](https://www.itouc.cn/)制作。使用 Vue 3、TypeScript 和 Go，支持镜像展示、公告、使用帮助与管理后台，同步任务由 [tunasync](https://github.com/tuna/tunasync) 执行。

## 部署

### 使用 Release

从 [Releases](https://git.itouc.cn/ITStudio_OUC/OUCMirror-WebUI/releases) 下载与服务器系统、CPU 架构匹配的预编译包并解压。没有预编译部署包时，按下一节自行编译。

部署目录只需包含：

```text
mirrorouc/
|-- mirrorouc       # 后端程序
`-- dist/           # 前端构建产物
```

### 自行编译

需要 Node.js 22.18+ 和 Go 1.27+。在目标服务器或相同系统与架构的构建环境中，从项目根目录执行：

```bash
npm ci
npm run build
mkdir -p .local
go -C backend build -o ../.local/mirrorouc .
```

将 `.local/mirrorouc` 与完整的 `dist/` 放到同一个部署目录，按下面的步骤启动。

## 对接 tunasync

先启动 tunasync Manager 和 Worker，确认 Manager API 可访问，并取得 Worker 配置中 `[global].name` 对应的 ID。

在部署目录中，首次运行以下命令创建管理员，随机密码会输出到终端：

```bash
./mirrorouc -data ./data -init-admin admin
```

然后启动服务，将 Manager 地址和 `YOUR_WORKER_ID` 替换为实际值：

```bash
./mirrorouc -static ./dist -data ./data -manager http://127.0.0.1:12345 -worker YOUR_WORKER_ID
```

默认监听 `127.0.0.1:8081`，可通过 `-listen` 修改。首页为 `/`，公告页为 `/announcements`，管理后台为 `/admin`，同步状态页为 `/status`。数据库与上传文件保存在 `data/`，更新时保留该目录。

镜像使用帮助位于 `/help/<源名>/`，在后台“镜像与任务”的“展示信息”中使用 Markdown 编辑。所有界面语言使用同一份中文帮助文章；未填写文章的镜像不显示帮助按钮。

帮助正文可插入交互式 Debian 配置块，`format` 使用 `list`（传统格式）或 `deb822`：

```mirror-config
{"type":"debian","format":"list"}
```

镜像目录与下载由已有的 Web 服务器提供，路径为 `/mirrors/<源名>/`。自定义目录页的 Fancyindex 模板随前端构建生成在 `dist/fancy-index/` 中。

需要本机监控采集或受限配置写入时，可按需对接 `backend/tools/` 中的工具。更多启动参数可通过 `./mirrorouc -help` 查看。
