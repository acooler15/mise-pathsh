// locate.go 是定位层：遍历已解析的 AST，从语法结构中找出"哪些字节
// 区间待替换"，产出 replacement 列表交给 converter.go 的 apply 做定点
// 拼接。本文件不实现任何转换规则（规则表在 rules.go），也不做替换
// 本身——只回答"改哪里"，"改成什么"由规则层决定。

package converter

import "mvdan.cc/sh/v3/syntax"

// replacement 表示一次定点替换：把 src[start:end] 换成 text。
// start/end 是原始源码的字节偏移；各区间两两不相交（scanFragment
// 只记录整个字面量片段的区间，不存在嵌套或部分重叠），这是 apply
// 能按位置降序安全拼接的前提。
type replacement struct {
	start int
	end   int
	text  []byte
}

// locate 自身不做解析；它遍历 src 已解析好的 AST，按规则表返回需要
// 的字节级 replacement 列表。
//
// 候选上下文只有两类：赋值语句的值（含数组元素）与命令字参数——
// 这正是 mise 的 bash 输出实际出现路径的位置（export PATH=...、
// export CARGO_HOME=...、command '...mise.exe' ...）。其余位置
// （case 模式、[[ ]] 测试表达式、heredoc 正文、无引号裸词）一律不
// 访问：mise 输出不在这些位置放 Windows 路径，少扫一层就少一分
// 错转风险（宁漏勿错）。候选 word 内部只考虑带引号的字面量部分；
// 展开段（$VAR、$(...)、反引号）由 AST 的 Parts 拆分天然隔离成独立
// 节点，不会被触碰（规则 4）。
//
// 只转换 '...' 与 "..." 两种引号上下文：无引号裸字面量里 `\` 是
// bash 的转义符（bash 读到的 `C:\x` 值已是 `C:x`），改写它等于改值
// 而非改拼写——mise 输出的路径总是带引号，所以不因此漏转。
// ANSI-C 引号 $'...' 与 locale 引号 $"..." 基于同样的保守理由跳过。
func locate(f *syntax.File, src []byte) []replacement {
	var reps []replacement
	add := func(r replacement) { reps = append(reps, r) }

	syntax.Walk(f, func(n syntax.Node) bool {
		switch v := n.(type) {
		case *syntax.Assign:
			if v.Value != nil {
				scanWord(v.Value, src, add)
			}
			if v.Array != nil {
				for _, el := range v.Array.Elems {
					if el.Value != nil {
						scanWord(el.Value, src, add)
					}
				}
			}
		case *syntax.CallExpr:
			for _, w := range v.Args {
				scanWord(w, src, add)
			}
		}
		return true
	})
	return reps
}

// scanWord 检查一个候选 word 的各个 Parts。export/declare/local 的
// 参数解析为 Assign 节点（由 locate 的 *syntax.Assign 分支访问），
// 普通命令参数解析为 CallExpr.Args；两条入口在此汇合。SglQuoted 的
// 内容是一个整体区间；DblQuoted 则逐个处理其 Lit 子节点——展开段
// 不是 Lit，被类型断言自然滤掉。
func scanWord(w *syntax.Word, src []byte, add func(replacement)) {
	for _, part := range w.Parts {
		switch q := part.(type) {
		case *syntax.SglQuoted:
			if q.Dollar {
				continue // $'...'：ANSI-C 转义，超出保守范围
			}
			start := int(q.Pos().Offset()) + 1 // 跳过开引号
			end := int(q.End().Offset()) - 1   // 跳过闭引号
			scanFragment(string(src[start:end]), QuoteSingle, start, add)
		case *syntax.DblQuoted:
			if q.Dollar {
				continue // $"..."：locale 字符串，超出保守范围
			}
			for _, sub := range q.Parts {
				lit, ok := sub.(*syntax.Lit)
				if !ok {
					continue // $VAR、$(...)、反引号展开：规则 4，不动
				}
				start := int(lit.Pos().Offset())
				end := int(lit.End().Offset())
				scanFragment(string(src[start:end]), QuoteDouble, start, add)
			}
		}
	}
}

// scanFragment 对一个原始字面量片段跑规则表：命中则记录覆盖该片段
// 源码区间的 replacement；未命中（ok=false）时不产生任何替换，片段
// 在输出中逐字节原样保留。
func scanFragment(raw string, kind QuoteKind, start int, add func(replacement)) {
	text, ok := ConvertFragment(raw, kind)
	if !ok {
		return
	}
	add(replacement{start: start, end: start + len(raw), text: []byte(text)})
}
