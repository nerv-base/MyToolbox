package compress

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Ghostscript struct {
	Executable string
	Root       string
}

// FindGhostscript prefers the version shipped in the app. PATH is only a
// development fallback; release builds include the complete runtime.
func FindGhostscript(resources string) (*Ghostscript, error) {
	name := "gs"
	if runtime.GOOS == "windows" {
		name = "gswin64c.exe"
	}
	if path := os.Getenv("MYTOOLBOX_GS"); path != "" {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(absolute); err != nil {
			return nil, fmt.Errorf("MYTOOLBOX_GS: %w", err)
		}
		return &Ghostscript{Executable: absolute}, nil
	}
	for _, root := range []string{
		filepath.Join(resources, "ghostscript"),
		filepath.Join(resources, runtime.GOOS+"-"+runtime.GOARCH, "ghostscript"),
	} {
		path := filepath.Join(root, "bin", name)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return &Ghostscript{Executable: path, Root: root}, nil
		}
	}
	if path, err := exec.LookPath(name); err == nil {
		return &Ghostscript{Executable: path}, nil
	}
	return nil, errors.New("压缩组件未就绪，请安装包含压缩组件的完整版本")
}

func (g *Ghostscript) Convert(ctx context.Context, input, output string, p Profile) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	args := []string{
		"-dSAFER", "-dBATCH", "-dNOPAUSE", "-dNOPROMPT", "-dPDFSTOPONERROR",
		"-sDEVICE=pdfwrite", "-dCompatibilityLevel=1.7",
		"-dDetectDuplicateImages=true", "-dCompressFonts=true", "-dSubsetFonts=true",
		"-dAutoRotatePages=/None", "-dPassThroughJPEGImages=false", "-dPassThroughJPXImages=false",
		"-dDownsampleColorImages=true", "-dColorImageDownsampleType=/Bicubic",
		fmt.Sprintf("-dColorImageResolution=%d", p.DPI), "-dColorImageDownsampleThreshold=1.0",
		"-dDownsampleGrayImages=true", "-dGrayImageDownsampleType=/Bicubic",
		fmt.Sprintf("-dGrayImageResolution=%d", p.DPI), "-dGrayImageDownsampleThreshold=1.0",
		"-dDownsampleMonoImages=true", "-dMonoImageResolution=300",
		"-dAutoFilterColorImages=false", "-dColorImageFilter=/DCTEncode",
		"-dAutoFilterGrayImages=false", "-dGrayImageFilter=/DCTEncode",
		"-sOutputFile=" + output,
	}
	if g.Root != "" {
		lib := []string{filepath.Join(g.Root, "Resource", "Init"), filepath.Join(g.Root, "lib"), filepath.Join(g.Root, "Resource", "Font")}
		args = append(args, "-I"+strings.Join(lib, string(os.PathListSeparator)))
	}
	params := fmt.Sprintf("<< /ColorImageDict << /QFactor %.2f >> /GrayImageDict << /QFactor %.2f >> >> setdistillerparams", p.QFactor, p.QFactor)
	args = append(args, "-c", params, "-f", input)
	cmd := exec.CommandContext(ctx, g.Executable, args...)
	configureProcess(cmd)
	var log limitedLog
	cmd.Stdout, cmd.Stderr = &log, &log
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		message := strings.ToLower(string(log.data))
		if strings.Contains(message, "password") || strings.Contains(message, "encrypted") {
			return errors.New("此 PDF 已加密，请先在其他工具中解锁后重试")
		}
		return fmt.Errorf("无法压缩此 PDF，文件可能损坏或包含不受支持的内容：%w", err)
	}
	return nil
}

// A malicious or damaged PDF must not grow the process's captured log forever.
type limitedLog struct{ data []byte }

func (l *limitedLog) Write(p []byte) (int, error) {
	n := len(p)
	if left := 16384 - len(l.data); left > 0 {
		l.data = append(l.data, p[:min(left, n)]...)
	}
	return n, nil
}
