package compress

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// A high-quality JPEG scan, two pages, searchable text and a filename containing
// spaces/Unicode/punctuation exercises the real subprocess, not a mock.
func TestGhostscriptScannedPDFIntegration(t *testing.T) {
	g, err := FindGhostscript(filepath.Join("..", "..", "resources"))
	if err != nil {
		t.Skip("install Ghostscript or prepare the bundled engine for integration tests")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "扫描件 ! original.pdf")
	writeScan(t, input)
	original, _ := os.ReadFile(input)
	service := Service{Converter: g}
	conservative, err := service.Compress(context.Background(), Request{Input: input, TargetBytes: 250000}, func(p Progress) {
		if p.DPI < 150 {
			t.Fatal("default search crossed its quality floor")
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if conservative.Bytes >= conservative.OriginalBytes || conservative.Output == "" {
		t.Fatalf("did not retain a smaller conservative result: %+v", conservative)
	}
	t.Logf("Conservative: %s → %s, fits=%v", FormatBytes(conservative.OriginalBytes), FormatBytes(conservative.Bytes), conservative.MeetsTarget)
	result, err := service.Compress(context.Background(), Request{Input: input, TargetBytes: 250000, AllowLowerQuality: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.MeetsTarget || result.Bytes >= 250000 || result.Bytes >= result.OriginalBytes || result.Output == input {
		t.Fatalf("did not compress the scan to its budget: %+v", result)
	}
	after, _ := os.ReadFile(input)
	if !bytes.Equal(original, after) {
		t.Fatal("modified the original scan")
	}
	textPath := filepath.Join(dir, "text.txt")
	cmd := exec.Command(g.Executable, "-dSAFER", "-dBATCH", "-dNOPAUSE", "-sDEVICE=txtwrite", "-sOutputFile="+textPath, result.Output)
	configureProcess(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("could not reopen the compressed PDF: %v %s", err, out)
	}
	text, _ := os.ReadFile(textPath)
	if strings.Count(string(text), "SEARCHABLE SCAN") != 2 {
		t.Fatalf("lost a page or searchable text: %q", text)
	}
	t.Logf("Scan: %s → %s, %d attempts; both pages' text preserved", FormatBytes(result.OriginalBytes), FormatBytes(result.Bytes), result.Attempts)
	assertNoWorkFiles(t, dir)
}

func writeScan(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1600, 2200))
	var random uint32 = 42
	for y := 0; y < 2200; y++ {
		for x := 0; x < 1600; x++ {
			random = random*1664525 + 1013904223
			v := uint8(235 + (random>>24)%20)
			img.SetRGBA(x, y, color.RGBA{v, v, v, 255})
		}
	}
	drawer := font.Drawer{Dst: img, Src: image.NewUniform(color.RGBA{32, 40, 36, 255}), Face: basicfont.Face7x13}
	for y := 100; y < 2100; y += 40 {
		drawer.Dot = fixed.P(100, y)
		drawer.DrawString("SCAN DOCUMENT   Invoice #2026   Small print must remain legible.   Total: 1280.00")
	}
	var jpg bytes.Buffer
	if err := jpeg.Encode(&jpg, img, &jpeg.Options{Quality: 98}); err != nil {
		t.Fatal(err)
	}
	content := []byte("q 595 0 0 842 0 0 cm /Im0 Do Q\nBT /F1 12 Tf 3 Tr 50 780 Td (SEARCHABLE SCAN) Tj ET\n")
	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /XObject << /Im0 5 0 R >> /Font << /F1 7 0 R >> >> /Contents 6 0 R >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /XObject << /Im0 5 0 R >> /Font << /F1 7 0 R >> >> /Contents 6 0 R >>"),
		append(append([]byte(fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width 1600 /Height 2200 /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>\nstream\n", jpg.Len())), jpg.Bytes()...), []byte("\nendstream")...),
		append(append([]byte(fmt.Sprintf("<< /Length %d >>\nstream\n", len(content))), content...), []byte("endstream")...),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"),
	}
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")
	offsets := []int{0}
	for i, object := range objects {
		offsets = append(offsets, pdf.Len())
		fmt.Fprintf(&pdf, "%d 0 obj\n", i+1)
		pdf.Write(object)
		pdf.WriteString("\nendobj\n")
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	if err := os.WriteFile(path, pdf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}
