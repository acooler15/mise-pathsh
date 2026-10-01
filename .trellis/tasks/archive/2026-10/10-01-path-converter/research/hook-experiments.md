# hook 链路实验记录（2026-10-01）

> 背景：评估"activate 输出中的 hook-env 调用点是否需要特殊处理"。
> 结论：注入方案（在 eval 调用点包装管道）已实现并**整体回退**——用户
> 判定方案方向错误。本文件仅存档实验事实，供后续方向决策参考，不代表
> 任何已采纳的设计。

## 实验环境

- mise 2026.9.18，Git Bash（MSYS2），go 1.27.1
- 测试目录：`/tmp/misepro`（java=zulu-17.68.203.0）、`/tmp/miserust`（rust=stable）

## 事实（均有可复现命令链路）

1. **activate 输出的钩子结构**：4 处 `eval "$(mise ...)"` ——
   `_mise_hook`（激活时 `--force` + command_not_found 兜底）、
   `_mise_hook_prompt_command`（每提示符，带 SKIP_FIRST_PROMPT /
   CHPWD_RAN 两道守卫）、`_mise_hook_chpwd`（cd 触发，置位去重标志）、
   `mise()` 函数的 `deactivate|shell|sh` 分支。三者互不委托、各自 eval；
   钩子调用的 `mise` 是脚本内定义的**函数**（函数遮蔽同名 exe），函数
   末行经 `command` 内建执行真实 mise.exe。

2. **PATH 自持**：source 转换后的 activate 后，`__MISE_ORIG_PATH` 与
   PATH 已是 POSIX 基线；此后钩子的 hook-env 输出 PATH 亦为 POSIX
   （cd 换工具集 / `mise deactivate` / `unset __MISE_SESSION __MISE_DIFF`
   模拟状态丢失，四种场景 PATH 分号数均为 0）。即 mise 的 PATH 处理
   **形态自适应**，继承当前基线。

3. **单变量逐值比对**：hook-env 输出中 mise 管理的单变量
   （CARGO_HOME/RUSTUP_HOME/GOROOT/JAVA_HOME/UV_PYTHON）为 Windows
   形态；**在 RAW（未转换）会话中手动把 CARGO_HOME 改为 POSIX 形态后
   cd，下一次 hook-env 输出 `unset CARGO_HOME`**——POSIX 值被判定为
   外来修改而清除；Windows 形态值则保留（对照实验证实因果）。即 mise
   对单变量做**逐字节值比对**，与 PATH 的形态自适应行为不同。

4. **重导出差异**：GOROOT/JAVA_HOME/UV_PYTHON 每次钩子都被重导出
   （值不匹配也能被下次钩子覆盖恢复）；CARGO_HOME/RUSTUP_HOME 不被
   重导出，值不匹配时直接丢失。

5. **全量转换钩子输出的后果**（已回退的实现）：cd 后 CARGO_HOME 变空
   （mise unset）；cargo/go 仍可用（工具回落默认路径：`~/.cargo` 等），
   PATH/GOROOT/JAVA_HOME 保持 POSIX。即"能跑，但 mise 管理的变量被
   静默清除"。

## 已否决的方案（记录理由）

- **eval 调用点注入管道**（含 --path-only 修正版）：用户判定方向错误
  而回退。技术上的代价：结构注入破坏"只改字面量"约束、每钩子多一次
  进程 spawn、把工具自身路径写进用户脚本、依赖对 mise 内部行为的推断。
