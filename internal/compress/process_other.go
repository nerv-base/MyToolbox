//go:build !windows

package compress

import "os/exec"

func configureProcess(*exec.Cmd) {}
