// converter.go 是编排层，位于整个数据流的中段：
//
//	stdin/文件 → []byte → Parse(KeepComments) → locate 定位候选区间
//	→ apply 按位置降序做字节级定点替换 → []byte → stdout
//
// 本文件不实现任何 bash 语法知识（解析交给 mvdan/sh/v3，候选区间
// 收集在 locate.go，转换规则在 rules.go），只负责把三者串起来。

package converter

import (
	"bytes"
	"fmt"
	"sort"

	"mvdan.cc/sh/v3/syntax"
)

// Convert 将 src 解析为 bash 脚本，把带引号字符串字面量内的 Windows
// 形态路径重写为 MSYS/Git-Bash 的 POSIX 形态，返回重写后的源码。
//
// AST 只用来"理解结构、定位候选"；重写本身是对命中片段的字节级定点
// 拼接，而非 mvdan printer 全文重打印——printer 会规范化缩进/引号/
// 换行，威胁"除目标片段外无 diff"的验收（design.md §2）。因此除
// 字面量内部之外一切逐字节保留：注释、引号风格、语句顺序原样不动，
// 输入输出的任何差异都可解释为"字符串字面量内部的变化"。
//
// 幂等（A2）：已转换的 POSIX 形态（/c/...）不匹配任何触发规则，
// 对输出再解析再转换一遍是 no-op，即 f(f(x)) == f(x)。
//
// 解析失败返回的错误消息携带输入名与行:列位置（如 `stdin:1:1: ...`），
// mvdan/sh 解析器给出的定位信息直接透传，不吞掉（spec/go/error-handling.md）。
func Convert(src []byte) ([]byte, error) {
	return ConvertNamed(src, "input")
}

// ConvertNamed 是 Convert 的显式命名版本：name 出现在诊断信息里
// （解析错误的 `name:line:col` 前缀），main 按输入来源传 "stdin"
// 或文件路径。
func ConvertNamed(src []byte, name string) ([]byte, error) {
	p := syntax.NewParser(syntax.KeepComments(true))
	f, err := p.Parse(bytes.NewReader(src), name)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	return apply(src, locate(f, src)), nil
}

// apply 把替换区间拼接回 src。各区间两两不相交（由 locate 保证），
// 按 start 降序应用：从右往左替换时，尚未应用的区间相对 src 前缀的
// 偏移不受影响；若按升序应用，前面的替换会移动后面区间的字节偏移，
// 使其失效（design.md §4）。
func apply(src []byte, reps []replacement) []byte {
	// 无触发即原样返回：保守原则下的幂等 no-op 快路径。
	if len(reps) == 0 {
		return src
	}
	sorted := make([]replacement, len(reps))
	// 复制后排序，不改写调用方传入的切片。
	copy(sorted, reps)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].start > sorted[j].start })

	out := src
	for _, r := range sorted {
		next := make([]byte, 0, len(out)-(r.end-r.start)+len(r.text))
		next = append(next, out[:r.start]...)
		next = append(next, r.text...)
		next = append(next, out[r.end:]...)
		out = next
	}
	return out
}
