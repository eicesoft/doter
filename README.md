# Doter

一个多平台的「顶部刘海」系统监控小部件。收起时是贴在屏幕顶部中央的一条胶囊，悬停展开为实时系统仪表盘。

- **后端**：Go + [Wails v3.1.0](https://v3.wails.io/)（正式版）
- **前端**：React 18 + TypeScript + Vite
- **系统指标**：[gopsutil/v4](https://github.com/shirou/gopsutil) + [distatus/battery](https://github.com/distatus/battery)
- **天气**：[Open-Meteo](https://open-meteo.com/)（免 API Key）+ [ipwho.is](https://ipwho.is/) 粗定位
- **平台**：Windows / macOS / Linux

## 功能

收起态是贴在屏幕顶部中央的一条胶囊，左右两个指标位可分别选择：

- CPU 使用率 / 内存占用 / 网络下行速率 / 电池电量 / 运行中的 Agent 数 / 时钟 / 天气
- 默认左侧 CPU、右侧天气

展开后是一个带原生磨砂背景的仪表盘，分六个页签：

### 系统总览

- CPU 环形仪表、线程 / 核心数、使用率走势
- 内存占用、Swap、磁盘容量
- 网络上下行速率走势、活动网卡
- 电池电量、充电状态、剩余时间（无电池时显示 AC）
- 系统运行时长、Load Average、架构
- 进程榜：按 CPU / 内存排序，支持按名称或路径搜索、结束进程、在文件管理器中定位

### 日历

- 月 / 周 / 日三种视图，默认月视图；支持前后翻页与「今天」快速定位
- 新建 / 编辑日程：标题、日期、开始 / 结束时间（留空为全天），支持删除
- 月视图按天显示日程摘要；周 / 日视图按小时排布时间块，全天日程单独一行
- 日程保存在 webview 本地存储，重启后仍保留

### 剪贴板

- 文字 / 链接 / 图片 / 富文本（HTML / RTF）历史，支持搜索、置顶、删除、点击写回
- 记录来源应用；跳过密码管理器与标记为隐藏 / 临时的内容
- 排除应用名单、历史条数、文本预览、图片上限均可在设置里调整
- 可选「复制后自动粘贴」，回到打开剪贴板前的应用执行粘贴
- 可随时暂停记录；键盘可用方向键选择、Enter 写回、`Cmd/Ctrl+1…9` 直接跳行

### 天气

- 当前天气：温度、体感、湿度、风力 / 阵风 / 风向、降水、云量、气压、能见度、紫外线、日出日落
- 5 天预报：天气、温度区间、降水概率
- 首次按公网 IP 粗定位；可搜索城市固定地点，选择保存在 `weather.json`
- 预报 10 分钟内存缓存；公制单位（°C、km/h、hPa、km）

### 音乐

- 系统 Now Playing 的封面、曲名、歌手、专辑与进度
- 播放 / 暂停、上一首 / 下一首、拖拽进度、随机 / 循环（按播放器能力显示）

### Agent 监控

- 监控 Claude Code / Codex CLI 会话，显示运行 / 等待 / 结束状态
- 项目、模型、分支、轮次与累计 / 上下文 token

### 跨页与系统能力

- 桌面告警：CPU 持续高负载、内存 / Swap 过高、磁盘空间不足、电量过低，以及长时间运行的 Agent 结束或等待输入时发送系统通知；点击通知回到对应页面
- 全屏 / 演示时自动隐藏岛屿，退出全屏后恢复
- 开机自启、单实例运行；设置页与托盘菜单可检查 GitHub Releases 更新
- 全局快捷键 `Ctrl+Shift+D` 显示 / 隐藏，`Cmd/Ctrl+Shift+V` 打开剪贴板页

点击右上角图钉可保持展开，`Esc` 收起。图钉左侧的设置按钮打开「界面 / 剪贴板 / 系统」三个标签页：界面页控制指标卡片、折叠胶囊、底部页签、悬停展开与时钟秒数；剪贴板页控制记录开关、自动粘贴、容量、排除应用与快捷键；系统页控制桌面告警、开机自启与全屏自动隐藏。底部页签可单独隐藏（日历 / 剪贴板 / 天气 / 音乐 / Agent 监控），「系统总览」始终保留。页脚可检查更新与恢复界面默认值。

> 富文本三平台均采集：macOS 使用原生 pasteboard，Windows / Linux 通过系统剪贴板的多格式接口读取 HTML / RTF（Wayland 取决于合成器是否支持 data-control）。来源应用在 Windows 取自剪贴板 owner，Linux 仅在 X11 下可用。
>
> 自动粘贴会激活打开剪贴板前的应用：macOS 需要辅助功能权限，Linux 需要 `xdotool`（X11）；失败时面板会提示，内容仍在剪贴板中。
>
> 密码管理器内容依赖系统敏感标记与排除名单双重过滤；Linux 的来源应用识别在 Wayland 或没有 X11 的会话中可能为空。
>
> Linux 的全屏检测依赖 `xprop`；没有 XWayland 的 Wayland 会话会保持可见。
>
> 天气依赖公网：无网络时天气页显示空态。预报只在内存中缓存，重启后需要重新请求；地点选择会持久化。
>
> 更新源为 GitHub Releases（`eicesoft/doter`）；发布资产文件名需包含平台和架构（如 `doter-darwin-arm64.zip`），可选提供 `checksums.txt` 校验和。`appVersion` 需要与 `build/config.yml` 的 `info.version` 保持一致。

## 目录结构

```
main.go                     应用入口：服务注册、托盘、快捷键、启动采样
internal/
  notch/
    notch.go                刘海窗口：无边框、透明、置顶、顶部居中、展开/收起尺寸
    glass_darwin.go         macOS 原生磨砂层；其他平台空实现
    hover_darwin.go         应用未激活时也能收到悬停事件
    camera_housing_darwin.go 摄像头刘海宽度与内置屏定位
  services/
    system.go               SystemService：快照、静态信息、进程搜索 / 结束 / 定位
    island.go               IslandService：窗口展开/隐藏/退出绑定
    weather.go              WeatherService：预报与城市搜索绑定
    music.go                MusicService：媒体快照、传输控制、事件推送
    agents.go               AgentsService：Claude/Codex 会话快照绑定
    clipboard.go            ClipboardService：历史、置顶、删除、写回
    settings.go             SettingsService：告警、剪贴板、自启与全屏偏好（settings.json）
    update.go               UpdateService：版本号与 GitHub Releases 更新流程
  weather/
    weather.go              Open-Meteo 预报、IP 定位、城市搜索、10 分钟缓存
    codes.go                WMO 天气代码 → 中文描述
  alerts/
    alerts.go               阈值规则、冷却 / 迟滞状态与 Agent 状态迁移通知
  fullscreen/
    fullscreen_*.go         macOS / Windows / Linux 全屏与演示状态检测
  prefs/
    prefs.go                后端偏好设置（告警、剪贴板、自启、全屏隐藏）
  agents/
    monitor.go              发现近 24h 会话、匹配运行中的 CLI 进程、聚合快照
    claude.go               ~/.claude/projects 日志增量解析
    codex.go                ~/.codex/sessions rollout 日志增量解析
  music/
    player.go               平台媒体适配器接口
    player_darwin.go        macOS：内嵌 mediaremote-adapter，首次运行解压到缓存
    player_windows.go       Windows：SMTC（smtc-suite-go）
    player_unsupported.go   其余平台：返回明确的不支持状态
    assets/                 内嵌的 MediaRemote 适配器与许可
  sysinfo/
    types.go                前后端共享的数据结构
    sampler.go              每秒采样、5 秒磁盘/进程、10 秒电池
    process_search.go       全量进程搜索（名称 / 可执行文件路径）
    process_actions.go      结束进程前的 PID + 启动时间校验
    reveal_*.go             各平台文件管理器定位
  clipboard/
    config.go               历史条数 / 预览大小 / 图片上限默认值与排除名单
    history.go              历史、富文本、来源应用与动态容量
    watcher.go              轮询、暂停、排除名单、图片上限与隐藏内容过滤
    paste.go                自动粘贴入口（按平台分派）
    paste_darwin.go         macOS：激活目标应用并发送 Cmd+V
    paste_windows.go        Windows：激活目标窗口并发送 Ctrl+V
    paste_linux.go          Linux：xdotool 激活窗口并发送 Ctrl+V
    clipboard_darwin.go     macOS 原生 pasteboard + 富文本
    clipboard_lib.go        Windows / Linux 文字、图片与 HTML / RTF 多格式后端
    clipboard_windows.go    Windows 剪贴板序列号与来源应用
    clipboard_linux.go      Linux 变更监听与 X11 来源应用
frontend/
  src/
    App.tsx                 悬停展开、图钉、Esc 收起、全局快捷键和通知跳转
    components/             CollapsedBar / ExpandedPanel / TabBar / WeatherPanel / WeatherIcon
                            MusicPanel / AgentsPanel / ClipboardPanel / SettingsPanel / Ring / Sparkline
    hooks/                  useSystemInfo / useSeries / useClock / useWeather / useMusic
                            useAgentSessions / useClipboard / useBackendSettings
    lib/                    format.ts 字节、速率、时长等格式化；settings.ts 前端偏好
  bindings/                 Wails 自动生成的 TypeScript 绑定（勿手改）
build/                      各平台打包配置与图标
```

## 开发

前置：Go 1.26+、Node 20+、Wails v3 CLI、Windows 需 WebView2 Runtime。

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0

# 生成 TypeScript 绑定（修改 Go 服务方法后）
wails3 generate bindings -ts

# 开发模式（前后端热更新）
wails3 dev

# 生产构建
wails3 build

# 单元测试；实时天气接口默认跳过
go test ./...
DOTER_LIVE_WEATHER=1 go test ./internal/weather -run TestLiveForecast
```

## 窗口实现说明

窗口由 `internal/notch` 统一创建：

| 平台 | 实现 |
| --- | --- |
| Windows | 无边框 + 透明 WebView2、`HiddenOnTaskbar`、关闭系统阴影圆角 |
| macOS | `NSPanel`（`NonActivating` / `FloatingPanel`）、`MacWindowLevelStatus`、跨 Space 常驻、无 Dock 图标（Accessory） |
| Linux | GTK 透明无边框窗口 |

原生窗口在「胶囊尺寸」与「仪表盘尺寸」之间切换，React 只负责在窗口内部做宽度/高度过渡动画。两个尺寸定义在 `internal/notch/notch.go` 顶部，必须与 `frontend/src/styles.css` 里的 `calc()` 留白一致。

Wails v3.1 自带的 `NewNotchWindow`（真实 macOS 摄像头刘海开窗）尺寸固定、无法在展开时调整，因此这里使用可动态调整的自定义面板窗口；在带刘海的 MacBook 上窗口位于菜单栏下方，避免被摄像头区域遮挡。

## 天气数据

只使用免密钥的公共接口，不内置任何密钥：

| 用途 | 接口 | 说明 |
| --- | --- | --- |
| 当前天气与 5 天预报 | `api.open-meteo.com/v1/forecast` | `timezone=auto`，返回 WMO 天气代码、日出日落等；`codes.go` 转中文描述 |
| 城市搜索 | `geocoding-api.open-meteo.com/v1/search` | `language=zh`，最多 8 条结果 |
| 首次定位 | `ipwho.is` | 按公网 IP 取城市与经纬度 |

首次打开天气页时若没有已保存的地点，会先按 IP 定位并把结果写入 `weather.json`；手动选择城市会覆盖该文件。预报在内存中保留 10 分钟，避免频繁请求；请求超时为 10 秒。

## 剪贴板设置

设置弹窗分为「界面 / 剪贴板 / 系统」三个标签页，剪贴板页的选项会立即写入 `settings.json` 并应用到运行时：

| 选项 | 默认值 | 范围 / 说明 |
| --- | --- | --- |
| 历史条数 | 100 | 10–1000；降低时立即裁掉最旧的条目 |
| 文本预览 | 4 KB | 1–64 KB；只限制面板里显示的文本，完整内容仍保留 |
| 图片上限 | 16 MB | 1–64 MB；超过上限的新图片不会入库 |
| 排除应用 | 常见密码管理器 | Bundle ID（macOS）或进程名（Windows / Linux），不区分大小写，`.exe` 可省略 |
| 自动粘贴 | 关闭 | 写回后激活打开剪贴板前的应用并发送粘贴快捷键 |
| 暂停记录 / 快捷键 | 按原设置 | 与之前一致 |

排除名单替换后会立即停止匹配来源的后续复制；系统标记为隐藏 / 临时的内容仍由 `watcher` 直接跳过。剪贴板历史和图片分别保存在 `clipboard.json` 与 `clipboard-images/`。

## 待补齐功能

按「影响现有功能完整性 → 平台/体验缺口 → 能力扩展」排序。

| 优先级 | 模块 | 缺口 | 位置 |
| --- | --- | --- | --- |
| P1 | 磁盘 | 采样层最多返回 4 个卷，卡片只渲染前 2 个，多卷机器会漏显示 | `sampler.go collectDisks`、`ExpandedPanel.tsx` |
| P1 | 进程搜索 | 搜索命中不计算 CPU%，且先按内存截断 30 条再交给 UI 排序；搜索时 CPU 列显示为 `—` | `process_search.go`、`ExpandedPanel.tsx` |
| P1 | CPU | 每秒采集的 `perCore` 没有展示，只用线程数兜底 | `sysinfo/types.go`、`ExpandedPanel.tsx` |
| P1 | 网络 | 速率是全部物理网卡的聚合值，卡片却显示当前最忙网卡的名字；多网卡时名称与数值不一致 | `collectNetwork`、`ExpandedPanel.tsx` |
| P1 | 告警 | 阈值、持续时间与冷却硬编码；设置页只有总开关，没有阈值或免打扰配置 | `alerts/alerts.go`、`SettingsPanel.tsx` |
| P1 | 告警 | 通知授权结果只写日志，设置页无法区分「已开启」与「系统未授权」 | `main.go requestNotificationPermission` |
| P1 | 天气 | 固定公制单位；无手动刷新；请求失败时不回退上一次预报 | `weather.go`、`useWeather.ts` |
| P1 | 设置 | 「恢复界面默认」只重置前端偏好，不回滚告警 / 剪贴板 / 自启 / 全屏等后端设置 | `SettingsPanel.tsx` |
| P2 | 音乐 | 无音量 / 静音控制；Linux 没有 Now Playing 适配 | `music/player.go`、`player_unsupported.go` |
| P2 | Agent | 仅支持 Claude Code / Codex，未覆盖 Gemini CLI、Cursor 等；无跳转终端或结束会话动作 | `agents/monitor.go` |
| P2 | 窗口 | 固定内置屏 / 当前屏，无显示器选择、拖动吸附、透明区域点击穿透 | `notch/notch.go` |
| P3 | 剪贴板 | Linux 的自动粘贴依赖 xdotool，来源应用在 Wayland 下为空 | `paste_linux.go`、`clipboard_linux.go` |
| P3 | 采样 | 温度、风扇、GPU 未接入 | `sysinfo` |
| P3 | 天气 | 露点、当前降水量、日最高 UV / 风速已取回但未展示 | `weather.go`、`WeatherPanel.tsx` |
| P3 | 更新 | 仅手动检查，无启动静默检查或更新渠道选择 | `services/update.go` |
| P3 | 界面 | 卡片标题用英文、其余文案用中文，未做本地化 | `ExpandedPanel.tsx` |
