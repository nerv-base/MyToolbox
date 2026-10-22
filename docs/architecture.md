# 模块架构

## 依赖方向

`main → toolbox + tools/pdfcompress → compress`

- `main.go`：应用生命周期、原生窗口、导航、系统菜单、文件关联与主机能力。
- `internal/toolbox`：工具契约与注册表，不导入任何具体工具。
- `internal/design`：原生视觉样式、控件、SVG 图标和动效参数。
- `internal/tools/pdfcompress`：PDF 工具自己的状态、原生视图和任务编排。
- `internal/compress`：与 UI 无关的压缩服务、字节预算搜索与 Ghostscript 适配器。
- `resources/<os>-<arch>`：平台专属的离线引擎。MyGo 打包时只拷贝当前平台资源。

所有工具均在编译时注册。主程序通过元数据绘制导航，通过 `View` 显示工具，
通过 `Close` 释放其任务。文件工具选择性实现 `FileReceiver`，未来非文件工具
不必携带文件处理接口。

## 新增工具

在 `internal/tools/<name>` 定义状态和构造函数，实现：

```go
type Tool interface {
    Info() Info
    View(*ui.Context)
    Close()
}
```

`Info` 包含稳定且唯一的 ID、名称、说明和 `*ui.SVG` 图标。
调用 `registry.Register(yourtool.New(host))` 即可加入导航；重复 ID 会报错。
工具应使用 `internal/design` 中的主题、按钮和动效，保持一致的体验。

主机 `Host` 提供 UI 更新调度、文件选择、打开结果和定位文件。
新工具按需使用这些能力，也可以持有自己的纯 Go 业务服务。

## 线程与任务

视图及其状态只在主线程读取和修改。压缩在单独的 goroutine 中顺序处理，
避免同时解码多份高分辨率扫描件造成内存峰值。
后台通过 `Window.Update` 投递所有进度及结果更新。构建期的 `ui.Element`
和 `*ui.Context` 不会传给后台任务或保存到长期状态。

`context.Context` 控制取消；Ghostscript 使用 `exec.CommandContext`。
每次引擎尝试最多 10 分钟。关闭窗口时取消并等待任务清理，再退出应用。

## 压缩策略

每次从原文件重新生成，避免多次有损转码叠加：

| 路径 | DPI / JPEG QFactor（越小画质越高） |
| --- | --- |
| 默认 | 300/.15 → 240/.25 → 200/.35 → 175/.45 → 150/.55 |
| 用户允许降低画质 | 120/.70 → 100/.85 → 85/1.00 → 72/1.30 |

保留每次尝试中体积最小的结果，达到 `目标字节数 × 99%` 后结束。
结果大于原件时不输出副本；已经达标时直接引用原件。
显式降低画质若反而产生更大结果，工具保留此前较好的输出。
画质下限是启发式策略，不做 OCR 准确率或人类可读性的保证。

不整页栅格化。Ghostscript 重写对象并压缩内嵌图片，尽量保留文字层和矢量内容。
原件始终保留，复杂 PDF 的交互功能与签名不能保证在重写后保持有效。

## 文件保护

输入和输出检测 PDF 标识；Ghostscript 进一步解析完整 PDF，解析错误停止该文件。
在原目录建立权限为 0700 的随机工作目录，结束、失败、取消都清理临时扫描数据。
输出以 `O_EXCL` 创建并安全编号，避免覆盖、同名竞态或同名符号链接。
写入失败时移除不完整结果；文件操作遵守操作系统的现有读写权限。
命令参数通过参数数组传递，文件名不进入 shell；引擎启用 SAFER，日志捕获有大小上限。

## 分发

源码中的压缩接口允许未来更换引擎。首版固定 Ghostscript 10.08.0，
macOS/Linux 从校验后的源码编译；Windows 从校验后的官方 x64 包提取。
构建流程为每份应用附上引擎源码与工具箱源码，保留 AGPL 及所有依赖许可证。
应用 UI 本身始终是 MyGo Native，由 Metal / Direct3D / OpenGL 或 CPU 渲染。
