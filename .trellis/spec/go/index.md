# Go 开发规范（go 层）

> 本项目唯一的代码规范层。每个 AI 会话的 `trellis-implement` / `trellis-check`
> 子代理都会按任务 jsonl 清单加载这里的文件。

---

## 项目定位

**mise-pathsh** 是一个 Go 命令行工具：把 Windows 上
`mise activate --shell bash` / `mise hook-env --shell bash`
输出的 bash 脚本中的 Windows 风格路径，转换为 Git-Bash（MSYS2）可用的
POSIX 路径。bash 脚本解析使用开源 Go 库（不手写解析器）。

---

## 规范索引

| 文件 | 内容 |
|------|------|
| [toolchain-guidelines.md](./toolchain-guidelines.md) | Go 命令优先用 PATH 中的 `go`，缺失时回退 mise；版本安装/切换走 mise |
| [directory-structure.md](./directory-structure.md) | Go 代码目录布局约定 |
| [path-conversion.md](./path-conversion.md) | **核心域约定**：转换范围、样本形态、转换原则 |
| [error-handling.md](./error-handling.md) | 错误包装与 CLI 退出码 |
| [logging-guidelines.md](./logging-guidelines.md) | stdout / stderr 分工 |
| [quality-guidelines.md](./quality-guidelines.md) | 提交前检查命令与测试要求 |

---

## 命令速查

```bash
# PATH 中有 go（含 mise 注入的 shims）时直接用：
go build ./...
go test ./...
gofmt -l .      # 期望输出为空
go vet ./...

# 仅当 PATH 中没有 go 时，回退 mise：
mise install && mise exec -- go build ./...
```

判断标准：`go version` 成功即直接用 `go`；失败则 `mise install` 后用
`mise exec -- <命令>`。详见
[toolchain-guidelines.md](./toolchain-guidelines.md)。

---

## 现状说明（2026-10-01 首个实现任务后）

仓库已有可用实现：`main.go`（根，stdin/文件 → stdout 过滤器）+
`internal/converter/`（rules / locate / converter 三层，规则表见
[path-conversion.md](./path-conversion.md)）。测试 fixture 在 `testdata/`
（`sample-activate.sh` 与 hook-env 样本的只读副本，原件禁止修改）。
规范中如仍有"待定稿"字样的陈旧条目，以后续任务的设计评审为准更新。
