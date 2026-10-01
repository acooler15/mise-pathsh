package converter

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"mvdan.cc/sh/v3/syntax"
)

// readTestdata 读取 testdata/ 下的 fixture。原始样本
// sample-activate.sh 与 hook-env 样本只读，测试一律使用副本
// （spec/go/directory-structure.md）。
func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("read testdata/%s: %v", name, err)
	}
	return b
}

// remainingWinDrive 匹配 `盘符:反斜杠` 序列；转换后的输出任何位置都
// 不允许残留。（不能用 `盘符:正斜杠` 匹配——会在 `/c/x/bin:/c/...`
// 这类 POSIX 列表边界上误报。）
var remainingWinDrive = regexp.MustCompile(`[A-Za-z]:\\`)

// assertNoBackslashInQuotedLiterals 重新解析转换输出，断言所有带引号
// 字面量都不再含反斜杠：在本 fixture 中带引号的反斜杠只可能是
// Windows 路径内容，`\typeset` 这类转义只出现在无引号位置。
func assertNoBackslashInQuotedLiterals(t *testing.T, out []byte) {
	t.Helper()
	f, err := syntax.NewParser(syntax.KeepComments(true)).Parse(strings.NewReader(string(out)), "out")
	if err != nil {
		t.Fatalf("converted output does not re-parse: %v", err)
	}
	syntax.Walk(f, func(n syntax.Node) bool {
		switch v := n.(type) {
		case *syntax.SglQuoted:
			if !v.Dollar && strings.Contains(v.Value, `\`) {
				t.Errorf("quoted literal still contains backslash: %q", v.Value)
			}
		case *syntax.DblQuoted:
			if v.Dollar {
				return true
			}
			for _, sub := range v.Parts {
				if lit, ok := sub.(*syntax.Lit); ok && strings.Contains(lit.Value, `\`) {
					t.Errorf("double-quoted literal still contains backslash: %q", lit.Value)
				}
			}
		}
		return true
	})
}

// assertFullyConverted 组合两项全量转换检查：输出无残留 `X:\` 序列、
// 引号字面量内无反斜杠。
func assertFullyConverted(t *testing.T, out []byte) {
	t.Helper()
	if remainingWinDrive.Match(out) {
		t.Errorf("converted output still contains a X:\\ drive path sequence")
	}
	assertNoBackslashInQuotedLiterals(t, out)
}

// TestConvertSnippet 用最小片段覆盖 Convert 编排：赋值 + 命令字面量、
// 双引号含展开段，以及幂等 no-op（已 POSIX 的输入原样通过）。
func TestConvertSnippet(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "assignment and command literal",
			in:   "export A='C:\\x'\ncommand 'C:\\x\\mise.exe' \"$@\"\n",
			want: "export A='/c/x'\ncommand '/c/x/mise.exe' \"$@\"\n",
		},
		{
			name: "double quote with expansion",
			in:   "export PATH=\"C:\\\\x\\\\bin:$PATH\"\n",
			want: "export PATH=\"/c/x/bin:$PATH\"\n",
		},
		{
			name: "already posix is a no-op",
			in:   "export PATH='/c/x/bin:$PATH'\n",
			want: "export PATH='/c/x/bin:$PATH'\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Convert([]byte(tt.in))
			if err != nil {
				t.Fatalf("Convert: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("Convert:\n  got  %q\n  want %q", got, tt.want)
			}
		})
	}
}

// TestConvertEmptyInput 验证空输入（nil、空串、单换行、仅注释）不报错
// 且原样通过——空脚本没有候选区间，属保守原则的自然结果。
func TestConvertEmptyInput(t *testing.T) {
	for _, in := range [][]byte{nil, {}, []byte(""), []byte("\n"), []byte("# only a comment\n")} {
		out, err := Convert(in)
		if err != nil {
			t.Fatalf("Convert(%q) error: %v", in, err)
		}
		if string(out) != string(in) {
			t.Fatalf("Convert(%q) = %q, want unchanged", in, out)
		}
	}
}

// TestConvertParseErrorCarriesPosition 验证解析错误携带 name:line:col
// 定位信息——mvdan/sh 的位置信息直接透传，不吞掉
// （spec/go/error-handling.md）。
func TestConvertParseErrorCarriesPosition(t *testing.T) {
	_, err := ConvertNamed([]byte("if then\n"), "stdin")
	if err == nil {
		t.Fatal("want parse error, got nil")
	}
	if !strings.Contains(err.Error(), "stdin:1:1:") {
		t.Fatalf("error %q does not carry name and line:col position", err)
	}
}

// expectedRewrite 仅凭规则层（其黄金值独立取自 cygpath 实测）重构一行
// fixture 的期望转换结果——刻意不经过 locate/apply，使全文件流水线的
// 断言对象是"规则层"而非"流水线自身"，避免自我印证。原始片段从行内
// 按结构提取；行尾 \r（activate fixture 混用 CRLF/LF 结尾）原样保留。
func expectedRewrite(t *testing.T, line string) string {
	t.Helper()
	core, cr := strings.TrimSuffix(line, "\r"), strings.HasSuffix(line, "\r")

	// command '...\mise.exe' / eval "$(command '...\mise.exe' ...)"：
	// mise 可执行路径以两种引号风格出现，但始终是字面单反斜杠。
	const miseExe = `C:\Users\tester\scoop\apps\mise\current\bin\mise.exe`
	if conv, ok := ConvertFragment(miseExe, QuoteSingle); ok {
		core = strings.ReplaceAll(core, miseExe, conv)
	}
	// 整值单引号赋值：export VAR='...'
	if i := strings.Index(core, "='"); i >= 0 && strings.HasSuffix(core, "'") {
		raw := core[i+2 : len(core)-1]
		if conv, ok := ConvertFragment(raw, QuoteSingle); ok {
			core = strings.Replace(core, "'"+raw+"'", "'"+conv+"'", 1)
		}
	}
	// 双引号 PATH 前缀：export PATH="...:$PATH"
	if strings.HasPrefix(strings.TrimSpace(core), `export PATH="`) && strings.HasSuffix(core, `:$PATH"`) {
		start := strings.Index(core, `"`) + 1
		end := strings.LastIndex(core, `:$PATH"`)
		raw := core[start:end]
		if conv, ok := ConvertFragment(raw, QuoteDouble); ok {
			core = strings.Replace(core, `"`+raw+`:`, `"`+conv+`:`, 1)
		}
	}
	if cr {
		core += "\r"
	}
	return core
}

// assertStructuralDiff 逐行实现 A5 结构无损检查：未变化的行必须逐字节
// 相同（钉住注释、引号风格、缩进、语句顺序与 shell 语义）；发生变化的
// 行必须等于该行经规则层重构的期望值——任何 diff 都恰好是"字符串
// 字面量内部的变化"这一契约的可执行形式。
func assertStructuralDiff(t *testing.T, src, out []byte) {
	t.Helper()
	inLines := strings.Split(string(src), "\n")
	outLines := strings.Split(string(out), "\n")
	if len(inLines) != len(outLines) {
		t.Fatalf("line count changed: %d -> %d", len(inLines), len(outLines))
	}
	for i := range inLines {
		if inLines[i] == outLines[i] {
			continue
		}
		if want := expectedRewrite(t, inLines[i]); outLines[i] != want {
			t.Errorf("line %d: diff not confined to string literals\n  in:   %q\n  out:  %q\n  want: %q",
				i+1, inLines[i], outLines[i], want)
		}
	}
}

// TestConvertActivateFixture 以 activate 样本副本为 fixture 做端到端
// 断言：结构无损逐行对照 + 全量转换检查（A1/A5）、目标片段逐个确认、
// 展开段与 shell 语义保留、整体幂等（A2）。
func TestConvertActivateFixture(t *testing.T) {
	src := readTestdata(t, "activate-sample.sh")
	out, err := Convert(src)
	if err != nil {
		t.Fatalf("Convert activate fixture: %v", err)
	}

	// A1：所有目标片段完成转换，除目标片段外无任何 diff。
	assertStructuralDiff(t, src, out)
	assertFullyConverted(t, out)
	for _, want := range []string{
		"export __MISE_ORIG_PATH='/c/Users/tester/bin:/c/Program Files/Git/mingw64/bin:",
		"export PATH='/e/develop/mise/shims:/c/Users/tester/bin:/c/Program Files/Git/mingw64/bin:",
		`export PATH="/c/Users/tester/scoop/apps/mise/current/bin:$PATH"`,
		"export __MISE_EXE='/c/Users/tester/scoop/apps/mise/current/bin/mise.exe'",
		"command '/c/Users/tester/scoop/apps/mise/current/bin/mise.exe'",
		"command '/c/Users/tester/scoop/apps/mise/current/bin/mise.exe' \"$command\" \"$@\"",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("converted output missing %q", want)
		}
	}
	// 规则 4/5：展开段、裸值与 shell 语义原样保留。
	for _, want := range []string{
		"export __MISE_ORIG_PATH=\"$PATH\"",
		"eval \"$(mise hook-env ${__MISE_FLAGS[@]+\"${__MISE_FLAGS[@]}\"} --shell-pid $$ -s bash \"$@\")\"",
		"printf -v PROMPT_COMMAND '%s' \"_mise_hook_prompt_command${_mise_prompt_command_value:+;$_mise_prompt_command_value}\"",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("converted output lost untouched fragment %q", want)
		}
	}

	// A2：幂等——对已转换的输出再转换一遍，结果不变。
	again, err := Convert(out)
	if err != nil {
		t.Fatalf("re-Convert: %v", err)
	}
	if string(again) != string(out) {
		t.Fatal("f(f(x)) != f(x): re-converting the converted fixture changed it")
	}
}

// TestConvertHookEnvFixture 以 hook-env 样本副本为 fixture 做端到端
// 断言：路径型单变量转换、非路径变量与 base64 串跳过、结尾 zsh 风格
// 行结构保留（A1/A5）、首个 PATH 值与 cygpath -up 黄金值逐段对照
// （A4）、整体幂等（A2）。
func TestConvertHookEnvFixture(t *testing.T) {
	src := readTestdata(t, "hook-env-sample.sh")
	out, err := Convert(src)
	if err != nil {
		t.Fatalf("Convert hook-env fixture: %v", err)
	}

	assertStructuralDiff(t, src, out)
	assertFullyConverted(t, out)

	// 路径型单变量完成转换（A1）。
	for _, want := range []string{
		"export CARGO_HOME='/c/Users/tester/.cargo'",
		"export GOBIN='/e/develop/mise/installs/go/1.27.1/bin'",
		"export GOROOT='/e/develop/mise/installs/go/1.27.1'",
		"export JAVA_HOME='/e/develop/mise/installs/java/zulu-17.68.203.0'",
		"export RUSTUP_HOME='/c/Users/tester/.rustup'",
		"export UV_PYTHON='/e/develop/mise/installs/python/3.14.7/python.exe'",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("converted output missing %q", want)
		}
	}
	// 非路径变量、base64 串与结尾 zsh 风格行原样保留
	// （规则 5 / 结构无损）。
	for _, want := range []string{
		"export RUSTUP_TOOLCHAIN=stable",
		"if typeset -f __mise_clear_completions >/dev/null; then __mise_clear_completions; unset -f __mise_clear_completions; fi",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("converted output lost untouched line %q", want)
		}
	}
	for _, prefix := range []string{"export __MISE_DIFF=", "export __MISE_SESSION="} {
		inLine, outLine := findLine(t, src, prefix), findLine(t, out, prefix)
		if inLine != outLine {
			t.Errorf("%s... line changed but must be byte-identical", prefix)
		}
	}

	// 黄金对照（A4）：首个 PATH 值逐段对照 cygpath -up 实测输出
	// （testdata/cygpath-expected-hookenv-path.txt）。Git 安装根下的段
	// 有意保留盘符形（/c/Program Files/Git/...，而非挂载规范形
	// /mingw64/bin 等）——已文档化的 v1 偏差（design.md §3.1）。
	golden := strings.TrimRight(string(readTestdata(t, "cygpath-expected-hookenv-path.txt")), "\r\n")
	got := quotedValue(t, findLine(t, out, "export PATH='"))
	goldSegs := strings.Split(golden, ":")
	gotSegs := strings.Split(got, ":")
	if len(gotSegs) != len(goldSegs) {
		t.Fatalf("first PATH segment count = %d, golden has %d", len(gotSegs), len(goldSegs))
	}
	for i := range goldSegs {
		if gotSegs[i] == goldSegs[i] {
			continue
		}
		if gotSegs[i] == "/c/Program Files/Git"+goldSegs[i] {
			continue // 已文档化偏差：盘符形
		}
		t.Errorf("segment %d: got %q, want golden %q (or documented drive-letter form)",
			i, gotSegs[i], goldSegs[i])
	}

	// A2：幂等。
	again, err := Convert(out)
	if err != nil {
		t.Fatalf("re-Convert: %v", err)
	}
	if string(again) != string(out) {
		t.Fatal("f(f(x)) != f(x): re-converting the converted fixture changed it")
	}
}

// findLine 返回 src 中第一个以 prefix 开头的行，找不到则使测试失败。
func findLine(t *testing.T, src []byte, prefix string) string {
	t.Helper()
	for _, line := range strings.Split(string(src), "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	t.Fatalf("no line starting with %q", prefix)
	return ""
}

// quotedValue 提取行内单引号包裹的值（供与黄金值逐段对照）。
func quotedValue(t *testing.T, line string) string {
	t.Helper()
	line = strings.TrimSuffix(line, "\r")
	const q = '\''
	open := strings.IndexByte(line, q)
	if open < 0 || !strings.HasSuffix(line, string(q)) || open == len(line)-1 {
		t.Fatalf("line %q has no single-quoted value", line)
	}
	return line[open+1 : len(line)-1]
}
