// Command mise-pathsh 把 mise 的 bash 集成输出转换为 Git-Bash / MSYS2
// 能正确解析的 POSIX 路径形态（C:\foo → /c/foo，PATH 列表 ; → :），
// 使 Windows 下
//
//	eval "$(mise activate --shell bash | mise-pathsh)"
//
// 得到冒号分隔、可直接工作的 PATH。
//
// 接口形态：cat 风格过滤器，无子命令（design.md §1 / prd.md D1）。
//
//	mise-pathsh              无参数时读 stdin，转换结果写 stdout
//	mise-pathsh FILE...      依次读每个文件，转换结果连续写 stdout（cat 语义）
//
// 输出纪律（spec/go/logging-guidelines.md）：stdout 只承载产品数据
// （转换后的脚本）——它会被重定向进管道或 eval，掺入任何非结果内容
// 都是缺陷；诊断信息（错误、定位）一律走 stderr；转换成功时保持静默。
//
// 退出码契约（spec/go/error-handling.md）：0 成功；1 转换/解析失败；
// 2 用法错误（输入文件不可读）。
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"mise-pathsh/internal/converter"
)

// progName 用作 stderr 诊断信息的行前缀。
const progName = "mise-pathsh"

// 三种退出码与契约一一对应：转换层错误 → 1；调用方用法过错 → 2。
// 退出决策集中在 main 包，internal/ 库代码只返回 error，不做退出。
const (
	exitOK      = 0
	exitConvert = 1
	exitUsage   = 2
)

// usageError 标记"调用方过错"类输入问题（文件不可读、stdin 不可读），
// report 据此映射到退出码 2；解析失败等转换类错误不包这层，走退出
// 码 1。实现 Error/Unwrap 仅为配合 errors.As 做类型判别，错误根因链
// 通过 Unwrap 原样保留。
type usageError struct{ err error }

func (e *usageError) Error() string { return e.err.Error() }
func (e *usageError) Unwrap() error { return e.err }

// main 只做接线：把真实 stdin/stdout/stderr 注入 run，并用其返回值
// 决定进程退出码。
func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run 执行过滤器并返回进程退出码。错误快速失败：遇到第一个错误即
// 停止，此前已写出的转换结果保持不动（对 cat 风格多文件输入是可接受
// 的截断语义）。依赖以 io.Reader/io.Writer 注入而非直接用 os.Stdxxx，
// 使本函数可以脱离真实进程 IO 做测试。
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		if err := convertStream("stdin", stdin, stdout); err != nil {
			return report(stderr, err)
		}
		return exitOK
	}
	for _, path := range args {
		if err := convertFile(path, stdout); err != nil {
			return report(stderr, err)
		}
	}
	return exitOK
}

// convertFile 打开单个输入文件并转换。打开失败属于调用方用法过错
// （包 usageError → 退出码 2），与解析/转换失败（退出码 1）区分开。
func convertFile(path string, stdout io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return &usageError{err}
	}
	defer f.Close()
	return convertStream(path, f, stdout)
}

// convertStream 读取一个输入源并转换写入 stdout。name 只用于诊断
// （解析错误前缀）；converter 返回的解析错误已携带 name 与 line:col
// 定位信息，这里不再二次包装，直接上抛给 report，避免吞掉定位。
func convertStream(name string, r io.Reader, stdout io.Writer) error {
	src, err := io.ReadAll(r)
	if err != nil {
		return &usageError{fmt.Errorf("reading %s: %w", name, err)}
	}
	out, err := converter.ConvertNamed(src, name)
	if err != nil {
		return err // 解析错误已带 name 与 line:col 定位信息
	}
	if _, err := stdout.Write(out); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}

// report 把错误打印到 stderr 并返回对应退出码：usageError → 2，
// 其余（解析/转换失败）→ 1。全部诊断输出集中于此，converter 内部
// 不打印任何东西（spec/go/logging-guidelines.md）。
func report(stderr io.Writer, err error) int {
	code := exitConvert
	var ue *usageError
	if errors.As(err, &ue) {
		code = exitUsage
	}
	fmt.Fprintf(stderr, "%s: %v\n", progName, err)
	return code
}
