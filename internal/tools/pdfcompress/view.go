package pdfcompress

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/nerv-base/MyToolbox/internal/compress"
	"github.com/nerv-base/MyToolbox/internal/design"
)

func (t *Tool) View(c *ui.Context) {
	theme := c.Theme()
	_, height := c.Size()
	gap, padding := float32(22), float32(30)
	if height < 720 {
		gap, padding = 14, 20
	}
	ui.Column(c).Fill().Padding(padding, 36, 24).Gap(gap).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, "文档工具").FontSize(11).TextColor(theme.TextMuted)
			ui.Text(c, "/").FontSize(11).TextColor(theme.Border)
			ui.Text(c, "PDF 压缩").FontSize(11).TextColor(theme.TextMuted)
			ui.Spacer(c)
			ui.Icon(c, design.Shield).Size(13, 13).TextColor(theme.Accent)
			ui.Text(c, "全程本地").FontSize(11).TextColor(theme.Accent)
		})
		ui.Column(c).Gap(8).Children(func() {
			ui.Text(c, "小一点，刚刚好。").FontSize(32).FontWeight(650).LetterSpacing(-.7)
			ui.Text(c, "为上传瘦身，让扫描件保留清晰细节。").FontSize(13).TextColor(theme.TextMuted)
		})
		if t.engineErr != nil {
			ui.Text(c, t.engineErr.Error()).TextColor(theme.Danger)
		}
		t.dropArea(c)
		t.targetArea(c)
		target, targetErr := compress.ParseTarget(t.target)
		jobs := t.pending(target)
		if targetErr != nil {
			ui.Text(c, targetErr.Error()).FontSize(12).TextColor(theme.Danger)
		}
		if t.notice != "" {
			ui.Text(c, t.notice).FontSize(12).TextColor(theme.Warning).
				Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 4}, Exit: &ui.Motion{}})
		}
		ui.Column(c.Key("queue")).Grow(1).MinHeight(90).Gap(12).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, "文件队列").FontWeight(600).FontSize(12)
				ui.Textf(c, "%02d", len(t.entries)).FontSize(11).TextColor(theme.TextMuted).
					ContentTransition(ui.ContentTransition{Numeric: true})
				ui.Spacer(c)
				if len(t.entries) > 0 {
					design.IconButton(c, "添加 PDF", design.Plus, t.running || t.picking).OnClick(t.pickFiles)
					design.Button(c, "清空", nil, false, t.running).Padding(5, 9).FontSize(11).
						OnClick(func() { t.entries, t.notice = nil, "" })
				}
			})
			ui.Scroll(c.Key("file-list")).Grow(1).Gap(9).Children(func() {
				if len(t.entries) == 0 {
					ui.Column(c.Key("empty")).FillWidth().Padding(22, 14).Center().Gap(8).Children(func() {
						ui.Icon(c, design.File).Size(24, 24).TextColor(theme.Border)
						ui.Text(c, "你的文件会出现在这里").FontSize(12).TextColor(theme.TextMuted)
						ui.Text(c, "支持批量添加，每份文件独立压缩").FontSize(11).TextColor(theme.TextMuted)
					})
				}
				for _, e := range t.entries {
					t.row(c, e)
				}
			})
		})
		ui.Divider(c)
		ui.Row(c).Gap(16).Children(func() {
			ui.Column(c).Grow(1).Gap(6).Children(func() {
				if t.running {
					ui.Textf(c, "已完成 %d / %d 份", t.done, t.total).FontSize(12).FontWeight(550).
						ContentTransition(ui.ContentTransition{Numeric: true})
					ui.Progress(c, float64(t.done)/float64(max(1, t.total))).Label("批量压缩进度").Height(3).MaxWidth(240)
				} else {
					var saved int64
					for _, e := range t.entries {
						if e.State == finished {
							saved += e.Result.OriginalBytes - e.Result.Bytes
						}
					}
					label := "原件保留，结果另存到原文件夹"
					if saved > 0 {
						label = "已为你节省 " + compress.FormatBytes(saved)
					}
					ui.Text(c, label).FontSize(12).FontWeight(550).ContentTransition(ui.ContentTransition{Numeric: saved > 0})
					ui.Text(c, "清晰度优先 · 自动选择压缩参数").FontSize(11).TextColor(theme.TextMuted)
				}
			})
			if t.running {
				ui.Spinner(c).Size(17, 17).TextColor(theme.Accent)
				design.Button(c, "取消压缩", nil, false, false).OnClick(func() {
					if t.cancel != nil {
						t.cancel()
					}
				})
			} else {
				label, icon := "开始压缩", design.Arrow
				if len(t.entries) > 0 && len(jobs) == 0 {
					label, icon = "已处理全部", design.Check
				}
				design.Button(c, label, icon, true, len(jobs) == 0 || targetErr != nil || t.engineErr != nil).
					Padding(12, 22).OnClick(func() { t.start(jobs, false) })
			}
		})
	})
}

