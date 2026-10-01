// Package converter 把 mise 的 bash 集成输出（`mise activate --shell
// bash`、`mise hook-env --shell bash`）中字符串字面量内的 Windows
// 形态路径，转换为 Git-Bash / MSYS2 能正确解析的 POSIX 形态
// （`C:\foo` → `/c/foo`，PATH 列表 `;` → `:`）。
//
// 转换是保守的（宁漏勿错）：只有能被肯定地识别为 Windows 路径
// （盘符前缀）或其分号列表的带引号字面量片段才被重写；其余一切——
// base64 串、无引号裸值、已是 POSIX 的输入、UNC 路径——原样通过。
// 错转一个路径会让用户的 shell 直接坏掉，漏转只是少优化一处，所以
// 拿不准的一律不转。结构无损由字节级定点替换保证：转换只作用于
// 字面量内部，脚本结构逐字节保留。
package converter

import "strings"

// QuoteKind 标记字面量片段所处的引号上下文。它决定原始片段里的
// 反斜杠该如何解读——同一个 `C:\\x` 在单引号里是五个字面字符、
// 在双引号里却是转义对，解释错了就会把"看起来像路径的值"改错。
type QuoteKind int

const (
	// QuoteSingle 是 '...' 字面量的内容。单引号内没有任何转义序列，
	// 反斜杠就是字面反斜杠，可直接按 Windows 路径解读。
	QuoteSingle QuoteKind = iota
	// QuoteDouble 是 "..." 字符串内的 Lit 部分。mise 在这里把每个
	// 反斜杠写成 `\\` 转义对；任何其他形态的反斜杠使用都使片段的
	// 真实值无法确认，整个片段跳过不转（见 unescapeDouble）。
	QuoteDouble
)

// ConvertFragment 转换一个处于"原始源码形态"的带引号字面量片段，
// 是转换规则表（design.md §3，spec/go/path-conversion.md 规则表 v1）
// 的唯一实现：
//
//   - 规则 1：`;` 分隔的列表，且每个非空段都是盘符路径 → 各段独立
//     转换，`;` 连接改写为 `:`。
//   - 规则 2/7：单个盘符路径（反斜杠或正斜杠形态）→ POSIX 形态：
//     盘符小写、`\`→`/`；尾随分隔符在改写中自然保留（规则 6）。
//   - 规则 3：双引号内的 `\\` 转义对在路径改写中一并归一。
//   - 规则 4/5/8：展开段、base64 串、无引号裸值、POSIX 路径与 UNC
//     路径永不触发；ok 为 false，调用方必须让片段逐字节原样保留。
//
// 匹配只看"触发条件"，不建变量名白/黑名单：`__MISE_DIFF`/
// `__MISE_SESSION` 这类 base64 值不含 `\`、无盘符前缀，按构造就
// 不可能触发。这比按变量名过滤更保守，也对 mise 版本演化（新增、
// 更名变量）更鲁棒。
func ConvertFragment(raw string, q QuoteKind) (text string, ok bool) {
	switch q {
	case QuoteSingle:
		return convertValue(raw)
	case QuoteDouble:
		eff, ok := unescapeDouble(raw)
		if !ok {
			return "", false
		}
		return convertValue(eff)
	default:
		return "", false
	}
}

