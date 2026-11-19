# Doter

一个多平台的「顶部刘海」系统监控小部件。收起时是贴在屏幕顶部中央的一条胶囊，悬停展开为实时系统仪表盘。

- **后端**：Go + [Wails v3.1.0](https://v3.wails.io/)（正式版）
- **前端**：React 18 + TypeScript + Vite
- **系统指标**：[gopsutil/v4](https://github.com/shirou/gopsutil) + [distatus/battery](https://github.com/distatus/battery)
- **平台**：Windows / macOS / Linux

## 功能

收起态（30px 高的胶囊）直接显示：

- CPU 使用率
- 本地时间（位于 CPU 与内存之间）
- 内存占用迷你进度条

悬停展开后展示：

- CPU 环形仪表、线程 / 核心数、使用率走势
- 内存占用、Swap、磁盘各卷容量
- 网络上下行速率走势、活动网卡
- 电池电量、充电状态、剩余时间（无电池时显示 AC）
- 系统运行时长、Load Average、架构
- CPU 占用最高的进程列表
- 剪贴板页：文字 / 链接 / 图片 / 富文本历史，支持搜索、置顶、删除、点击写回；记录来源应用，跳过密码管理器与标记为隐藏 / 临时的内容，可随时暂停记录
- 音乐页：系统 Now Playing 的封面、曲名、歌手、专辑与进度，支持播放/暂停、上一首/下一首、拖拽进度、随机/循环（按播放器能力显示）
- Agent 页：监控 Claude Code / Codex CLI 会话，显示运行/等待/结束状态、项目、模型、分支、轮次与累计 / 上下文 token
- 桌面告警：CPU 持续高负载、内存 / Swap 过高、磁盘空间不足、电量过低，以及长时间运行的 Agent 结束或等待输入时发送系统通知；点击通知回到对应页面

点击右上角图钉可保持展开，`Esc` 收起。图钉左侧的设置按钮可开关指标卡片、悬停展开、时钟秒数、桌面告警、剪贴板记录与剪贴板快捷键。托盘菜单可显示 / 隐藏、退出；全局快捷键 `Ctrl+Shift+D` 切换显示 / 隐藏，`Cmd/Ctrl+Shift+V` 直接打开剪贴板页。

> 剪贴板后端目前完整实现于 macOS；Windows / Linux 的对应后端仍为空，页面会显示空历史。macOS 首次启用告警时会请求通知权限，开发构建未签名时可能拿不到授权。

## 目录结构

```
main.go                     应用入口：服务注册、托盘、启动采样
internal/
  notch/notch.go            刘海窗口：无边框、透明、置顶、顶部居中、展开/收起尺寸
  services/
    system.go               SystemService：快照 + 静态信息绑定
    island.go               IslandService：窗口展开/隐藏/退出绑定
    music.go                MusicService：媒体快照、传输控制、事件推送
    agents.go               AgentsService：Claude/Codex 会话快照绑定
    settings.go             SettingsService：告警开关与剪贴板暂停（持久化到 settings.json）
  alerts/
    alerts.go               阈值规则、冷却 / 迟滞状态与 Agent 状态迁移通知
  prefs/
    prefs.go                后端偏好设置（告警、剪贴板暂停、剪贴板快捷键）
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
  clipboard/
    history.go              历史、富文本与图片来源应用
    watcher.go              轮询、暂停、隐藏内容 / 密码管理器过滤
frontend/
  src/
    App.tsx                 悬停展开、图钉、Esc 收起、全局快捷键事件
    components/             CollapsedBar / ExpandedPanel / MusicPanel / AgentsPanel / ClipboardPanel / Ring / Sparkline
    hooks/                  useSystemInfo / useSeries / useClock / useMusic / useAgentSessions / useBackendSettings
    lib/format.ts           字节、速率、时长等格式化
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

## 媒体读取

音乐页只读取系统级的 Now Playing 会话，不直接对接具体播放器。

| 平台 | 实现 | 说明 |
| --- | --- | --- |
| macOS | [mediaremote-adapter](https://github.com/ungive/mediaremote-adapter)（BSD-3-Clause） | 预编译的 universal framework + perl 脚本内嵌在二进制中，第一次打开音乐页时解压到 `~/Library/Caches/doter/mediaremote-adapter/<version>`。macOS 15.4 起直接调用 MediaRemote 会被拒绝，这里借 `/usr/bin/perl` 的 Apple 签名标识继续读取系统会话，因此没有自动化授权弹窗。 |
| Windows | [smtc-suite-go](https://github.com/xiaowumin-mark/smtc-suite-go)（MIT） | 通过 WinRT SMTC 读取当前媒体会话，需 CGO。 |
| Linux | — | 暂不支持，音乐页显示明确的空态。 |

适配器按面板生命周期启停：打开音乐页时启动，全部离开后 20 秒停止；应用退出时由 `ServiceShutdown` 终止辅助进程。

## 后续可扩展

- 采样指标（风扇、温度、GPU）接入 `internal/sysinfo`
- 多显示器选择 / 拖动吸附
- 透明区域点击穿透（Windows `WM_NCHITTEST` / macOS `hitTest`）