func (t *Tool) dropArea(c *ui.Context) {
	theme := c.Theme()
	_, height := c.Size()
	dropHeight := float32(174)
	if height < 720 {
		dropHeight = 142
	}
	zone := ui.Row(c.Key("drop-zone")).FillWidth().Height(dropHeight).Center().Gap(18).
		Radius(16).Background(design.Panel(c)).Border(1, theme.Border).BorderStyle(ui.BorderDashed)
	drag := zone.FileDragOver() && !t.running
	bg, border := design.Panel(c), theme.Border
	pose := ui.Pose{}
	if drag {
		bg, border, pose.Scale = theme.Accent.Alpha(.06), theme.Accent, 1.008
	}
	zone.Background(zone.AnimateColor("face", bg, design.Quick)).
		BorderColor(zone.AnimateColor("border", border, design.Quick)).AnimatePose(pose, design.Gentle)
	if paths := zone.DroppedFiles(); len(paths) > 0 {
		t.AddFiles(paths)
	}
	zone.Children(func() {
		art := ui.Image(c, design.Scan).Size(116, 104)
		artPose := ui.Pose{}
		if drag {
			artPose = ui.Pose{Y: -5, Rotate: -4, Scale: 1.04}
		}
		art.AnimatePose(artPose, design.Gentle)
		ui.Column(c).Gap(9).Children(func() {
			label := "将 PDF 拖到这里"
			if drag {
				label = "松开，交给工具箱"
			}
			ui.Text(c, label).FontSize(17).FontWeight(600).ContentTransition(ui.ContentTransition{})
			ui.Text(c, "单份或多份，都可以。").FontSize(12).TextColor(theme.TextMuted)
			design.Button(c, "选择 PDF…", design.Plus, false, t.running || t.picking).Padding(7, 12).
				AlignSelf(ui.Start).OnClick(t.pickFiles)
		})
	})
}

func (t *Tool) targetArea(c *ui.Context) {
	theme := c.Theme()
	ui.Row(c).FillWidth().Gap(12).Padding(14, 16).Radius(12).Background(design.Panel(c)).Border(1, theme.Border).Children(func() {
		ui.Column(c).Grow(1).Gap(4).Children(func() {
			ui.Text(c, "目标大小").FontSize(12).FontWeight(600)
			ui.Text(c, "每份文件的上传上限").FontSize(11).TextColor(theme.TextMuted)
		})
		ui.TextInput(c.Key("target-size"), &t.target).Label("目标大小").Width(74).Height(36).FontSize(18).
			FontWeight(600).TextAlign(ui.Center).Background(theme.Surface).Border(0, ui.Transparent).Radius(8).Disabled(t.running)
		ui.Text(c, "MB").FontSize(11).TextColor(theme.TextMuted)
		ui.Box(c).Width(1).Height(22).Margin(0, 4).Background(theme.Border)
		for _, value := range []string{"3", "5", "10"} {
			b := design.Button(c.Key(value), value+" MB", nil, false, t.running).Padding(8, 10).FontSize(11)
			if t.target == value {
				b.Background(theme.Accent.Alpha(.09)).TextColor(theme.Accent).BorderColor(theme.Accent.Alpha(.18))
			}
			b.OnClick(func() { t.target = value })
		}
	})
}

