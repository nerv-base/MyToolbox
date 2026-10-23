# 首版验证记录

验证环境：Linux x64，Go 1.27.3，MyGo
`v0.6.3-0.20261022161515-5caa69f2e74c`，随包编译的 Ghostscript 10.08.0。

## 已验证

- `go test -race ./...`：压缩策略、输出保护、任务取消、批量失败隔离、
  注册表、原生界面点击及减少动态效果全部通过。
- `go vet ./...` 与 `go tool mygo vet`：通过。
- `CGO_ENABLED=0`、`mygo_noinspector` 的五个目标组合交叉编译通过：
  `linux/amd64`、`linux/arm64`、`darwin/amd64`、`darwin/arm64`、`windows/amd64`。
- Linux 原生应用已通过 MyGo 打包为可安装归档和 `.deb`，完整带上离线引擎与许可证。
- Linux GTK 实际窗口在虚拟显示器中成功启动；环境无 GPU，使用 MyGo 的 CPU 渲染回退。
- 浅色、深色、紧凑窗口的实际原生渲染已检查，图标、中文文本和控件显示正常。
- 动效演示由 MyGo 的原生渲染器逐帧导出，完成后不再申请空闲动画帧。

## 真实引擎测试样本

自动生成一份含高质量 JPEG 扫描图片与可搜索文字层的双页 PDF：

| 路径 | 原大小 | 目标 | 结果 | 行为 |
| --- | --- | --- | --- | --- |
| 默认清晰度 | 2,163,196 字节 | 250,000 字节 | 280,850 字节 | 提示无法兼顾目标，保留较清晰结果 |
| 允许降低画质 | 同一原件 | 同一目标 | 约 139,500 字节 | 达标，两页文字层均保留 |

文件名含中文、空格和感叹号。测试重新打开输出并读取文字，验证页数及文字层；
检查原件字节未变。测试样本不是用户的真实 iPhone 扫描件，不能代替实际文件的画质验收。

## 原生 CI 构建

2026-10-23，[Verify](https://github.com/nerv-base/MyToolbox/actions/runs/39884118920)
与[五平台构建](https://github.com/nerv-base/MyToolbox/actions/runs/39884168600)均通过。
`darwin/amd64`、`darwin/arm64`、`windows/amd64`、`linux/amd64`、`linux/arm64`
分别在对应原生 runner 上准备离线引擎、执行真实压缩集成测试及原生视图测试、
构建应用并上传产物。每份产物包含引擎与工具箱的对应源码。

## 待实机验收

macOS / Windows 的完整桌面交互、安装器体验，以及 Linux ARM64 的桌面交互仍需实机验证；
CI 中的原生视图与引擎测试不代替这些检查。
macOS Developer ID、公证与 Windows 代码签名证书尚未配置。
