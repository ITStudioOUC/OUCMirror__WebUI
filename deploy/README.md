# MirrorOUC 部署

当前预览：`http://100.69.2.2:8080`，后台：`/admin`。这是单 Worker 适配版本，使用服务器现有的 tunasync Manager、Worker 和并行 rsync 配置，不重新安装或替换同步引擎。

## 运行结构

| 服务 | 位置与用途 |
| --- | --- |
| Nginx 预览 | 只监听 `100.69.2.2:8080`；原有 80 端口站点保持原配置 |
| mirrorouc.service | `mirrorouc` 普通用户，API 与静态文件监听 `127.0.0.1:8081` |
| mirrorouc-config.service | root 受限配置写入器，Unix socket 只允许 `mirrorouc` 组访问 |
| mirrorouc-metrics.timer | 每 10 秒采集 Linux CPU／内存、NFS `/data` I/O 和磁盘容量 |
| mirrorouc-counters.service | 添加独立 nftables 计数表，只计数，不修改已有规则、不丢弃数据包 |
| mirrorouc-speeds.service | 每约 2 秒读取 TCP 接收计数，按仓库汇总并行连接 |
| tunasync | 继续使用已有 Manager `127.0.0.1:12345` 与 Worker `srv-software-mirror` |

代码发布目录 `/opt/mirrorouc/releases/<release>`；`/opt/mirrorouc/current` 指向当前版本。业务数据库和上传图片保存在本地磁盘 `/var/lib/mirrorouc/`，不要放在 NFS 上。监控历史位于其 `metrics/history.json`，保留最近 24 小时；刚启动时没有完整 24 小时历史。

## 首次安装／更新

本地 Node.js 22.18+ 与 Go 1.27+：

```sh
npm ci
npm run check:i18n
npm run check:monitoring
npm run build
cd backend
go test ./...
go vet ./...
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o ../mirrorouc .
```

把 `mirrorouc`、`dist/` 和 `deploy/` 放进新的 `/opt/mirrorouc/releases/<release>/`，确认二进制可执行。在 Debian 服务器上执行：

```sh
python3 /opt/mirrorouc/releases/<release>/deploy/test_config_helper.py
python3 /opt/mirrorouc/releases/<release>/deploy/test_job_speeds.py
sh /opt/mirrorouc/releases/<release>/deploy/install-metrics.sh
sh /opt/mirrorouc/releases/<release>/deploy/install-preview.sh /opt/mirrorouc/releases/<release>
systemctl status mirrorouc mirrorouc-config mirrorouc-metrics.timer
curl http://127.0.0.1:8081/healthz
nginx -t
```

安装脚本针对当前服务器路径。迁移到其他服务器时，应先调整 `nginx-preview.conf`、环境文件、配置写入器的固定路径与 Manager 地址。脚本不会清理旧发布目录或删除镜像文件。

首次安装会创建管理员 `admin`，随机密码写入 root 才能读取的 `/etc/mirrorouc/initial-admin-password`。本次部署的本地凭据文件为被 Git 忽略的 `.local/deployment/admin-credentials.json`。后台「账户」可修改当前密码，修改后所有旧会话失效。此密码文件不会跟随密码修改而更新。

环境配置文件 `/etc/mirrorouc/mirrorouc.env` 不随发布覆盖，权限 0600。正式启用 HTTPS 后将 `MIRROROUC_SECURE_COOKIES=true` 并重启服务；当前 Tailnet 预览通过 HTTP 访问，故为 false。登录 Cookie 为 HttpOnly、SameSite=Strict，有效期 12 小时；写入接口另需会话 CSRF Token 与同源请求。

## 配置发布和权限

Web 服务不能直接写 `/etc/tunasync`。配置写入器固定只接受 `/etc/tunasync/worker.conf` 和 `/etc/tunasync/mirrors.d/*.conf`：

- 全局只可修改 concurrent、interval、retry、timeout。
- 已有镜像只可修改 upstream、interval、retry、timeout。自定义 command、rsync_options、provider 等字段必须保留。
- 新增只支持 rsync／two-stage-rsync，文件名固定为 `mirrorouc-<name>.conf`；不接受任意 Shell 命令。
- 编辑前后均验证 TOML、数值、上游地址和文件内容哈希。产生冲突时拒绝覆盖。
- 先生成可阅读的完整文件前后预览，再由管理员确认应用；应用失败恢复原文件并尝试重新加载。
- SQLite 保留变更版本与审计记录；应用过的变更可以恢复至变更前，若文件已被其他修改覆盖则拒绝恢复。
- 写入器保留待完成事务日志 `/var/lib/mirrorouc-config/pending.json`；若进程在写入中退出，下次启动恢复原文件。若恢复失败则拒绝继续写入，需要运维检查该文件与 Worker 状态。

Manager 返回成功表示 reload 命令已提交，并不等于 Worker 已完成重载，需查看 Worker 服务和任务状态。已有仓库禁用后用「开始同步」重新启动；tunasync 不提供单独的 enable 命令。