func (t *Tool) row(c *ui.Context, e *entry) {
	theme := c.Theme()
	card := ui.Column(c.Key(e.Path)).FillWidth().Padding(14).Gap(10).Radius(12).
		Background(design.Panel(c)).Border(1, theme.Border).Transition(design.RowMotion)
	card.Children(func() {
		ui.Row(c).Gap(12).Children(func() {
			ui.Box(c).Size(34, 40).Center().Radius(7).Background(theme.Surface).Children(func() {
				ui.Icon(c, design.File).Size(19, 19).TextColor(theme.Accent)
			})
			ui.Column(c).Grow(1).MinWidth(0).Gap(5).Children(func() {
				ui.Text(c, filepath.Base(e.Path)).SingleLine().FontSize(12).FontWeight(600)
				label, color := t.statusText(e, theme)
				ui.Text(c, label).FontSize(11).TextColor(color).ContentTransition(ui.ContentTransition{})
			})
			ui.Column(c).AlignItems(ui.End).Gap(4).Children(func() {
				value := compress.FormatBytes(e.Size)
				if e.State == finished {
					value = compress.FormatBytes(e.Result.Bytes)
				}
				ui.Text(c, value).FontSize(13).FontWeight(600).ContentTransition(ui.ContentTransition{Numeric: true, Down: true})
				if e.State == finished && e.Result.Bytes < e.Size {
					ui.Textf(c, "−%.0f%%", 100*(1-float64(e.Result.Bytes)/float64(e.Size))).FontSize(10).TextColor(theme.Success)
				}
			})
			design.IconButton(c, "移除 "+filepath.Base(e.Path), design.Close, t.running).OnClick(func() { t.remove(e) })
		})
		if e.State == working {
			ui.Progress(c, -1).Label("正在压缩 " + filepath.Base(e.Path)).Height(3)
		}
		if e.State == finished || e.State == failed || e.State == canceled {
			ui.Row(c.Key("result-actions")).Gap(7).Wrap().Transition(ui.ElementTransition{
				Duration: 220 * time.Millisecond, Enter: &ui.Motion{Y: 4},
			}).Children(func() {
				if e.Result.Output != "" {
					design.Button(c, "打开结果", nil, false, t.running).FontSize(11).Padding(5, 9).
						OnClick(func() { t.host.OpenFile(e.Result.Output) })
					design.IconButton(c, "在文件夹中显示", design.Folder, t.running).
						OnClick(func() { t.host.RevealFile(e.Result.Output) })
				}
				if e.State == finished && !e.Result.MeetsTarget {
					design.Button(c, "继续压缩（降低画质）", nil, false, t.running).FontSize(11).Padding(5, 9).
						OnClick(func() { t.start([]*entry{e}, true) })
				} else if e.State == failed || e.State == canceled {
					design.Button(c, "重试", nil, false, t.running).FontSize(11).Padding(5, 9).
						OnClick(func() { t.start([]*entry{e}, false) })
				}
			})
		}
	})
}

func (t *Tool) remove(e *entry) {
	for i, candidate := range t.entries {
		if candidate == e {
			t.entries = append(t.entries[:i], t.entries[i+1:]...)
			return
		}
	}
}

func (t *Tool) statusText(e *entry, theme *ui.Theme) (string, ui.Color) {
	switch e.State {
	case working:
		if e.Progress.Total > 0 {
			return fmt.Sprintf("正在压缩 · 尝试 %d / %d", e.Progress.Attempt, e.Progress.Total), theme.Accent
		}
		return "正在分析文件…", theme.Accent
	case canceled:
		return "已取消，可重新压缩", theme.TextMuted
	case failed:
		return e.Error, theme.Danger
	case finished:
		target := " · 上限 " + compress.FormatBytes(e.Result.TargetBytes)
		if e.Result.AlreadyFits {
			return "原文件已达标，直接使用即可" + target, theme.Success
		}
		if e.Result.MeetsTarget {
			return "压缩完成，已达到目标" + target, theme.Success
		}
		if e.Result.Output == "" {
			return "原件已接近最小体积，尚未达到目标" + target, theme.Warning
		}
		return "已保留较清晰结果，尚未达到目标" + target, theme.Warning
	}
	return "等待压缩 · " + compress.FormatBytes(e.Size), theme.TextMuted
}
