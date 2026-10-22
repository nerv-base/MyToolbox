package pdfcompress

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/nerv-base/MyToolbox/internal/compress"
	"github.com/nerv-base/MyToolbox/internal/design"
	"github.com/nerv-base/MyToolbox/internal/toolbox"
)

// This optional storyboard renders sample states to inspect native motion. It
// does not run compression; real compression is verified by the integration test.
func TestNativeMotionStoryboard(t *testing.T) {
	dir := os.Getenv("MYTOOLBOX_MOTION_PREVIEW")
	if dir == "" {
		t.Skip("set MYTOOLBOX_MOTION_PREVIEW to export native frames")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	tool := New(toolbox.Host{}, nil, nil)
	tool.target = "0.25"
	tt := ui.NewTester(func(c *ui.Context) { design.Theme(c); tool.View(c) }, 790, 780)
	frame := 0
	capture := func(count int) {
		for range count {
			tt.Advance(time.Second / 30)
			f, err := os.Create(filepath.Join(dir, fmt.Sprintf("frame-%03d.png", frame)))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, tt.Image()); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			frame++
		}
	}
	capture(18)
	e := &entry{Path: "iPhone 扫描件.pdf", Size: 2163196}
	tool.entries = []*entry{e}
	tt.Frame()
	capture(18)
	tool.running, tool.total, e.State = true, 1, working
	e.Progress = compress.Progress{Attempt: 2, Total: 5}
	tt.Frame()
	capture(24)
	tool.running, tool.done, e.State = false, 1, finished
	e.Result = compress.Result{OriginalBytes: e.Size, Bytes: 280850, TargetBytes: 250000, Output: "iPhone 扫描件_compressed.pdf"}
	tt.Frame()
	capture(24)
	e.Result.Bytes, e.Result.MeetsTarget = 139500, true
	tt.Frame()
	capture(24)
	if tt.FramePending() {
		t.Fatal("completed storyboard still animates")
	}
}
