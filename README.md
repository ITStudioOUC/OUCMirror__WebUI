# MirrorOUC

中国海洋大学软件镜像源（MirrorOUC）网页界面

本项目计划提供镜像检索、同步状态、使用帮助、软件下载及站点公告，并配套管理后台，用于维护站点内容、软件源配置和同步任务

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
npm run check:monitoring  # 检查监控演示数据、时间窗口和资源总量
npm run build       # TypeScript 检查与生产构建
npm run preview     # 预览 dist 中的构建结果
```

## 当前界面

Vue 3 + TypeScript + Vite，使用 vue-i18n 管理界面文案。当前首页支持镜像搜索、同步信息滚动动画、公告和帮助弹窗、亮色／暗色／跟随系统，以及中英文切换。语言和主题选择保存在当前浏览器。

语言入口显示当前语言名称，点击后打开与站点主题一致的自定义下拉菜单；支持方向键、Home/End 定位、Enter/空格选择、Esc 取消，以及 Tab 离开。选项来自 `src/locales/index.ts` 中注册的语言，名称使用各语言文件的 `common.languageName`；注册新语言后，选择器、语言偏好恢复和镜像搜索会自动使用新的语言列表。

桌面端使用三列镜像列表与侧栏；手机端使用折叠导航。表格的后两列每 5 秒在「最近同步 + Size」与「下次同步 + Status」之间统一切换。表头每轮向上滚动一次，条目内容随机滚动 3～6 次后落定；使用系统的减少动态效果设置时直接显示最终值。页面隐藏时暂停轮换并取消未完成的动画。项目说明在悬浮提示及详情中查看。页脚使用宽幅分栏结构，在手机端纵向排列。新增页面也需兼顾手机端，并验证 320、390、768 和 1440 像素宽度下的中英文与明暗主题。

当前为界面原型，15 个镜像、仓库容量及同步任务均为演示数据，尚未连接镜像服务器。示例配置使用 `example.invalid` 保留域名；下载入口前往项目官网。Markdown 公告发布、管理后台、真实目录与 tunasync 接入属于后续开发范围，详细方案见 [plan.md](plan.md)。

`/status` 提供服务器监控与同步明细：IPv4 / IPv6 流量、两组磁盘容量、IOPS、吞吐量、CPU 和内存图表；可切换 1 / 6 / 24 小时范围，默认每 5 秒刷新图表，页面隐藏时暂停刷新。支持暂停图表刷新、悬停或方向键查看时间点、点击图例隐藏曲线，以及按名称或上游地址搜索同步任务。桌面端图表双列排列，手机端单列排列，同步表改为纵向条目。监控数据在 `src/monitoring.ts` 中生成，示例上游元数据在 `src/data.ts` 中，均未连接真实服务器。

## 源码约定

- `src/App.vue`：首页与弹窗交互。
- `src/style.css`：爱特官网同款明暗配色、布局、动画和移动端样式；色值来源与映射见 [DESIGN.md](DESIGN.md)。
- `src/locales/zh-CN/`、`src/locales/en-US/`：界面文案及演示内容；新增文案通过语言键引用。
- 各语言的 `mirrors.ts` 定义表头、状态、相对时间、容量与公告日期模板，`formats.ts` 定义日期和数字格式；组件通过 `t`、`d`、`n` 使用，不自行生成界面文案。
- `src/data.ts`：演示镜像与公告元数据。
- `src/components/StatusPage.vue`、`MonitoringChart.vue`：监控布局、图表与同步明细交互；`src/monitoring.ts` 定义监控数据结构和演示曲线，`src/useMirrorFormats.ts` 复用首页与状态页的时间、容量格式。
- `public/assets/`：页面实际使用的素材。

根目录 `素材/` 是本地素材目录，已加入 `.gitignore`，不得提交；`node_modules/`、`dist/`、`.local/` 和环境变量文件同样不纳入 Git。

中国海洋大学软件镜像源（MirrorOUC）由[中国海洋大学爱特工作室](https://www.itouc.cn/)承建。
