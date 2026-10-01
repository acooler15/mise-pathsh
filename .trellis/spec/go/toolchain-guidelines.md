# 工具链规范（PATH 优先，mise 兜底）

> 来源：开发者确认的执行顺序约定 + 项目 `mise.toml` 现状。本节是硬约定。

---

## 命令执行顺序（PATH 优先）

1. **优先直接使用 PATH 中的 `go`**。mise 激活后会把它管理的工具（shims）
   注入 PATH，因此 PATH 里的 `go` 通常就是 mise 管理的版本，直接
   `go build` / `go test` 即可，**不需要也不应该绕道 `mise exec`**。
2. **PATH 中没有 `go` 时才回退 mise**：先 `mise install` 确保
   `mise.toml` 声明的版本就绪，再用 `mise exec -- <命令>` 执行。
3. **安装或切换 Go 版本一律通过 mise**（`mise use go@<版本>`），
   禁止系统包管理器、官方安装脚本、手动下载。

```bash
# 判断走哪条路：
go version || mise install    # 有 go 直接用；没有则让 mise 补齐
```

## 项目现状（记录现实）

`mise.toml` 当前内容：

```toml
[tools]
go = "latest"
```

`go = "latest"` 可复现性弱。若实现任务需要可复现构建，用
`mise use go@<版本>` 写入项目 `mise.toml` 钉住版本——是否钉、钉哪个版本
由实现任务决定，当前不强行改动。

## 其他

- 本项目不含 Python 组件，全局约定中的 uv 规则在此不适用。
- 构建产物（编译出的 exe）不提交进仓库。
- 注意：本项目本身就研究 mise 的 PATH 注入行为（见
  [path-conversion.md](./path-conversion.md)），"PATH 里是什么就用什么"
  的原则与项目主题一致；只有 PATH 兜不住时 mise 才出手。
