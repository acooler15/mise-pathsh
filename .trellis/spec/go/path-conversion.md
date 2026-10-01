# 核心域约定：路径转换

> 本工具的唯一职责域。所有实现与评审必须对照本文件和真实样本
> `sample-activate.sh`。

---

## 适用范围（已确认）

- **只处理两条命令的 bash 输出**：
  `mise activate --shell bash` 与 `mise hook-env --shell bash`。
- 其他 shell（zsh / fish / pwsh）的输出不在范围内，明确不做。

## 解析方式（已定稿，2026-10-01）

- 使用 **`mvdan.cc/sh/v3`**（v3.14.1，`syntax.NewParser(syntax.KeepComments(true))`）
  解析 bash 脚本；AST 只用于**定位**待转换片段，不做 printer 全文重打印。
- > **Warning（依赖坑）**：模块路径是 `mvdan.cc/sh/v3`，不是
  > `github.com/mvdan/sh/v3`（后者只是重定向页）。`go get github.com/mvdan/sh/v3`
  > 会报 `module declares its path as: mvdan.cc/sh/v3` 而失败，go.mod 里也
  > 只能写 `mvdan.cc/sh/v3`。
- **结构无损的实现方式**：AST 定位命中区间后做**字节级定点替换**（按位置
  降序应用）。禁止用 mvdan printer 重排整个文件——printer 会规范化缩进/
  引号/换行，破坏"除目标片段外无 diff"的验收。

## 样本中观察到的 Windows 路径形态

真实来源：`sample-activate.sh`（本机 `mise activate --shell bash` 输出）。

**形态 1 —— 单引号字符串内的 Windows PATH 值**（分号分隔 + 反斜杠）：

```bash
export __MISE_ORIG_PATH='C:\Users\tester\bin;...;E:\develop\mise\shims;...'
```

**形态 2 —— 双引号内的混合形态**（`\\` 转义反斜杠 + 冒号分隔 + `$PATH` 展开）：

```bash
export PATH="C:\\Users\\tester\\scoop\\apps\\mise\\current\\bin:$PATH"
```

**形态 3 —— 单引号命令参数中的可执行文件路径**：

```bash
command 'C:\Users\tester\scoop\apps\mise\current\bin\mise.exe' "$@"
```

## 转换原则

1. **目标形态**：Git-Bash / MSYS2 的 POSIX 路径——
   `C:\foo\bar` → `/c/foo/bar`；PATH 分隔符 `;` → `:`。
2. **保守转换**：只转换能确认为 Windows 路径的片段（盘符前缀、分号分隔的
   PATH 列表等），拿不准的保持原样。宁可漏转，不可错转。
3. **结构无损**：解析→打印必须保真——注释、引号风格、语句顺序全部保留；
   转换只作用于字符串字面量内部，不得触碰脚本结构。
4. **shell 语义不动**：`$PATH`、`$*`、`$$` 等变量展开、控制流、函数定义
   原样保留。
5. **规则表（v1 定稿，2026-10-01）**：检测只作用于**字符串字面量内部**
   （单/双引号 Lit 部分），按"触发条件"匹配而非变量名白名单——
   `__MISE_DIFF`/`__MISE_SESSION` 的 base64 值不含 `\`，天然不触发：

   | # | 触发 | 动作 |
   |---|------|------|
   | 1 | 值含 `;` 且**每个**非空段匹配 `^[A-Za-z]:[\\/]` | 各段转 POSIX，`;`→`:` |
   | 2 | 片段匹配 `^[A-Za-z]:[\\/][^;]*$` | 盘符→小写 `/x/`，`\`→`/` |
   | 3 | 双引号内 `\\` 序列 | 先按转义对归一（`unescapeDouble`），仅接受纯转义对；`\$`、奇数长 `\` 串拒绝转换 |
   | 4 | `$VAR` / `$(...)` / 反引号展开段 | 不动（AST Parts 隔离） |
   | 5 | base64、裸值（无引号）、无盘符前缀文本 | 不动 |
   | 6 | 尾随 `\` | 保留为尾随 `/` |
   | 7 | 正斜杠形态 `X:/...` | 同规则 2 |
   | 8 | UNC `\\server\...` | 保守跳过 |

   **语义对齐 cygpath**（测试 oracle：黄金值实测存档于任务
   `10-01-path-converter/research/cygpath-oracle.md`）：小写盘符、尾随
   斜杠保留、POSIX 输入原样通过（幂等依据）、列表逐段转换。
   **有意偏差**：不做 MSYS 挂载规范形（Git 根下路径输出 `/c/Program
   Files/Git/cmd` 而非 `/cmd`），二者解析到同一位置，功能等价。
   **不做运行时 exec cygpath**：hook-env 每提示符触发，进程 spawn 开销
   不可接受；cygpath 仅作测试 oracle。
   **附加边界**：不转换无引号裸字面量（bash 会把 `\` 当转义符，改写
   属于改值而非改拼写）；候选上下文仅为 Assign 值与命令字参数
   （`${x:-"..."}` 等默认值展开内的引号串不访问——漏转方向，可接受）。

6. **幂等**：`f(f(x)) == f(x)`——POSIX 形态不匹配任何触发规则，第二遍
   天然 no-op；有测试断言，改动规则表时不得破坏。

---

## 钩子链路不做处理（2026-10-01 决策）

`mise activate` 输出自带的钩子函数（`_mise_hook` /
`_mise_hook_prompt_command` / `_mise_hook_chpwd`，及 `mise()` 函数）
**原样保留，不做任何修改或包装**——三者经 `mise()` 函数漏斗式调用
mise.exe，本工具只在入口对 activate 输出过滤一次，不介入钩子链路。
实验依据（PATH 自持基线、mise 对单变量逐值比对并清除不匹配值、
重导出差异）见任务 `10-01-path-converter` 归档中的
`research/hook-experiments.md`；曾试过的"eval 调用点注入管道"方案
已否决并整体回退，不要再走回头路。
