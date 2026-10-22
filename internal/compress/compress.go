// Package compress searches for a PDF that fits a byte budget while keeping
// image quality above an explicit floor. It never rasterizes whole pages.
package compress

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MB uses decimal bytes, matching typical upload limits. Outputs use 1% less
// than the requested limit to accommodate sites with a strict '<' comparison.
const MB int64 = 1_000_000

type Profile struct {
	DPI     int
	QFactor float64 // Ghostscript JPEG quantization: smaller means higher quality.
}

var readable = []Profile{{300, .15}, {240, .25}, {200, .35}, {175, .45}, {150, .55}}
var smaller = []Profile{{120, .7}, {100, .85}, {85, 1}, {72, 1.3}}

type Converter interface {
	Convert(context.Context, string, string, Profile) error
}

type Request struct {
	Input       string
	TargetBytes int64
	// AllowLowerQuality is explicit user consent to go below the default floor.
	AllowLowerQuality bool
}

type Result struct {
	Output        string
	OriginalBytes int64
	Bytes         int64
	TargetBytes   int64
	MeetsTarget   bool
	AlreadyFits   bool
	Attempts      int
}

type Progress struct{ Attempt, Total, DPI int }

type Service struct{ Converter Converter }

func ParseTarget(value string) (int64, error) {
	n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < .01 || n > 10000 {
		return 0, errors.New("请输入 0.01 到 10000 之间的大小（MB）")
	}
	return int64(math.Floor(n * float64(MB))), nil
}

func FormatBytes(n int64) string {
	if n < MB {
		return fmt.Sprintf("%.1f KB", float64(n)/1000)
	}
	return fmt.Sprintf("%.2f MB", float64(n)/float64(MB))
}

func ValidatePDF(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("无法读取文件：%w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() {
		return 0, errors.New("请选择 PDF 文件")
	}
	header := make([]byte, 1024)
	n, err := f.Read(header)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if !bytes.Contains(header[:n], []byte("%PDF-")) {
		return 0, errors.New("文件没有有效的 PDF 标识")
	}
	return info.Size(), nil
}

func (s Service) Compress(ctx context.Context, req Request, progress func(Progress)) (Result, error) {
	result := Result{TargetBytes: req.TargetBytes}
	if req.TargetBytes < 10000 {
		return result, errors.New("目标大小至少为 0.01 MB")
	}
	input, err := filepath.Abs(req.Input)
	if err != nil {
		return result, err
	}
	result.OriginalBytes, err = ValidatePDF(input)
	if err != nil {
		return result, err
	}
	result.Bytes = result.OriginalBytes
	if err := ctx.Err(); err != nil {
		return result, err
	}
	budget := req.TargetBytes - max(1, req.TargetBytes/100)
	if result.Bytes <= budget {
		result.MeetsTarget, result.AlreadyFits, result.Output = true, true, input
		return result, nil
	}
	if s.Converter == nil {
		return result, errors.New("压缩组件未就绪，请安装完整版本")
	}
	work, err := os.MkdirTemp(filepath.Dir(input), ".mytoolbox-")
	if err != nil {
		return result, fmt.Errorf("无法在原文件夹保存结果，请将 PDF 移至可写文件夹：%w", err)
	}
	defer os.RemoveAll(work)
	profiles := readable
	if req.AllowLowerQuality {
		profiles = smaller
	}
	best := ""
	for i, profile := range profiles {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if progress != nil {
			progress(Progress{i + 1, len(profiles), profile.DPI})
		}
		candidate := filepath.Join(work, fmt.Sprintf("attempt-%d.pdf", i))
		if err := s.Converter.Convert(ctx, input, candidate, profile); err != nil {
			return result, err
		}
		result.Attempts++
		size, err := ValidatePDF(candidate)
		if err != nil {
			return result, fmt.Errorf("压缩结果无效：%w", err)
		}
		if size < result.Bytes {
			if best != "" {
				_ = os.Remove(best)
			}
			best, result.Bytes = candidate, size
		} else {
			_ = os.Remove(candidate)
		}
		if result.Bytes <= budget {
			result.MeetsTarget = true
			break
		}
	}
	if best != "" {
		result.Output, err = saveUnique(ctx, best, input)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

// Reserve with O_EXCL: a rerun, another app or a symlink cannot replace a file.
func saveUnique(ctx context.Context, source, original string) (string, error) {
	in, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer in.Close()
	stem := strings.TrimSuffix(original, filepath.Ext(original)) + "_compressed"
	for i := 1; i <= 10000; i++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		name := stem + ".pdf"
		if i > 1 {
			name = fmt.Sprintf("%s_%d.pdf", stem, i)
		}
		out, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("保存失败：%w", err)
		}
		_, copyErr := io.Copy(out, &contextReader{ctx, in})
		if copyErr == nil {
			copyErr = out.Sync()
		}
		closeErr := out.Close()
		if copyErr != nil || closeErr != nil {
			_ = os.Remove(name)
			return "", errors.Join(copyErr, closeErr)
		}
		return name, nil
	}
	return "", errors.New("同名结果过多，请清理原文件夹后重试")
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