## 监控口径

HTTPS／rsync 使用 TCP 443／873 的 IPv4、IPv6 收发字节计数，双向合计，排除回环接口。8080／80 端口的 HTTP 预览流量不计入 HTTPS，也不展示 HTTP 曲线。没有经过这些端口的流量不会被推算进对应协议。没有有效采样时显示等待采集。

`/status` 同步明细的「拉取速度」来自 Linux TCP_INFO 的 `bytes_received`，通过 `ss -tinpeH` 读取，关联 `tunasync-worker.service` 的进程与仓库目标路径，并追溯 git／ssh 等辅助进程的父进程。多个 rsync 子进程共享同一个 socket 时按 inode／socket cookie 去重，多个并行连接求和；约每 2 秒采样并刷新，单位 B/s、KB/s、MB/s、GB/s。只计算 TCP 接收，不把上传量、NFS I/O 或磁盘写入速度当作拉取速度。

这是采样窗口的接收速率，采样间建立并关闭的短连接可能漏记，不作为精确流量计费数据。首次采样不使用连接生存期累计值冒充瞬时速度。仅同步中的仓库显示数值；有效采样中的零表示当时没有收到数据，未同步／采样失败／数据过期时显示「—」。数据超过 15 秒即失效。相关原始字段见 [Linux TCP_INFO](https://github.com/torvalds/linux/blob/master/include/uapi/linux/tcp.h) 和 [iproute2 ss](https://github.com/iproute2/iproute2/blob/main/misc/ss.c)。

NFS I/O 来自 `/proc/self/mountstats` 的 `/data` 挂载，磁盘容量来自 statvfs；CPU、内存来自 `/proc`。采样历史仅覆盖采集器启用后的时间，不补造旧数据。图表最多展示 241 个采样点，均值、当前值、峰值从当前窗口全部原始有效采样计算。

镜像文件由 Nginx 直接提供，API 不读取整个仓库；镜像目录响应设置 CSP sandbox 和 nosniff，避免仓库中的 HTML／脚本获得管理页面同源权限。

## 备份与回退

日常备份包括 SQLite、`uploads/`、`/etc/mirrorouc/` 和 `/etc/tunasync/`。数据库运行时使用 SQLite backup API 或 `sqlite3 ... '.backup ...'`；直接复制数据库文件前先停止 `mirrorouc.service`，同时处理 WAL 文件。备份文件包含账户哈希、会话和可能带内部地址的配置，应限制读取权限。

应用版本回退：将 `/opt/mirrorouc/current` 指回旧发布目录，再 `systemctl restart mirrorouc`。本次新增数据库表使用 `CREATE TABLE IF NOT EXISTS`；未来若出现不兼容数据库迁移，需要同时恢复对应数据库备份。

只撤下预览时，删除 `/etc/nginx/sites-enabled/mirrorouc-preview`，通过 `nginx -t` 后 reload Nginx，并停止 `mirrorouc.service` 与 `mirrorouc-config.service`。这不会修改原有 80 端口镜像站点。停止新增采集器不会影响 tunasync；若需要删除计数表，只删除 `inet mirrorouc_monitor`，不要 flush 全局 nftables 规则。

## 验证范围

`backend/*_test.go` 使用临时数据库与假的 Manager，覆盖鉴权、角色、CSRF、草稿隔离、并发冲突、任务提交、配置应用／恢复、上游不可用缓存、图片校验及监控统计。`test_config_helper.py` 独立验证写入白名单、自定义命令保留、注入拒绝与失败恢复，不操作真实同步配置。

本次部署核验真实 API、镜像目录、日志、监控与后台草稿保存。没有为验证效果而停止、重启仓库，或应用修改后的生产同步参数。

## 统一中文内容升级

公告的 title／body 和镜像的 help 已改为单个字符串，所有界面语言共用中文内容。旧版本的语言映射在服务启动时以事务迁移：优先保留 zh-CN，其次 zh；没有中文键的历史记录保留原内容供管理员整理。原始 JSON 保存在 SQLite 的 shared_content_backups 表，迁移会递增记录 revision，避免旧编辑页面覆盖新内容；重复启动不会重复迁移。

此次内容格式升级需要在回退旧应用时同时恢复升级前的数据库备份。部署前使用 SQLite backup API 生成一致性快照，备份目录仅 root 可读。迁移不修改镜像介绍、站点文案或同步配置。

## 首页相关链接

`settings/site` 的 `relatedLinks` 独立控制首页侧栏，与页脚 `links` 分开。旧设置缺少该字段时，接口补充原有的三个入口，不改写数据库。显式保存空数组会隐藏该栏目；仍打开旧版表单的客户端提交时，若未携带该字段，后端保留当前已保存的相关链接。条目支持 `kind=help` 的站内使用帮助与 `kind=url` 的 HTTP(S)／站内绝对路径；不接受脚本、data URL、协议相对地址或反斜杠路径。名称支持多语言，数组顺序即显示顺序。
