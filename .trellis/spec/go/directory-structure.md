# 目录结构

> 现状：仓库尚无 Go 源码。以下为首个实现任务落定时的基线约定；
> 实现中若需偏离，先改本文件再写代码。

---

## 布局约定

| 位置 | 内容 |
|------|------|
| 仓库根 | `go.mod`（module 名建议与仓库同名，实现任务定稿）、`mise.toml`、`sample-activate.sh` |
| 入口 | 单命令 CLI：`main.go` 放仓库根，或 `cmd/mise-pathsh/`（二选一后更新本行） |
| `internal/` | 全部可复用逻辑，按职责分子包，预期方向：bash 解析适配、路径转换核心、转换规则表。不建 `pkg/` |
| `*_test.go` | 与被测代码同目录（Go 惯例） |
| `testdata/` | 端到端转换 fixture。Go 工具链约定：`testdata` 目录不会被编译，可放任意文件 |

---

## 特殊文件

- **`sample-activate.sh`**：`mise activate --shell bash` 在本机的真实输出
  原始样本，是转换逻辑的第一手素材与回归 fixture。**只读，禁止修改**；
  测试需要变体时复制到 `testdata/` 再处理。
