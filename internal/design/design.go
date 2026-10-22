// Package design keeps the toolbox's visual and motion language in one place.
// Everything is rendered by MyGo Native, including illustrations and controls.
package design

import (
	"time"

	"github.com/egoist/mygo/ui"
)

var Quick = ui.Animation{Duration: 150 * time.Millisecond, Ease: ui.CubicBezier(.2, .8, .2, 1)}
var Gentle = ui.Animation{Spring: ui.SpringOptions{Duration: 340 * time.Millisecond, Bounce: .08}}
var RowMotion = ui.ElementTransition{
	Animation: &Gentle, Enter: &ui.Motion{Y: 10, Collapse: true}, Exit: &ui.Motion{Y: -5, Collapse: true},
}

func Theme(c *ui.Context) {
	t := *c.Theme()
	t.Radius, t.Spacing = 8, 4
	if t.Dark {
		t.Background, t.Surface, t.SurfaceHover, t.SurfacePressed = ui.Hex("#151a1d"), ui.Hex("#20272b"), ui.Hex("#2b353a"), ui.Hex("#354047")
		t.Text, t.TextMuted, t.Border = ui.Hex("#edf3f1"), ui.Hex("#9aa9ae"), ui.Hex("#344045")
		t.Accent, t.AccentHover, t.AccentPressed = ui.Hex("#81d8be"), ui.Hex("#a2e6d2"), ui.Hex("#67c3a6")
		t.AccentText, t.Success = ui.Hex("#142e26"), ui.Hex("#81d8be")
	} else {
		t.Background, t.Surface, t.SurfaceHover, t.SurfacePressed = ui.Hex("#fafbf9"), ui.Hex("#f0f3ef"), ui.Hex("#e8eee8"), ui.Hex("#dce6dd")
		t.Text, t.TextMuted, t.Border = ui.Hex("#223029"), ui.Hex("#657269"), ui.Hex("#e0e7df")
		t.Accent, t.AccentHover, t.AccentPressed = ui.Hex("#286d53"), ui.Hex("#347f63"), ui.Hex("#205640")
		t.AccentText, t.Success = ui.Hex("#ffffff"), ui.Hex("#286d53")
	}
	t.Focus = t.Accent.Alpha(.5)
	if c.Preferences().HighContrast {
		t.TextMuted = t.Text
		t.Border = t.Text
		t.Focus = t.Accent
	}
	c.SetTheme(&t)
}

func Panel(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.Hex("#1b2225")
	}
	return ui.Hex("#ffffff")
}

// Button keeps pointer/keyboard semantics and native focus rings while adding
// a short color response and a spring that carries velocity on interruption.
func Button(c *ui.Context, label string, icon *ui.SVG, primary, disabled bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Label(label).Disabled(disabled).Gap(7).Center().Padding(9, 13).Radius(9)
	bg, fg, border := Panel(c), t.Text, t.Border
	if primary {
		bg, fg, border = t.Accent, t.AccentText, t.Accent
	}
	pose := ui.Pose{}
	if !disabled {
		if b.Hovered() {
			bg = t.SurfaceHover
			if primary {
				bg = t.AccentHover
			}
			pose.Y = -1
		}
		if b.Pressed() {
			bg, pose.Y, pose.Scale = t.SurfacePressed, 0, .975
			if primary {
				bg = t.AccentPressed
			}
		}
	} else {
		bg, fg, border = t.Surface, t.TextMuted, t.Border
	}
	b.Background(b.AnimateColor("face", bg, Quick)).TextColor(fg).Border(1, border).AnimatePose(pose, Gentle)
	b.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(15, 15)
		}
		ui.Text(c, label).SingleLine().FontWeight(550)
	})
	return b
}

func IconButton(c *ui.Context, label string, icon *ui.SVG, disabled bool) ui.Element {
	b := ui.ButtonBase(c).Label(label).Tooltip(label).Disabled(disabled).Size(28, 28).Center().Radius(7)
	bg := ui.Transparent
	if b.Hovered() && !disabled {
		bg = c.Theme().SurfaceHover
	}
	b.Background(b.AnimateColor("face", bg, Quick)).Children(func() {
		ui.Icon(c, icon).Size(14, 14).TextColor(c.Theme().TextMuted)
	})
	pose := ui.Pose{}
	if b.Pressed() && !disabled {
		pose.Scale = .9
	}
	return b.AnimatePose(pose, Gentle)
}

func svg(paths string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">` + paths + `</svg>`))
}

var Logo = svg(`<rect x="3" y="7" width="18" height="14" rx="4"/><path d="M8 7V5a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M8 12l4 4 4-4M12 10v6"/>`)
var Plus = svg(`<path d="M12 5v14M5 12h14"/>`)
var Arrow = svg(`<path d="M5 12h14M13 6l6 6-6 6"/>`)
var Close = svg(`<path d="m6 6 12 12M6 18 18 6"/>`)
var Shield = svg(`<path d="m12 3 8 3v6c0 5-8 9-8 9s-8-4-8-9V6zM8 12l3 3 5-6"/>`)
var Check = svg(`<path d="m5 12 4 4L19 6"/>`)
var Folder = svg(`<path d="M3 7a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>`)
var Bolt = svg(`<path d="m13 2-9 12h7l-1 8 10-12h-7z"/>`)
var File = svg(`<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8zM14 2v6h6M8 13h8M8 17h5"/>`)

// Scan is the drop area's small editorial illustration. It is a native vector,
// so it stays sharp on Retina displays and animates without a bitmap surface.
var Scan = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="140" height="120" viewBox="0 0 140 120" fill="none"><rect x="24" y="22" width="62" height="82" rx="8" fill="#e9f0e8" stroke="#bdcdbf" transform="rotate(-12 24 22)"/><rect x="54" y="10" width="62" height="84" rx="8" fill="#ffffff" stroke="#99b4a2" transform="rotate(8 54 10)"/><path d="m69 32 29 4M67 42l29 4M66 52l18 2M64 66l28 4M63 76l20 3" stroke="#bfd0c1" stroke-width="3" stroke-linecap="round"/><circle cx="105" cy="88" r="19" fill="#286d53"/><path d="M105 78v19m-6-6 6 6 6-6" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
