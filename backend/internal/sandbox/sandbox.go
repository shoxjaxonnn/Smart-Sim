package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Result struct {
	Passed     bool   `json:"passed"`
	TimedOut   bool   `json:"timed_out"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

func RunPython(ctx context.Context, studentCode, tests string) (Result, error) {
	var res Result
	start := time.Now()

	pythonPath, err := findPython()
	if err != nil {
		return res, err
	}

	dir, err := os.MkdirTemp("", "smartedu-sandbox-*")
	if err != nil {
		return res, err
	}
	defer os.RemoveAll(dir)

	joined := strings.TrimSpace(studentCode) + "\n\n" + strings.TrimSpace(tests) + "\n"
	scriptPath := filepath.Join(dir, "sandbox.py")
	if err := os.WriteFile(scriptPath, []byte(joined), 0600); err != nil {
		return res, err
	}

	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(runCtx, pythonPath, "-I", "-B", scriptPath)
	cmd.Dir = dir
	cmd.Env = []string{
		"PYTHONIOENCODING=utf-8",
		"PYTHONUNBUFFERED=1",
		"PYTHONNOUSERSITE=1",
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	res.DurationMs = time.Since(start).Milliseconds()
	res.Stdout = stdout.String()
	res.Stderr = stderr.String()

	if runCtx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.Error = "timeout"
		return res, nil
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.Error = err.Error()
			return res, nil
		}
	} else {
		res.ExitCode = 0
		res.Passed = true
	}

	if res.ExitCode != 0 {
		res.Passed = false
	}
	return res, nil
}

func findPython() (string, error) {
	if p := os.Getenv("PYTHON_EXECUTABLE"); p != "" {
		return p, nil
	}
	candidates := []string{"python", "python3", "py"}
	if runtime.GOOS == "windows" {
		candidates = []string{"python", "py"}
	}
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("python executable not found")
}
