// Package pdfcompress implements one independently registered native tool.
package pdfcompress

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/nerv-base/MyToolbox/internal/compress"
	"github.com/nerv-base/MyToolbox/internal/toolbox"
)

var pdfIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6M8 13h8M8 17h5"/></svg>`))

type status int

const (
	queued status = iota
	working
	finished
	failed
	canceled
)

type entry struct {
	Path     string
	Size     int64
	State    status
	Result   compress.Result
	Error    string
	Progress compress.Progress
}

type Tool struct {
	host       toolbox.Host
	service    compress.Service
	engineErr  error
	entries    []*entry
	target     string
	notice     string
	running    bool
	picking    bool
	done       int
	total      int
	cancel     context.CancelFunc
	workerDone chan struct{}
}

func New(host toolbox.Host, converter compress.Converter, engineErr error) *Tool {
	return &Tool{host: host, service: compress.Service{Converter: converter}, engineErr: engineErr, target: "5"}
}

func (*Tool) Info() toolbox.Info {
	return toolbox.Info{ID: "pdf-compress", Name: "压缩 PDF", Description: "为上传瘦身，保留清晰细节", Icon: pdfIcon}
}

func (t *Tool) Close() {
	if t.cancel != nil {
		t.cancel()
	}
	// Window.Update posts without waiting. It is safe to join here even when
	// closing on the UI thread, so subprocesses and temporary scans are cleaned
	// before the application process exits.
	if t.workerDone != nil {
		<-t.workerDone
	}
}

// AddFiles also handles the shell's file drops. Paths are canonicalized so a
// repeated drop or an alias cannot enqueue the same input twice.
func (t *Tool) AddFiles(paths []string) {
	if t.running {
		t.notice = "正在处理当前批次，完成后可以继续添加文件。"
		return
	}
	known := make(map[string]bool)
	for _, e := range t.entries {
		known[pathKey(e.Path)] = true
	}
	var rejected []string
	for _, path := range paths {
		absolute, err := filepath.Abs(path)
		if err != nil {
			rejected = append(rejected, filepath.Base(path))
			continue
		}
		if resolved, err := filepath.EvalSymlinks(absolute); err == nil {
			absolute = resolved
		}
		if known[pathKey(absolute)] {
			continue
		}
		if !strings.EqualFold(filepath.Ext(absolute), ".pdf") {
			rejected = append(rejected, filepath.Base(path))
			continue
		}
		size, err := compress.ValidatePDF(absolute)
		if err != nil {
			rejected = append(rejected, filepath.Base(path)+"："+err.Error())
			continue
		}
		known[pathKey(absolute)] = true
		t.entries = append(t.entries, &entry{Path: absolute, Size: size})
	}
	t.notice = ""
	if len(rejected) > 0 {
		t.notice = "已跳过无效文件：" + strings.Join(rejected, "；")
	}
}

func pathKey(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func (t *Tool) pickFiles() {
	if t.host.PickFiles == nil || t.running || t.picking {
		return
	}
	t.picking = true
	t.host.PickFiles("选择要压缩的 PDF", []string{"pdf"}, func(paths []string, err error) {
		t.picking = false
		if err != nil {
			t.notice = "无法选择文件：" + err.Error()
			return
		}
		t.AddFiles(paths)
	})
}

func (t *Tool) start(entries []*entry, lowerQuality bool) {
	if t.running || len(entries) == 0 {
		return
	}
	target, err := compress.ParseTarget(t.target)
	if err != nil {
		t.notice = err.Error()
		return
	}
	if lowerQuality && len(entries) == 1 && entries[0].Result.TargetBytes != 0 {
		// The row's consent applies to the target shown beside that result.
		target = entries[0].Result.TargetBytes
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel, t.running, t.done, t.total = cancel, true, 0, len(entries)
	workerDone := make(chan struct{})
	t.workerDone = workerDone
	t.notice = ""
	jobs := append([]*entry(nil), entries...)
	previous := make(map[*entry]compress.Result, len(jobs))
	for _, e := range jobs {
		previous[e] = e.Result
		e.State, e.Error, e.Progress = queued, "", compress.Progress{}
	}
	go func() {
		defer close(workerDone)
		defer cancel()
		for _, e := range jobs {
			if ctx.Err() != nil {
				t.host.Update(func() { e.State = canceled })
				continue
			}
			t.host.Update(func() { e.State = working })
			result, err := t.service.Compress(ctx, compress.Request{
				Input: e.Path, TargetBytes: target, AllowLowerQuality: lowerQuality,
			}, func(p compress.Progress) {
				t.host.Update(func() { e.Progress = p })
			})
			t.host.Update(func() {
				t.done++
				switch {
				case errors.Is(err, context.Canceled):
					e.State = canceled
				case err != nil:
					e.State, e.Error = failed, err.Error()
				default:
					old := previous[e]
					// Keep a better result if an explicit lower-quality retry grows.
					if lowerQuality && old.Output != "" && old.Bytes <= result.Bytes && old.TargetBytes == result.TargetBytes {
						if result.Output != "" && !result.AlreadyFits {
							_ = os.Remove(result.Output)
						}
						result = old
					}
					e.State, e.Result, e.Size = finished, result, result.OriginalBytes
				}
			})
		}
		t.host.Update(func() {
			t.running, t.cancel = false, nil
			if ctx.Err() != nil {
				t.notice = "已取消。已完成的结果仍保留在原文件夹。"
			}
		})
	}()
}

// Finished files are reused when new files are added. Changing the budget
// makes them eligible again, without generating duplicate outputs by default.
func (t *Tool) pending(target int64) []*entry {
	var jobs []*entry
	for _, e := range t.entries {
		if e.State != finished || e.Result.TargetBytes != target {
			jobs = append(jobs, e)
		}
	}
	return jobs
}
