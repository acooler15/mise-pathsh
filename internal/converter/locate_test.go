package converter

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"mvdan.cc/sh/v3/syntax"
)

// parseForTest 解析 src 并返回 locate 找到的替换区间（测试辅助）。
func parseForTest(t *testing.T, src string) []replacement {
	t.Helper()
	f, err := syntax.NewParser(syntax.KeepComments(true)).Parse(strings.NewReader(src), "test")
	if err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}
	return locate(f, []byte(src))
}

// applyReps 按位置降序把替换拼回 src，镜像 converter.go 的应用策略，
// 使本文件能独立验证"区间 + 拼接 = 期望输出"，不必经过 Convert 全流水线。
func applyReps(t *testing.T, src string, reps []replacement) string {
	t.Helper()
	sorted := make([]replacement, len(reps))
	copy(sorted, reps)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].start > sorted[j].start })
	out := []byte(src)
	for _, r := range sorted {
		next := make([]byte, 0, len(out)-(r.end-r.start)+len(r.text))
		next = append(next, out[:r.start]...)
		next = append(next, r.text...)
		next = append(next, out[r.end:]...)
		out = next
	}
	return string(out)
}

// TestLocateRangesAndContexts 覆盖 locate 的候选上下文与排除项：赋值
// 值、数组元素、命令字参数（含嵌套在命令替换里的）命中；case 模式、
// 测试表达式、heredoc、$'...'/$"..."、无引号裸值等排除。每例同时断言
// 区间数量与拼接后的完整输出。
func TestLocateRangesAndContexts(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantReps int
		wantOut  string // 应用替换后的期望完整输出
	}{
		{
			name:     "single-quoted assign value",
			src:      "export __MISE_EXE='C:\\x\\mise.exe'\n",
			wantReps: 1,
			wantOut:  "export __MISE_EXE='/c/x/mise.exe'\n",
		},
		{
			name:     "double-quoted assign value keeps quotes and expansion",
			src:      "export PATH=\"C:\\\\x\\\\bin:$PATH\"\n",
			wantReps: 1,
			wantOut:  "export PATH=\"/c/x/bin:$PATH\"\n",
		},
		{
			name:     "command word argument",
			src:      "command 'C:\\x\\mise.exe' \"$@\"\n",
			wantReps: 1,
			wantOut:  "command '/c/x/mise.exe' \"$@\"\n",
		},
		{
			name:     "two assigns in one declaration",
			src:      "export A='C:\\x' B='E:\\y'\n",
			wantReps: 2,
			wantOut:  "export A='/c/x' B='/e/y'\n",
		},
		{
			name:     "array elements",
			src:      "M=('/C:\\x' \"E:\\\\y\")\n",
			wantReps: 1, // '/C:\x' 无盘符前缀不触发；"E:\\y" 正常转换
			wantOut:  "M=('/C:\\x' \"/e/y\")\n",
		},
		{
			name:     "prefix assignment",
			src:      "FOO='C:\\x' cmd arg\n",
			wantReps: 1,
			wantOut:  "FOO='/c/x' cmd arg\n",
		},
		{
			name:     "nested call inside command substitution",
			src:      "eval \"$(command 'C:\\x\\mise.exe' \"$@\")\"\n",
			wantReps: 1,
			wantOut:  "eval \"$(command '/c/x/mise.exe' \"$@\")\"\n",
		},
		{
			name:     "unquoted value skipped",
			src:      "export FOO=C:\\bar\n",
			wantReps: 0,
			wantOut:  "export FOO=C:\\bar\n",
		},
		{
			// 规则 4（AST Parts 隔离）：展开段本身永不重写；其后字面量
			// 列表尾部按规则机械转换（每个非空段都是盘符路径）。
			name:     "expansion untouched, literal list tail converts",
			src:      "export A=\"$HOME;C:\\\\x\"\n",
			wantReps: 1,
			wantOut:  "export A=\"$HOME:/c/x\"\n",
		},
		{
			name:     "expansion after converted literal",
			src:      "export A=\"C:\\\\x:${HOME}\"\n",
			wantReps: 1,
			wantOut:  "export A=\"/c/x:${HOME}\"\n",
		},
		{
			name:     "test clause untouched",
			src:      "[[ \"C:\\x\" == a ]]\n",
			wantReps: 0,
			wantOut:  "[[ \"C:\\x\" == a ]]\n",
		},
		{
			name:     "case pattern untouched",
			src:      "case $x in 'C:\\x') echo hi;; esac\n",
			wantReps: 0,
			wantOut:  "case $x in 'C:\\x') echo hi;; esac\n",
		},
		{
			name:     "heredoc body untouched",
			src:      "cat <<EOF\nC:\\x\nEOF\n",
			wantReps: 0,
			wantOut:  "cat <<EOF\nC:\\x\nEOF\n",
		},
		{
			name:     "ANSI-C and locale quotes skipped",
			src:      "A=$'C:\\\\x' B=$\"C:\\\\y\"\n",
			wantReps: 0,
			wantOut:  "A=$'C:\\\\x' B=$\"C:\\\\y\"\n",
		},
		{
			name:     "comments untouched",
			src:      "# path C:\\x stays\nexport A='C:\\x'\n",
			wantReps: 1,
			wantOut:  "# path C:\\x stays\nexport A='/c/x'\n",
		},
		{
			name:     "no trigger no replacement",
			src:      "export T=stable\nexport D=eAFkbase64==\n",
			wantReps: 0,
			wantOut:  "export T=stable\nexport D=eAFkbase64==\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reps := parseForTest(t, tt.src)
			if len(reps) != tt.wantReps {
				t.Fatalf("locate(%q) found %d replacements (%v), want %d",
					tt.src, len(reps), formatReps(reps), tt.wantReps)
			}
			if got := applyReps(t, tt.src, reps); got != tt.wantOut {
				t.Fatalf("apply(locate(%q)) =\n  %q\nwant\n  %q", tt.src, got, tt.wantOut)
			}
		})
	}
}

