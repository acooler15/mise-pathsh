# 技术设计：mise bash 输出路径转换器

> 对应 prd.md（10-01-path-converter）。规范约束见 `spec/go/`。

---

## 1. 架构与边界

- **module 名**：`mise-pathsh`（本地工具，无 VCS 发布计划；日后如需开源，
  go.mod 加前缀是一行改动）。
- **布局**（符合 spec/go/directory-structure.md）：
  - `main.go`（仓库根）——接口层：读输入（无参=stdin，文件参数=逐个读，
    cat 风格）、写 stdout、错误→stderr+退出码。**无子命令**（Q1 决策）。
  - `internal/converter/`——全部核心逻辑：
    - `rules.go`：转换函数与规则匹配（规则表唯一来源，表驱动测试对象）
    - `locate.go`：AST 遍历，定位待转换字节区间
    - `converter.go`：`Convert(src []byte) ([]byte, error)` 编排
- 不引入 CLI 框架（标准库 flag 足够）；不引入日志库。

## 2. 核心技术决策：AST 定位 + 字节级定点替换（非全文重打印）

**首选方案**：用 `mvdan/sh/v3`（v3.14.1，`parser.KeepComments`）解析出 AST，
只为"理解结构"服务；转换实现为**对命中的字节区间做定点替换**（按 Pos
降序应用，避免偏移失效），**不使用 printer 重排整个文件**。

理由：
1. mvdan printer 会规范化缩进/引号/换行——威胁"除目标片段外无 diff"的
   验收标准（A1/A5）；定点替换使结构无损 trivially 成立。
2. 幂等（A2）更容易推理：POSIX 形态 `/c/...` 不匹配任何触发规则，
   第二遍天然 no-op。
3. 退路（若 AST 定位在某形态上失效）：同文件内对命中行做行级正则替换，
   仍不做全文重打印。两方案共享 rules.go，切换成本低。

**为什么不全文正则**：需要区分引号上下文（单引号原样、双引号 `\\` 转义）、
保留 `$PATH` 展开段、跳过 base64 串——AST 的 word 拆分（Parts：Lit 与
展开段）天然提供这些边界，正则会重蹈覆辙。

## 3. 转换规则表（v1 定稿）

触发检测只作用于**字符串字面量内部**（Lit 节点），顺序即优先级：

