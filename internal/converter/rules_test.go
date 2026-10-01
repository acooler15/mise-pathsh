package converter

import "testing"

// 黄金值取自 research/cygpath-oracle.md（cygwin 3.6.10 的
// `cygpath -u` / `-up` 实测，2026-10-01），作为测试 oracle 逐段断言，
// 保证转换语义对齐 cygpath 而非自说自话。
const (
	goldenSingle = `C:\Users\tester\bin;E:\develop\mise\shims;C:\Program Files\dotnet`
	goldenList   = `/c/Users/tester/bin:/e/develop/mise/shims:/c/Program Files/dotnet`
)

// TestConvertFragmentSingleQuoted 覆盖单引号上下文下的规则表正反例
// （验收 A4：每条规则至少一个正例与反例，含带空格路径、base64、
// 裸值）。want 为空表示不应触发任何规则。
func TestConvertFragmentSingleQuoted(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string // "" 表示：无规则触发，片段原样保留
	}{
		// 规则 2：单个反斜杠形态 Windows 路径。
		{"drive path", `C:\Users\tester\bin`, "/c/Users/tester/bin"},
		{"oracle golden single", `C:\Users\tester\bin`, "/c/Users/tester/bin"},
		{"lowercase drive input", `e:\develop\mise\shims`, "/e/develop/mise/shims"},
		{"exe argument", `C:\Users\tester\scoop\apps\mise\current\bin\mise.exe`,
			"/c/Users/tester/scoop/apps/mise/current/bin/mise.exe"},
		{"spaces preserved", `C:\Program Files\dotnet`, "/c/Program Files/dotnet"},
		{"parens preserved", `C:\Program Files (x86)\NVIDIA Corporation\PhysX\Common`,
			"/c/Program Files (x86)/NVIDIA Corporation/PhysX/Common"},
		{"drive root", `C:\`, "/c/"},
		// 规则 6：尾随分隔符保留为尾随 `/`。
		{"trailing backslash", `C:\Users\tester\bin\`, "/c/Users/tester/bin/"},
		// 规则 7：正斜杠 Windows 形态同样接受。
		{"forward slash form", `C:/Users/tester/bin`, "/c/Users/tester/bin"},
		{"forward slash root", `E:/develop`, "/e/develop"},
		// 规则 1：`;` 分隔列表，且每段都必须是盘符路径。
		{"two segment list", `C:\a;E:\b`, "/c/a:/e/b"},
		{"oracle golden list", goldenSingle, goldenList},
		{"trailing empty segment", `C:\a;E:\b;`, "/c/a:/e/b:"},
		{"inner empty segment", `C:\a;;E:\b`, "/c/a::/e/b"},
		{"uppercase drive in list", `C:\a;e:\b`, "/c/a:/e/b"},
		// 空格不影响匹配与替换（design.md §3）；只有出现非盘符前缀的段
		// 才使整个列表放弃转换。
		{"trailing space in segment preserved", `C:\a ;E:\b`, "/c/a :/e/b"},

		// 反例集：一律不得触发（保守原则，宁漏勿错）——UNC、裸盘符、
		// base64、裸值、含非路径段的列表等。
		{"posix passthrough", `/already/posix`, ""},
		{"relative posix", `bin/tools`, ""},
		{"unc path skipped", `\\server\share\dir`, ""},
		{"unc list segment skipped", `\\srv\share;C:\a`, ""},
		{"drive relative skipped", `C:foo`, ""},
		{"bare drive skipped", `C:`, ""},
		{"bare value skipped", `stable`, ""},
		{"base64 skipped", `eAFqXpyfk9KwOC+1vH2Vs2OQu3+8h7+v6w5nq5jQ4tSi4pjgzLzknMTMohi95MSi9Pyl7v5Onn431VytYlJSy1Jz8gticjOLU2My84pLEnNyimPS82MM9YzM9QxjkjLzlrn7B/n7h9xUIqx8pZdjmCPY7pt6OFVnJZYlxlSV5pTqGprrmVnoGRkY6xmsDgoNDgkNAOvdicXdRaXFJaUFG6CqQvz9fZw9HD39lhWXJCblpK4MDYsPiAzx8Pe7aYjT3oLKkoz8vBhjPUMTPfMYCE8vtSJ1SUFiScYcYkPjpg5OC5JzEktTUnWT81NSY4z0DPWMLAxuGuFUjTUYQOF9UxWnnjyw0SZ6RgZ6Bjc9cCsryNXNzcxLScvPScko1C0pSs3JySyOMdAz0zM0jwEZEp+bn1Kak1oco5eUmYcnZgEryIwx0LMw1zPEEzyQkISGK554R1EXE5xclFlQUrwHS1yD0yg4JHBHZnFhTmZJaoyxnqmxnkmMHij56iZl5hXfVMYZKqVlMQZ6hkZ6RoY3NXAqqkwsyosx0TO00DMAuQAAZglBgQ`, ""},
		{"flag with semicolon skipped", `--flag=a;b`, ""},
		{"one bad segment disables list", `C:\a;b`, ""},
		{"leading space segment disables list", `C:\a; E:\b`, ""},
		{"mixed posix segment disables list", `/c/a;E:\b`, ""},
		{"windows path not at start", `pre C:\x`, ""},
		{"empty fragment", ``, ""},
		{"colon before drive", `:C:\x`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ConvertFragment(tt.in, QuoteSingle)
			if tt.want == "" {
				if ok {
					t.Fatalf("ConvertFragment(%q) = %q, want untouched", tt.in, got)
				}
				return
			}
			if !ok {
				t.Fatalf("ConvertFragment(%q) not triggered, want %q", tt.in, tt.want)
			}
			if got != tt.want {
				t.Fatalf("ConvertFragment(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestConvertFragmentDoubleQuoted 覆盖双引号上下文（规则 3）：`\\`
// 转义对归一后按单引号规则转换；反斜杠用法无法确认为纯转义对的
// 片段（奇数长 `\` 串、`\$`、`\"`）必须原样跳过。
func TestConvertFragmentDoubleQuoted(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		// 规则 3：`\\` 转义对在改写中一并归一。
		{"doubled backslash path", `C:\\Users\\x`, "/c/Users/x"},
		{"mise PATH prefix form", `C:\\Users\\tester\\scoop\\apps\\mise\\current\\bin:`,
			"/c/Users/tester/scoop/apps/mise/current/bin:"},
		{"trailing doubled separator", `C:\\Users\\x\\`, "/c/Users/x/"},
		{"doubled list", `C:\\a;E:\\b`, "/c/a:/e/b"},
		{"forward slash form", `C:/Users/x`, "/c/Users/x"},
		{"no backslash no trigger", `plain-value`, ""},

		// 反例：反斜杠用法无法确认为纯 `\\` 转义对的片段一律不动。
		{"lone backslash skipped", `C:\Users\x`, ""},
		{"odd run skipped", `C:\\x\y`, ""},
		{"dollar escape skipped", `C:\\x\$y`, ""},
		{"escaped quote skipped", `C:\\x\"y`, ""},
		{"trailing lone backslash skipped", `C:\\x\`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ConvertFragment(tt.in, QuoteDouble)
			if tt.want == "" {
				if ok {
					t.Fatalf("ConvertFragment(%q, Double) = %q, want untouched", tt.in, got)
				}
				return
			}
			if !ok {
				t.Fatalf("ConvertFragment(%q, Double) not triggered, want %q", tt.in, tt.want)
			}
			if got != tt.want {
				t.Fatalf("ConvertFragment(%q, Double) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestConvertFragmentIdempotent 在规则层钉住 A2 幂等性质：转换结果
// 是 POSIX 形态，不匹配任何触发规则，因此对输出再转换必然 no-op。
// 改动规则表时不得破坏本测试。
func TestConvertFragmentIdempotent(t *testing.T) {
	// 单引号幂等：转换后的片段不再触发任何规则。
	for _, in := range []string{
		`C:\Users\tester\bin`,
		`C:\a;E:\b;C:\Program Files\dotnet`,
	} {
		once, ok := ConvertFragment(in, QuoteSingle)
		if !ok {
			t.Fatalf("ConvertFragment(%q, Single) not triggered", in)
		}
		if twice, ok := ConvertFragment(once, QuoteSingle); ok {
			t.Fatalf("re-convert of %q triggered again: %q", once, twice)
		}
	}
	// mise 转义形态下的双引号幂等：POSIX 输出不得再次触发规则。
	in := `C:\\Users\\tester\\scoop\\apps\\mise\\current\\bin:`
	once, ok := ConvertFragment(in, QuoteDouble)
	if !ok || once != "/c/Users/tester/scoop/apps/mise/current/bin:" {
		t.Fatalf("ConvertFragment(%q, Double) = %q (ok=%v)", in, once, ok)
	}
	if twice, ok := ConvertFragment(once, QuoteDouble); ok {
		t.Fatalf("re-convert of %q triggered again: %q", once, twice)
	}
}

// TestConvertPathDeviationFromCygpath 钉住已文档化的 v1 有意偏差
// （design.md §3.1）：不做 MSYS 挂载表规范形，Git 安装根下的路径
// 输出盘符形——与 cygpath 解析到同一位置，功能等价。
func TestConvertPathDeviationFromCygpath(t *testing.T) {
	got, ok := ConvertFragment(`C:\Program Files\Git\cmd`, QuoteSingle)
	if !ok || got != "/c/Program Files/Git/cmd" {
		t.Fatalf("Git-root path = %q (ok=%v), want /c/Program Files/Git/cmd", got, ok)
	}
}