func formatReps(reps []replacement) string {
	var sb strings.Builder
	for i, r := range reps {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "[%d:%d]=%q", r.start, r.end, r.text)
	}
	return sb.String()
}

// TestLocateReplacementRangeInsideQuotes 断言替换区间严格落在引号字符
// 之间：引号风格本身永远不属于替换范围——这是"引号风格保留、任何
// diff 都可解释为字面量内部变化"（A5）的直接保证。
func TestLocateReplacementRangeInsideQuotes(t *testing.T) {
	src := "export A='C:\\x'\n"
	reps := parseForTest(t, src)
	if len(reps) != 1 {
		t.Fatalf("want 1 replacement, got %d", len(reps))
	}
	r := reps[0]
	if src[r.start] == '\'' || src[r.end-1] == '\'' {
		t.Fatalf("replacement range %d:%d includes quote characters: %q", r.start, r.end, src[r.start:r.end])
	}
	if got := src[r.start:r.end]; got != `C:\x` {
		t.Fatalf("replacement range covers %q, want %q", got, `C:\x`)
	}
}

// TestLocateDisjointRanges 断言区间两两不相交——这正是 apply 能按
// 位置降序做定点拼接的良定义前提（design.md §4）。
func TestLocateDisjointRanges(t *testing.T) {
	src := "export A='C:\\x' B=\"E:\\\\y:$PATH\" C='C:\\z'\n"
	reps := parseForTest(t, src)
	if len(reps) != 3 {
		t.Fatalf("want 3 replacements, got %d (%s)", len(reps), formatReps(reps))
	}
	sorted := make([]replacement, len(reps))
	copy(sorted, reps)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].start < sorted[j].start })
	for i := 1; i < len(sorted); i++ {
		if sorted[i].start < sorted[i-1].end {
			t.Fatalf("overlapping ranges: %s", formatReps(reps))
		}
	}
}
