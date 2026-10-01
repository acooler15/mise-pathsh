package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTemp 写入一个临时输入文件并返回其路径，供多文件参数用例使用。
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.sh")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestRunStdinSilentSuccess 验证无参数路径（stdin → stdout）与静默
// 成功纪律：退出码 0、stdout 为转换结果、stderr 必须为空
// （spec/go/logging-guidelines.md）。
func TestRunStdinSilentSuccess(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, strings.NewReader("export A='C:\\x'\n"), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if stdout.String() != "export A='/c/x'\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	// 静默成功：stderr 必须为空。
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty on success: %q", stderr.String())
	}
}

// TestRunFileArgsConcatenate 验证文件参数的 cat 语义：多个文件依次
// 读取，转换结果连续拼接到同一个 stdout。
func TestRunFileArgsConcatenate(t *testing.T) {
	f1 := writeTemp(t, "export A='C:\\x'\n")
	f2 := writeTemp(t, "export B='E:\\y'\n")
	var stdout, stderr bytes.Buffer
	code := run([]string{f1, f2}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if stdout.String() != "export A='/c/x'\nexport B='/e/y'\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty on success: %q", stderr.String())
	}
}

// TestRunUnreadableFileExits2 验证用法错误契约（退出码 2）：文件不可读
// 时 stdout 不得有部分输出，stderr 必须提及出错的文件名。
func TestRunUnreadableFileExits2(t *testing.T) {
	var stdout, stderr bytes.Buffer
	missing := filepath.Join(t.TempDir(), "does-not-exist.sh")
	code := run([]string{missing}, nil, &stdout, &stderr)
	if code != exitUsage {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), missing) {
		t.Fatalf("stderr %q does not mention the file", stderr.String())
	}
}

// TestRunDirectoryArgExits2 验证目录作为文件参数同样属于调用方用法
// 过错（打开失败 → 退出码 2），且诊断走 stderr。
func TestRunDirectoryArgExits2(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{t.TempDir()}, nil, &stdout, &stderr)
	if code != exitUsage {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stderr.Len() == 0 {
		t.Fatal("stderr empty for directory argument")
	}
}

// TestRunParseErrorExits1 验证解析失败契约（退出码 1，区别于用法
// 错误的 2），并断言诊断带 name:line:col 定位信息——解析库给出的
// 位置直接透传，不吞掉（spec/go/error-handling.md）。
func TestRunParseErrorExits1(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, strings.NewReader("if then\n"), &stdout, &stderr)
	if code != exitConvert {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout not empty: %q", stdout.String())
	}
	// 诊断必须带 name:line:col 定位信息。
	if !strings.Contains(stderr.String(), "stdin:1:1:") {
		t.Fatalf("stderr %q lacks line:col position", stderr.String())
	}
}

// TestRunParseErrorFileExits1 验证文件输入的解析错误必须提及文件名
// （而非笼统的 stdin），保证多文件场景下能定位到出错的输入。
func TestRunParseErrorFileExits1(t *testing.T) {
	f := writeTemp(t, "echo 'unterminated\n")
	var stdout, stderr bytes.Buffer
	code := run([]string{f}, nil, &stdout, &stderr)
	if code != exitConvert {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), f+":") {
		t.Fatalf("stderr %q does not mention the file name", stderr.String())
	}
}
