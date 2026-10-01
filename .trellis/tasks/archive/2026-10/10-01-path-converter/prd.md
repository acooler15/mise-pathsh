# mise bash 输出路径转换器（10-01-path-converter）

## 目标 / 用户价值

在 Windows 的 Git Bash（MSYS2）中直接、正确地使用 mise 的 bash 集成。
`mise activate --shell bash` / `mise hook-env --shell bash` 在 Windows 上
输出的脚本内嵌 Windows 形态路径（`C:\...` 反斜杠、分号分隔 PATH），导致
Git Bash 下 PATH 解析失效。本工具（Go CLI）把这类输出转换为 Git Bash 可用
的 POSIX 形态（`/c/...`、冒号分隔），使 `eval "$(... | mise-pathsh)"` 一行
接入成为可能。

## 已确认事实（仓库与环境证据）

- **activate 输出形态**（真实样本 `sample-activate.sh`，仓库根）：
  1. 单引号内的 Windows PATH 值：`export __MISE_ORIG_PATH='C:\...;E:\...'`
  2. 双引号内混合形态：`export PATH="C:\\...\\bin:$PATH"`（`\\` 转义 +
     冒号分隔 + `$PATH` 展开）
  3. 单引号命令参数：`command 'C:\...\mise.exe' "$@"`
- **hook-env 输出形态**（2026-10-01 实采样，mise 2026.9.18，存档于
  `research/hook-env-sample.sh`）：
  4. 多段 `export PATH='...'`（单引号 + 分号 + 反斜杠，同形态 1）
  5. **路径型单变量赋值**：`CARGO_HOME`、`GOBIN`、`GOROOT`、`JAVA_HOME`、
     `RUSTUP_HOME`、`UV_PYTHON` 等值为单个 Windows 路径
  6. **非路径变量**：`RUSTUP_TOOLCHAIN=stable`（无引号裸值，不得转换）
  7. **base64 压缩串**：`__MISE_DIFF` / `__MISE_SESSION`——不得转换
     （其字符集不含 `\`，保守规则天然避开，规则表已显式记录）
  8. 结尾含 zsh 风格条件行（`if typeset -f ...`），结构必须原样保留
- **工具链**：`go 1.27.1` 已在 PATH 直接可用（mise 安装注入）；mise
  2026.9.18。
- **解析库**：`github.com/mvdan/sh/v3` v3.14.1（`go list -m -versions`
  实查为最新，支持保留注释的解析）。
- **规范约束**：`spec/go/` 全部适用——保守转换（宁漏勿错）、结构无损、
  退出码 0/1/2、stdout 只出产品数据、PATH 优先工具链。

## 关键决策

- **D1 接口形态 = stdin→stdout 过滤器**（cat 风格：无参读 stdin，
  文件参数逐个读到 stdout；无子命令）。提问未获回应，按推荐方案落定，
  最终规划摘要审批时可否决。
- **D2 module 名 `mise-pathsh`**；布局 `main.go`（根）+
  `internal/converter/`。
- **D3 转换实现 = AST 定位 + 字节级定点替换**，不做全文重打印
  （mvdan/sh printer 会规范化格式，威胁"除目标片段外无 diff"验收）。
- **D4 规则表按"触发条件"而非变量名白名单**：base64 串无 `\` 字符天然
  不触发，比变量名过滤对 mise 版本演化更鲁棒。详见 design.md §3。
- **D5 cygpath = 语义参考与测试 oracle，不作为运行时依赖**（开发者建议
  采纳）。实测（cygwin 3.6.10）：小写盘符、尾随斜杠保留、`C:/` 输入、
  列表 `;`→`:` 全部采纳；**挂载规范形为有意偏差**（cygpath 把 Git 根下
  路径输出为 `/cmd` 等规范形，v1 输出功能等价的盘符形，避免每提示符
  exec 开销）；UNC v1 保守跳过（bash 传参实测 UNC 会被 MSYS 参数转换
  失真）。黄金对照存 `research/cygpath-oracle.md`，规则表逐段断言。

## 需求（MVP）

- R1 **接口**：stdin→stdout 过滤器（D1）。
- R2 **适用范围**：只处理 `mise activate --shell bash` 与
  `mise hook-env --shell bash` 的输出；其他输入按保守原则处理
  （无触发片段即原样通过）。
- R3 **转换规则表**：design.md §3 v1 定稿——分号 PATH 列表、单盘符
  路径、双引号 `\\` 归一、展开段隔离、base64/裸值跳过、尾随斜杠保留、
  `C:/` 形态接受、UNC 跳过；语义对齐 cygpath（D5，偏差已文档化）。
- R4 **结构无损与幂等**：注释、引号风格、语句顺序保留；
  `f(f(x)) == f(x)`。
- R5 **质量门槛**：gofmt / go vet / go test 全绿（PATH 优先命令）；
  转换逻辑表驱动测试；`sample-activate.sh` 与 hook-env 样本副本作
  fixture。

## 验收标准

- [ ] A1 对 `sample-activate.sh` 与 hook-env 样本副本全量转换：所有
      分号分隔 Windows PATH 值变为冒号分隔 POSIX 形态；路径型单变量
      （CARGO_HOME 等）为 POSIX 形态；**除目标片段外无任何 diff**。
- [ ] A2 幂等：对转换输出再转换一次，结果不变（有测试断言）。
- [ ] A3 集成验证（手动）：Git Bash 中 source 转换后的 activate 输出，
      `echo $PATH` 为冒号分隔且 `mise --version` 可执行；转换后的
      hook-env 输出 source 后 `go version` 正常。
- [ ] A4 表驱动单测覆盖 R3 规则表每一行（含反例：base64、裸值、带空格
      路径）；黄金值取自 cygpath 实测对照（`research/cygpath-oracle.md`，
      非 Git 根路径与 `cygpath -up` 输出逐段一致）；解析失败退出码 1、
      用法错误退出码 2，诊断走 stderr。
- [ ] A5 结构无损：转换 diff 仅出现在字符串字面量内部（注释/引号/顺序
      不变，有测试断言）。

## Out of Scope

- 其他 shell（zsh / fish / pwsh）的输出处理。
- 修改 mise 自身行为或向 mise 上游报 issue。
- `.bashrc` 集成生成器 / `install` 子命令（延后，README 给出手动接入示例）。
- 非 Windows 环境的路径优化（已 POSIX 的输入原样通过）。
