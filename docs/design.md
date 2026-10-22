# 界面与动效

## 视觉

温和、清晰、克制。暖白背景、墨绿强调色，深色模式使用石墨底色与浅薄荷绿。
界面使用系统字体，中文自然回退；字号和控件颜色由 MyGo 的主题体系承载。
插画与图标均由原生 SVG 绘制，支持高 DPI 显示。

固定工具导航，默认直接进入 PDF 压缩页。拖放、目标、队列、结果有清晰的视觉顺序，
主要操作始终在底部。最低窗口宽度 820，高度 640；长文件队列可滚动。
色彩强调主操作和状态，同时以文字说明结果，避免只依赖红绿区分。

## 原生动效

借鉴 Motion 的可中断弹簧、布局过渡，以及 SwiftUI 的数值内容过渡。
实现使用 MyGo 自带 `AnimatePose`、`AnimateColor`、`ElementTransition`、
`ContentTransition` 与 `Timeline`。

| 场景 | 行为 |
| --- | --- |
| 页面首次出现 | 280ms 淡入与 340ms / 8px 的轻微上移，仅首次播放 |
| 按钮悬停 | 150ms 颜色过渡，340ms 轻弹簧上移 1px |
| 按钮按压 | 缩至 97.5%，松开后保留速度并回弹 |
| 拖入文件 | 边框与底色响应，区域缩放至 100.8%，文档插画轻微抬起 |
| 队列增删 | 有稳定 key 的布局过渡，新增/移除与相邻文件平滑归位 |
| 结果出现 | 220ms / 4px 的轻微淡入，操作按钮就近出现 |
| 数值变化 | 文件计数、体积、节省量使用数值内容过渡 |
| 处理进度 | 真实完成份数；当前文件使用不确定进度，避免伪造百分比 |

减少动态效果遵循系统 `ReduceMotion`：位移、缩放和弹簧归位直接完成，
必要状态依然清晰。空闲界面不持续申请动画帧；离开视口的进度动画不持续绘制。
所有按钮保留键盘焦点、原生可访问语义、Tab/Enter/Space 操作和焦点环。

设计实现与 MyGo 的公开文档对应：

- https://github.com/egoist/mygo/blob/main/docs/ui/motion.md
- https://github.com/egoist/mygo/blob/main/docs/ui/transitions.md
- https://github.com/egoist/mygo/blob/main/docs/ui/custom-widgets.md

浅色、深色、紧凑布局与真实 Linux 窗口截图由原生渲染器验证。