// unescapeDouble 把双引号字面量片段里的 `\\` 转义对折叠成单个反斜杠，
// 使片段能按普通 Windows 路径检查（否则 `C:\\x` 会因第二个字符不是
// 路径分隔符而不匹配盘符规则）。只接受"纯 `\\` 转义对"：出现奇数长
// 的 `\` 串（如 `\a`）或 `\$`、`\"` 这类其他转义时拒绝转换——那些
// 序列改变的是 bash 运行时的值本身而非拼写，我们不对它们建模，整个
// 片段跳过（宁漏勿错）。
//
// 返回的文本在转换后可以原样拼回双引号内、无需重新转义：DblQuoted
// 的 Lit 部分按语法不可能含 `"` 或反引号；奇数长 `\` 串已在上面拒绝；
// 转换出的 POSIX 路径不含 `$`，双引号内没有需要转义的字符。
func unescapeDouble(raw string) (string, bool) {
	if !strings.Contains(raw, `\`) {
		return raw, true
	}
	var b strings.Builder
	b.Grow(len(raw))
	for i := 0; i < len(raw); {
		if raw[i] != '\\' {
			b.WriteByte(raw[i])
			i++
			continue
		}
		n := 0
		for i+n < len(raw) && raw[i+n] == '\\' {
			n++
		}
		if n%2 != 0 {
			return "", false
		}
		for j := 0; j < n/2; j++ {
			b.WriteByte('\\')
		}
		i += n
	}
	return b.String(), true
}

// convertValue 对转义已归一的片段值应用触发规则（规则的唯一分派点，
// 顺序即优先级：先列表、后单路径）。凡不能被肯定地认定为 Windows
// 路径或其列表的值，一律不重写。
func convertValue(v string) (string, bool) {
	if !strings.Contains(v, ";") {
		if !isDrivePath(v) {
			return "", false
		}
		return convertPath(v), true
	}
	// 规则 1：要求每个非空段都是盘符路径，一段不匹配即整体放弃——
	// 防止把恰好含 `;` 的非路径值（如 `--flag=a;b`）误转。这是
	// "宁漏勿错"在列表场景的具体化：全部转换或全部不转，不做部分转换。
	segs := strings.Split(v, ";")
	out := make([]string, len(segs))
	converted := false
	for i, seg := range segs {
		if seg == "" {
			continue // 空段原样传递，保持 `C:\a;;E:\b` 的空段结构
		}
		if !isDrivePath(seg) {
			return "", false
		}
		out[i] = convertPath(seg)
		converted = true
	}
	if !converted {
		return "", false
	}
	return strings.Join(out, ":"), true
}

// isDrivePath 判断 p 是否为带盘符前缀的单个 Windows 路径：
// ^[A-Za-z]:[\\/]。正斜杠形态 `X:/...` 同样接受（规则 7，mise 在
// 部分输出里也用这种形态）。其余部分可含空格（`C:\Program
// Files\...`），空格不影响匹配；`C:foo`、`C:` 这类缺分隔符的盘符
// 相对路径被排除——MSYS 无法确认其指向，不转。
func isDrivePath(p string) bool {
	if len(p) < 3 {
		return false
	}
	if !isDriveLetter(p[0]) || p[1] != ':' {
		return false
	}
	return p[2] == '\\' || p[2] == '/'
}

// isDriveLetter 判断字节是否为 ASCII 盘符字母（大小写均可）。
func isDriveLetter(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// convertPath 把单个盘符路径改写为 MSYS/Git-Bash 的 POSIX 形态：
// 盘符小写（`/c/...` 是 MSYS 惯例，语义对齐 cygpath 实测，黄金值见
// research/cygpath-oracle.md）、`\`→`/`。`X:` + 分隔符映射为 `/x/`，
// 因此 `C:\` → `/c/`，尾随分隔符按构造保留（规则 6 无需特判）。
//
// 对 cygpath 的有意偏差：不做 MSYS 挂载表规范形——Git 安装根下的
// 路径输出 `/c/Program Files/Git/cmd` 而非 `/cmd`。二者在 MSYS
// 命名空间解析到同一位置，功能等价；挂载感知需要运行时查询 MSYS
// runtime，而 hook-env 每个提示符都会触发，Windows 进程 spawn 的
// 延迟（数十毫秒/次）不可接受，v1 保持零外部依赖（design.md §3.1）。
// 若集成实测（A3）暴露盘符形的实际缺陷，再启用该节预留的
// "启动时一次性根查询 + 前缀规范化"模式。
func convertPath(p string) string {
	drive := p[0]
	if drive >= 'A' && drive <= 'Z' {
		drive += 'a' - 'A'
	}
	rest := strings.ReplaceAll(p[2:], `\`, "/")
	return "/" + string(drive) + rest
}
