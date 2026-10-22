package pdfcompress

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/nerv-base/MyToolbox/internal/compress"
	"github.com/nerv-base/MyToolbox/internal/design"
	"github.com/nerv-base/MyToolbox/internal/toolbox"
)

type batchConverter struct{}

func (batchConverter) Convert(_ context.Context, in, out string, _ compress.Profile) error {
	if filepath.Base(in) == "broken.pdf" {
		return errors.New("损坏的 PDF")
	}
	return os.WriteFile(out, append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("x"), 20000)...), 0600)
}

func queuedFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("x"), 100000)...), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNativeBatchFlowAndFailureIsolation(t *testing.T) {
	updates := make(chan func(), 64)
	tool := New(toolbox.Host{Update: func(fn func()) { updates <- fn }}, batchConverter{}, nil)
	tool.target = "0.05"
	good, bad := queuedFile(t, "good.pdf"), queuedFile(t, "broken.pdf")
	tool.AddFiles([]string{good, good, bad, "invalid.png"})
	if len(tool.entries) != 2 || tool.notice == "" {
		t.Fatal("deduplication or invalid-file feedback failed")
	}
	tt := ui.NewTester(func(c *ui.Context) { design.Theme(c); tool.View(c) }, 790, 800)
	if err := tt.Click("开始压缩"); err != nil {
		t.Fatal(err)
	}
	guard := time.NewTimer(time.Minute)
	defer guard.Stop()
	for tool.running {
		select {
		case update := <-updates:
			update()
		case <-guard.C:
			t.Fatal("batch did not finish")
		}
	}
	tt.Frame()
	tt.Advance(time.Second)
	if tool.entries[0].State != finished || !tool.entries[0].Result.MeetsTarget || tool.entries[1].State != failed {
		t.Fatalf("one failure prevented other results: %+v %+v", tool.entries[0], tool.entries[1])
	}
	if !tt.HasText("打开结果") || !tt.HasText("重试") {
		t.Fatalf("missing result actions: %v", tt.Texts())
	}
	jobs := tool.pending(50000)
	if len(jobs) != 1 || jobs[0] != tool.entries[1] {
		t.Fatal("a completed file should not be recompressed when other files are pending")
	}
	if len(tool.pending(60000)) != 2 {
		t.Fatal("changing the budget should make completed files eligible again")
	}
	if err := tt.Click("清空"); err != nil {
		t.Fatal(err)
	}
	if len(tool.entries) != 0 {
		t.Fatal("clear action failed")
	}
}

type cancelConverter struct{ entered chan struct{} }

func (c cancelConverter) Convert(ctx context.Context, _, _ string, _ compress.Profile) error {
	close(c.entered)
	<-ctx.Done()
	return ctx.Err()
}

func TestCloseCancelsAndJoinsWorker(t *testing.T) {
	updates := make(chan func(), 64)
	entered := make(chan struct{})
	tool := New(toolbox.Host{Update: func(fn func()) { updates <- fn }}, cancelConverter{entered}, nil)
	tool.target = "0.05"
	input := queuedFile(t, "private-scan.pdf")
	tool.AddFiles([]string{input})
	tool.start(tool.entries, false)
	select {
	case <-entered:
	case <-time.After(time.Minute):
		t.Fatal("converter did not start")
	}
	tool.Close()
	select {
	case <-tool.workerDone:
	default:
		t.Fatal("Close returned before subprocess cleanup")
	}
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(input), ".mytoolbox-*"))
	if len(matches) > 0 {
		t.Fatal("private scan temporary data survived app close")
	}
}

func TestNativeTargetPresetsAndReducedMotion(t *testing.T) {
	tool := New(toolbox.Host{}, batchConverter{}, nil)
	tt := ui.NewTester(func(c *ui.Context) { design.Theme(c); tool.View(c) }, 640, 710)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if err := tt.Click("3 MB"); err != nil || tool.target != "3" {
		t.Fatalf("preset failed: %s %v", tool.target, err)
	}
	tt.SetDark(true)
	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("idle/reduced-motion view continues requesting frames")
	}
}
