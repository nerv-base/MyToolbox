//go:generate go run ./cmd/icon

package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/ui"
	"github.com/nerv-base/MyToolbox/internal/compress"
	"github.com/nerv-base/MyToolbox/internal/design"
	"github.com/nerv-base/MyToolbox/internal/toolbox"
	"github.com/nerv-base/MyToolbox/internal/tools/pdfcompress"
)

const appName = "MyToolbox!!!!!"

type application struct {
	registry toolbox.Registry
	selected string
	window   *mygo.Window
}

func (a *application) view(c *ui.Context) {
	design.Theme(c)
	theme := c.Theme()
	ui.Row(c).Fill().Gap(0).Children(func() {
		ui.Column(c).Width(210).FillHeight().Background(theme.Surface).Padding(18, 14).Gap(32).
			BorderWidth(0, 1, 0, 0).BorderColor(theme.Border).Children(func() {
			ui.Row(c).Gap(10).Padding(9, 4).Children(func() {
				ui.Box(c).Size(34, 34).Radius(10).Center().Background(theme.Accent).Children(func() {
					ui.Icon(c, design.Logo).Size(21, 21).TextColor(theme.AccentText)
				})
				ui.Column(c).Gap(4).Children(func() {
					ui.Text(c, "MyToolbox!!!!!").FontSize(15).FontWeight(650).LetterSpacing(-.3)
					ui.Text(c, "让日常，轻一点。").FontSize(10).TextColor(theme.TextMuted)
				})
			})
			ui.Sidebar(c, &a.selected, func() {
				ui.SidebarSection(c, "我的工具", nil, func() {
					for _, tool := range a.registry.Tools() {
						info := tool.Info()
						ui.SidebarItem(c, info.ID, info.Icon, info.Name)
					}
				})
			}).Grow(1)
			ui.Column(c).Gap(12).Padding(4).Children(func() {
				ui.Row(c).Gap(7).Children(func() {
					ui.Icon(c, design.Shield).Size(13, 13).TextColor(theme.Accent)
					ui.Text(c, "文件留在你的设备上").FontSize(10).TextColor(theme.TextMuted)
				})
				ui.Text(c, "v0.1.0").FontSize(10).TextColor(theme.TextMuted)
			})
		})
		pane := ui.Column(c.Key("tool-pane")).Grow(1).FillHeight()
		intro := pane.Timeline("intro", ui.TimelineOptions{})
		opacity := intro.Tween(ui.Tween{From: 0, To: 1, Duration: 280 * time.Millisecond})
		y := intro.Tween(ui.Tween{From: 8, To: 0, Duration: 340 * time.Millisecond}, ui.WithPrevious(0))
		pane.Opacity(opacity).Translate(0, y).Children(func() {
			if tool := a.registry.Get(a.selected); tool != nil {
				tool.View(c)
			}
		})
	})
}

func (a *application) addFiles(paths []string) {
	if receiver, ok := a.registry.Get(a.selected).(toolbox.FileReceiver); ok {
		receiver.AddFiles(paths)
	}
}

func main() {
	a := &application{selected: "pdf-compress"}
	var earlyFiles []string
	mygo.App.OnOpenFile(func(path string) {
		if a.window == nil {
			earlyFiles = append(earlyFiles, path)
		} else {
			a.window.Update(func() { a.addFiles([]string{path}) })
		}
	})
	mygo.App.WhenReady(func() {
		resources, err := mygo.App.Path(mygo.PathResources)
		if err != nil {
			log.Fatal(err)
		}
		engine, engineErr := compress.FindGhostscript(resources)
		var converter compress.Converter
		if engineErr == nil {
			converter = engine
		}
		host := toolbox.Host{
			Update: func(fn func()) { a.window.Update(fn) },
			PickFiles: func(title string, extensions []string, done func([]string, error)) {
				go func() {
					paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
						Parent: a.window, Title: title, Multiple: true,
						Filters: []mygo.FileFilter{{Name: "PDF", Extensions: extensions}},
					})
					a.window.Update(func() { done(paths, err) })
				}()
			},
			OpenFile: func(path string) {
				go func() {
					if err := mygo.Shell.OpenPath(path); err != nil {
						mygo.Dialog.Error("无法打开文件", err.Error())
					}
				}()
			},
			RevealFile: func(path string) { go mygo.Shell.ShowItemInFolder(path) },
		}
		if err := a.registry.Register(pdfcompress.New(host, converter, engineErr)); err != nil {
			log.Fatal(err)
		}
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu},
			{Label: "文件", Submenu: []*mygo.MenuItem{
				{Label: "添加 PDF…", Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) {
					host.PickFiles("选择 PDF", []string{"pdf"}, func(paths []string, err error) {
						if err == nil {
							a.addFiles(paths)
						}
					})
				}},
				mygo.Separator(),
				{Role: mygo.RoleQuit},
			}},
			{Role: mygo.RoleEditMenu, Label: "编辑"},
			{Role: mygo.RoleWindowMenu, Label: "窗口"},
		}))
		a.window = mygo.NewWindow(mygo.WindowOptions{
			Title: appName, Width: 1000, Height: 760, MinWidth: 820, MinHeight: 640,
			StateKey: "main", Content: ui.View(a.view),
		})
		a.window.OnFileDrop(func(e *mygo.FileDropEvent) { a.window.Update(func() { a.addFiles(e.Paths) }) })
		a.window.OnClosed(a.registry.Close)
		if len(earlyFiles) > 0 {
			a.addFiles(earlyFiles)
			earlyFiles = nil
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
