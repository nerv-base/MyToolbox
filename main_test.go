package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/nerv-base/MyToolbox/internal/toolbox"
	"github.com/nerv-base/MyToolbox/internal/tools/pdfcompress"
)

// The actual application view is rendered by MyGo's CPU renderer, including
// the sidebar, Chinese text, custom native controls and motion, without a webview.
func TestNativeShellLayouts(t *testing.T) {
	a := &application{selected: "pdf-compress"}
	if err := a.registry.Register(pdfcompress.New(toolbox.Host{}, nil, nil)); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.view, 1060, 820)
	tt.Advance(time.Second)
	if !tt.HasText("小一点，刚刚好。") || !tt.HasText("开始压缩") {
		t.Fatalf("missing native content: %v", tt.Texts())
	}
	if tt.FramePending() {
		t.Fatal("idle shell keeps drawing frames")
	}
	if dir := os.Getenv("MYTOOLBOX_PREVIEW"); dir != "" {
		for _, name := range []string{"light", "dark", "compact"} {
			if name == "dark" {
				tt.SetDark(true)
			}
			if name == "compact" {
				tt.SetDark(false)
				tt.SetSize(820, 680)
			}
			tt.Advance(time.Second)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			f, err := os.Create(filepath.Join(dir, "native-"+name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, tt.Image()); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}
