# mise-pathsh

Windows 下 Git Bash（MSYS2）使用 [mise](https://mise.jdx.dev/) bash 集成时的
路径转换过滤器。

`mise activate --shell bash` / `mise hook-env --shell bash` 在 Windows 上输出
的脚本内嵌 Windows 形态路径（`C:\...` 反斜杠、分号分隔 PATH），Git Bash 无法
直接解析。本工具把这类输出转换为 POSIX 形态（`/c/...`、冒号分隔），其余内容
原样保留（注释、引号风格、语句顺序不变；转换幂等）。

## 构建

需要 Go 1.27+（本仓库用 mise 声明，见 `mise.toml`）：

```bash
go build -o mise-pathsh.exe .
```

## 用法

cat 风格过滤器：无参数读 stdin，文件参数逐个读到 stdout；诊断走 stderr。
退出码：`0` 成功、`1` 输入不是合法 bash（带行:列诊断）、`2` 用法错误
（文件不可读等）。

## 接入 Git Bash

在 `~/.bashrc` 中一行接入：

```bash
eval "$(mise activate bash | mise-pathsh)"
```

`mise-pathsh` 只在这一行出现一次。activate 输出转换后自带钩子
（`_mise_hook_prompt_command` 等），之后每次提示符 / cd 时 mise 的
`hook-env` 由钩子自动调用并原样 eval，**不再需要 mise-pathsh 参与**：
hook-env 的 PATH 输出会继承转换后的 POSIX 基线，保持正确形态。

注意：activate 接入之后不要再手动往 `PROMPT_COMMAND` 里追加 hook-env
调用（无论是否经过 mise-pathsh）——activate 安装的钩子已经在里面了，
重复添加会让每次提示符刷新两遍。

非 Windows 输入（已是 POSIX 路径）原样通过，接入后不影响其他平台脚本。

## 转换语义

- 触发条件按"内容形态"而非变量名：值含 `;` 且每段均为 `X:\` 或 `X:/`
  盘符路径 → 列表转换（`;`→`:`）；单个盘符路径 → `/x/...`；双引号内
  `\\` 转义对归一。
- 不动：`$VAR` 等展开段、base64 串、无引号裸值、无盘符前缀文本、
  UNC 路径（保守跳过）。
- 输出对齐 cygpath 惯例：小写盘符、尾随斜杠保留；有意偏差是不做 MSYS
  挂载规范形（`/c/Program Files/Git/cmd` 而非 `/cmd`），二者解析等价。

规则细节见 `.trellis/spec/go/path-conversion.md`。
