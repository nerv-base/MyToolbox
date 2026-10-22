package compress

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type converterFunc func(context.Context, string, string, Profile) error

func (f converterFunc) Convert(ctx context.Context, in, out string, p Profile) error {
	return f(ctx, in, out, p)
}

func fakePDF(t *testing.T, path string, size int) []byte {
	t.Helper()
	data := append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte("x"), size-9)...)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestTargetValidation(t *testing.T) {
	for _, s := range []string{"", "0", "-1", "NaN", "Inf", "5 MB", "10001", "0.001"} {
		if _, err := ParseTarget(s); err == nil {
			t.Errorf("accepted invalid target %q", s)
		}
	}
	for s, expected := range map[string]int64{"3": 3000000, " 2.5 ": 2500000, "0.01": 10000} {
		if n, err := ParseTarget(s); err != nil || n != expected {
			t.Errorf("ParseTarget(%q) = %d, %v", s, n, err)
		}
	}
}

func TestSearchStopsAtHighestQualityThatFitsAndNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "scan 中文 !.pdf")
	original := fakePDF(t, input, 100000)
	existing := filepath.Join(dir, "scan 中文 !_compressed.pdf")
	existingBytes := fakePDF(t, existing, 40000)
	var attempted []Profile
	s := Service{Converter: converterFunc(func(_ context.Context, in, out string, p Profile) error {
		if in != input {
			t.Fatal("each attempt must start from the original, avoiding generation loss")
		}
		attempted = append(attempted, p)
		size := 70000
		if len(attempted) == 2 {
			size = 45000
		}
		fakePDF(t, out, size)
		return nil
	})}
	result, err := s.Compress(context.Background(), Request{Input: input, TargetBytes: 50000}, nil)
	if err != nil || !result.MeetsTarget || result.Bytes != 45000 || len(attempted) != 2 {
		t.Fatalf("unexpected search: %+v, %v, attempts=%v", result, err, attempted)
	}
	if result.Output != filepath.Join(dir, "scan 中文 !_compressed_2.pdf") {
		t.Fatal("did not reserve a unique output:", result.Output)
	}
	for path, expected := range map[string][]byte{input: original, existing: existingBytes} {
		got, _ := os.ReadFile(path)
		if !bytes.Equal(got, expected) {
			t.Fatalf("overwrote %s", path)
		}
	}
	assertNoWorkFiles(t, dir)
}

func TestQualityFloorKeepsSmallestResultAndRequiresOptIn(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "large.pdf")
	fakePDF(t, input, 100000)
	var dpis []int
	s := Service{Converter: converterFunc(func(_ context.Context, _, out string, p Profile) error {
		dpis = append(dpis, p.DPI)
		if p.DPI < 150 {
			t.Fatal("crossed the quality floor without consent")
		}
		// The smallest result is not necessarily the last one.
		size := 70000
		if p.DPI == 200 {
			size = 60000
		}
		fakePDF(t, out, size)
		return nil
	})}
	r, err := s.Compress(context.Background(), Request{Input: input, TargetBytes: 10000}, nil)
	if err != nil || r.MeetsTarget || r.Bytes != 60000 || r.Output == "" || len(dpis) != len(readable) {
		t.Fatalf("bad fallback: %+v, %v", r, err)
	}
	assertNoWorkFiles(t, dir)
}

func TestAlreadyFitsAndCancellationLeaveOriginalAlone(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "original.pdf")
	fakePDF(t, input, 50000)
	r, err := (Service{}).Compress(context.Background(), Request{Input: input, TargetBytes: 100000}, nil)
	if err != nil || !r.AlreadyFits || r.Output != input {
		t.Fatalf("already fits: %+v %v", r, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := Service{Converter: converterFunc(func(ctx context.Context, _, out string, _ Profile) error {
		fakePDF(t, out, 30000)
		cancel()
		return ctx.Err()
	})}
	_, err = s.Compress(ctx, Request{Input: input, TargetBytes: 10000}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	assertNoWorkFiles(t, dir)
	files, _ := os.ReadDir(dir)
	if len(files) != 1 {
		t.Fatalf("left an output after cancellation: %v", files)
	}
}

func TestGrowingAndInvalidOutputsAreNotPublished(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{true: "invalid", false: "larger"}[invalid], func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "a.pdf")
			fakePDF(t, input, 30000)
			s := Service{Converter: converterFunc(func(_ context.Context, _, out string, _ Profile) error {
				if invalid {
					return os.WriteFile(out, []byte("not a pdf"), 0600)
				}
				fakePDF(t, out, 50000)
				return nil
			})}
			r, err := s.Compress(context.Background(), Request{Input: input, TargetBytes: 10000}, nil)
			if invalid && err == nil {
				t.Fatal("accepted invalid output")
			}
			if r.Output != "" || r.Bytes != 30000 {
				t.Fatalf("published a larger or invalid output: %+v", r)
			}
			assertNoWorkFiles(t, dir)
		})
	}
}

func assertNoWorkFiles(t *testing.T, dir string) {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(dir, ".mytoolbox-*"))
	if len(matches) != 0 {
		t.Fatalf("left temporary scan data behind: %v", matches)
	}
}