| # | 触发条件（字面量或其片段） | 动作 | 示例 |
|---|---------------------------|------|------|
| 1 | 值含 `;`，且每个非空段匹配 `^[A-Za-z]:\\`（Windows 目录列表） | 各段独立转换，`;`→`:`，重组 | `'C:\a;E:\b'` → `/c/a:/e/b` |
| 2 | 片段整体匹配 `^[A-Za-z]:\\[^;]*$`（单路径） | 盘符→`/x/`（小写），`\`→`/` | `'C:\Users\x'` → `/c/Users/x` |
| 3 | 双引号内的 `\\` 序列（字面量部分） | `\\`→`/`（并入规则 2 的路径归一） | `"C:\\x\\bin:$PATH"` → `/c/x/bin:$PATH` |
| 4 | `$VAR` / `$(...)` / 反引号展开段（非 Lit） | **不动**（AST 天然隔离） | `"$PATH"` 原样 |
| 5 | base64 串（`__MISE_DIFF`/`__MISE_SESSION`）、无引号裸值、不含盘符前缀的任何文本 | **不动**（规则 1/2 均不触发即跳过） | `RUSTUP_TOOLCHAIN=stable` |
| 6 | 尾随 `\` | 保留为尾随 `/` | `C:\Users\x\` → `/c/Users/x/` |
| 7 | 正斜杠 Windows 形态 `X:/...` | 同规则 2（匹配放宽为 `^[A-Za-z]:[\\/]`） | `C:/x/bin` → `/c/x/bin` |
| 8 | UNC `\\server\share\...` | **v1 保守跳过**（宁漏勿错；mise PATH 中罕见） | — |

### 3.1 cygpath 语义对齐策略（参考实现）

cygpath（cygwin 3.6.10，Git for Windows 自带）是本领域参考实现。
实测记录与黄金值：`research/cygpath-oracle.md`。

- **采纳其语义**：小写盘符、`\`→`/`、尾随斜杠保留、`C:/` 形态输入、
  列表 `;`→`:` 逐段转换、POSIX 输入原样通过（幂等依据）。
- **有意偏差——挂载规范形**：cygpath 感知 MSYS 挂载表，把 Git 安装根
  `C:\Program Files\Git` 下的路径输出为 `/cmd`、`/mingw64/bin` 等规范形；
  v1 输出 `/c/Program Files/Git/cmd` 等盘符形。二者在 MSYS 命名空间中
  解析到同一位置，功能等价，且 mise round-trip 由 A3 实测验证。挂载
  感知需要运行时查询 MSYS runtime（`cygpath -w /` 之类的 exec），v1 不做；
  若 A3 暴露盘符形的实际缺陷，再启用"启动时一次性根查询 + 前缀规范化"
  模式（本节预留）。
- **不做运行时 exec cygpath**：hook-env 每次 `PROMPT_COMMAND` 触发，
  Windows 进程 spawn 延迟（数十毫秒/次）在每提示符场景不可接受；
  保持零外部依赖。cygpath 仅作为**测试 oracle**（黄金值已实测捕获）。
- **Go 移植调研**：无成熟维护的 Go cygpath 移植（OpenShift
  source-to-image 的 `pkg/util/cygpath` 只是该项目的单函数辅助，不适用）；
  结论=自实现上述窄子集，用 oracle 黄金值逐段断言。

附加约束：
- 带空格路径（`C:\Program Files\...`）由规则 1/2 内部处理（空格不影响
  匹配与替换；列表段以 `;` 为唯一分隔符）。
- 盘符大小写：输出统一小写盘符 `/c/`（MSYS 惯例），路径其余部分保持原样。
- **变量名不作为匹配条件**（黑名单/白名单都不建）：`__MISE_DIFF` 的值
  无 `\` 字符，规则天然不触发——比按变量名过滤更保守、对 mise 版本演化
  更鲁棒。
- `command 'C:\...\mise.exe' "$@"` 的参数（规则 2）：转换为 POSIX 绝对
  路径，bash 解析无歧义。

## 4. 数据流

```
stdin/文件 → []byte
  → syntax.NewParser(KeepComments).Parse
  → syntax.Walk：收集候选区间（Assign 的值、Cmd 的字参数）
      每个候选 word：对 Lit 部分按规则表 1→2 检测，产出 (start,end,替换文本)
  → 区间按位置降序应用定点替换
  → []byte → stdout（成功，退出码 0）
解析失败 → stderr 带行:列诊断 → 退出码 1
用法错误（文件不可读等）→ stderr → 退出码 2
```

## 5. 兼容性与风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| mvdan/sh 解析 mise 输出出现未预期节点（如 hook-env 末尾 zsh 风格行） | 定位遗漏 → 漏转换（可接受方向） | 端到端 fixture 断言转换计数；宁漏勿错 |
| MSYS2 对 POSIX 形态环境变量（GOROOT 等）的反向转换行为 | A3 手动验证可能发现某变量转换后 native 工具异常 | 规则表按"触发条件"实现，若需变量级排除可在 rules.go 加一行排除表（设计已留位） |
| 单引号值内出现 `;` 但非路径（如 `--flag=a;b`） | 误转？不会——规则 1 要求**每段**都匹配盘符路径 | 已由触发条件约束，测试覆盖反例 |
| 盘符形 vs cygpath 挂载规范形（Git 根下路径） | PATH 非规范形（`/c/Program Files/Git/cmd` 而非 `/cmd`） | 功能等价；黄金对照逐段断言 + A3 实测；失败则启用 §3.1 预留的根查询模式 |
| 非 Windows 输入（已 POSIX） | 幂等 no-op | 规则不触发；A2 覆盖 |

## 6. 回滚

纯新增工具，无迁移、无外部状态；"回滚"= 停止使用。每步实现可独立提交
（见 implement.md 回滚点）。
