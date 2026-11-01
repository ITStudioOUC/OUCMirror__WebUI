# MirrorOUC

中国海洋大学软件镜像源（MirrorOUC）网页界面

本项目提供镜像检索、同步状态、使用帮助、软件下载及站点公告，并配套管理后台，用于维护站点内容、软件源配置和同步任务

同步引擎采用 [TUNA tunasync](https://github.com/tuna/tunasync)

## 本地预览

使用 Node.js 22.18 或更高版本：

```sh
npm ci
npm run dev
```

打开终端输出的地址，默认是 `http://127.0.0.1:5173/`。

```sh
npm run check:i18n  # 检查中英文语言键与插值参数
npm run check:monitoring  # 检查监控图表的数据夹具与资源总量约束
npm run build       # TypeScript 检查与生产构建
npm run preview     # 预览 dist 中的构建结果
```

## 当前界面

Vue 3 + TypeScript + Vite，使用 vue-i18n 管理界面文案。当前首页支持镜像搜索、同步信息滚动动画、公告和帮助弹窗、亮色／暗色／跟随系统，以及中英文切换。语言和主题选择保存在当前浏览器。

语言入口显示当前语言名称，点击后打开与站点主题一致的自定义下拉菜单；支持方向键、Home/End 定位、Enter/空格选择、Esc 取消，以及 Tab 离开。选项来自 `src/locales/index.ts` 中注册的语言，名称使用各语言文件的 `common.languageName`；注册新语言后，选择器、语言偏好恢复和镜像搜索会自动使用新的语言列表。

桌面端使用三列镜像列表与侧栏；手机端使用折叠导航。表格的后两列每 5 秒在「最近同步 + Size」与「下次同步 + Status」之间统一切换。表头每轮向上滚动一次，条目内容随机滚动 3～6 次后落定；使用系统的减少动态效果设置时直接显示最终值。页面隐藏时暂停轮换并取消未完成的动画。项目说明在悬浮提示及详情中查看。页脚使用宽幅分栏结构，在手机端纵向排列。新增页面也需兼顾手机端，并验证 320、390、768 和 1440 像素宽度下的中英文与明暗主题。

前台已接入真实后端，不会在连接失败时切换为演示数据。首页每 15 秒获取镜像状态，连接中断时保留上一次数据并显示提示。镜像名称直接打开 `/mirrors/<name>/`，使用帮助和公告通过 Markdown 安全渲染。公告只展示已发布内容；公告标题／正文和镜像使用说明均维护一份中文内容，切换界面语言时显示同一版。界面文案、镜像介绍及站点设置继续支持多语言。

`/status` 展示服务器实测 IPv4 / IPv6 HTTPS、rsync 流量，NFS 磁盘容量、IOPS、吞吐量以及 CPU / 内存；支持 1 / 6 / 24 小时范围，图表每 10 秒刷新，页面隐藏时暂停。同步明细另提供按仓库汇总的上游 TCP 拉取速度，约每 2 秒采样与刷新，并适配手机端。统计值从完整采样计算，曲线最多取 241 个点。桌面端图表双列，手机端单列，同步明细在窄屏改为纵向布局。未采集到的指标不会冒充为零。

`/admin` 提供登录、Markdown 公告草稿／发布、镜像展示信息与使用帮助、任务命令和日志、全局／单个镜像调度配置、站点标识／头像／首页相关链接／页脚友链／下载项、管理员和公告编辑账户，以及操作记录。同步间隔遵循 tunasync 的原生语义：从上次同步结束后开始计算，单位为分钟。任务命令只表示已提交，最终状态以 Worker 为准。

操作记录按时间倒序，每页 10 条；超过一页时显示分页控制，可查看全部历史记录。接口 `/api/v1/admin/audit?page=1` 返回当前页、总条数和总页数，未带 `page` 的旧版请求继续返回最近 200 条数组。

## 后端与部署

后端使用 Go 标准库 HTTP、SQLite 和 TOML 解析器，不依赖数据库外部服务。详细部署、权限、备份和回退操作见 [deploy/README.md](deploy/README.md)。

```sh
# 需要 Go 1.27 或更高版本
cd backend
go mod download
go test ./...
go vet ./...
go build -o mirrorouc .
# 首次运行生成初始管理员密码，妥善保存终端输出
./mirrorouc -data ./data -init-admin admin
# 对接自己的 Manager 和 Worker；前端开发服务器已代理 /api 与 /uploads
./mirrorouc -data ./data -manager http://127.0.0.1:12345 -worker YOUR_WORKER_ID
```

生产构建通过 `-static ../dist` 或 `MIRROROUC_STATIC` 指定静态目录；`MIRROROUC_WORKER_CONFIG`、`MIRROROUC_LOGS`、`MIRROROUC_METRICS` 分别指定 Worker 配置、日志根目录和采样历史。缺少配置时，对应管理功能会显示不可用，不会启动假的同步引擎。管理页复用站点的明暗主题和语言选择。

当前服务器预览地址为 `http://100.69.2.2:8080`，管理入口 `/admin`，原有 80 端口下载服务保留。预览仅监听该 Tailnet 地址，正式域名与 HTTPS 尚未配置。

## 源码约定

- `src/App.vue`：首页与弹窗交互。
- `src/style.css`：爱特官网同款明暗配色、布局、动画和移动端样式；色值来源与映射见 [DESIGN.md](DESIGN.md)。
- `src/locales/zh-CN/`、`src/locales/en-US/`：界面文案及数据夹具文案；新增文案通过语言键引用。
- 各语言的 `mirrors.ts` 定义表头、状态、相对时间、容量与公告日期模板，`formats.ts` 定义日期和数字格式；组件通过 `t`、`d`、`n` 使用，不自行生成界面文案。
- `src/data.ts`：镜像类型与旧版界面数据夹具；生产页面通过 `src/usePublicData.ts` 和 `src/api.ts` 获取接口数据。
- `src/components/AdminPage.vue`、`src/locales/admin-messages.ts`：管理表单与中英文文案。
- `backend/`：鉴权、内容管理、tunasync 适配、配置预览与版本恢复、监控接口及集成测试。
- `deploy/`：systemd、Nginx、实测采集器与受限配置写入工具。
- `src/components/StatusPage.vue`、`MonitoringChart.vue`：监控布局、图表与同步明细交互；`src/monitoring.ts` 定义监控数据结构和演示曲线，`src/useMirrorFormats.ts` 复用首页与状态页的时间、容量格式。
- `public/assets/`：页面实际使用的素材。

根目录 `素材/` 是本地素材目录，已加入 `.gitignore`，不得提交；`node_modules/`、`dist/`、`.local/` 和环境变量文件同样不纳入 Git。

中国海洋大学软件镜像源（MirrorOUC）由[中国海洋大学爱特工作室](https://www.itouc.cn/)承建。

首页侧栏“相关链接”在后台「站点设置 → 首页相关链接」管理，支持多语言名称、站内／站外网址、使用帮助入口、增删与排序。它使用独立的 `relatedLinks` 配置；页脚友情链接继续使用 `links`。保存站点设置后，首页下次刷新或轮询即可生效。
