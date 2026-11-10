# Doter

一个多平台的「顶部刘海」系统监控小部件。收起时是贴在屏幕顶部中央的一条胶囊，悬停展开为实时系统仪表盘。

- **后端**：Go + [Wails v3.1.0](https://v3.wails.io/)（正式版）
- **前端**：React 18 + TypeScript + Vite
- **系统指标**：[gopsutil/v4](https://github.com/shirou/gopsutil) + [distatus/battery](https://github.com/distatus/battery)
- **平台**：Windows / macOS / Linux

## 功能

收起态（44px 高的胶囊）直接显示：

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

点击右上角图钉可保持展开，`Esc` 收起。托盘菜单可显示 / 隐藏、退出；全局快捷键 `Ctrl+Shift+D` 切换显示 / 隐藏。

## 目录结构

```
main.go                     应用入口：服务注册、托盘、启动采样
internal/
  notch/notch.go            刘海窗口：无边框、透明、置顶、顶部居中、展开/收起尺寸
  services/
    system.go               SystemService：快照 + 静态信息绑定
    island.go               IslandService：窗口展开/隐藏/退出绑定
  sysinfo/
    types.go                前后端共享的数据结构
    sampler.go              每秒采样、5 秒磁盘/进程、10 秒电池
frontend/
  src/
    App.tsx                 悬停展开、图钉、Esc 收起
    components/             CollapsedBar / ExpandedPanel / Ring / Sparkline
    hooks/                  useSystemInfo / useSeries / useClock
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

## 后续可扩展

- 采样指标（风扇、温度、GPU）接入 `internal/sysinfo`
- 多显示器选择 / 拖动吸附
- 透明区域点击穿透（Windows `WM_NCHITTEST` / macOS `hitTest`）
- 设置面板与指标开关