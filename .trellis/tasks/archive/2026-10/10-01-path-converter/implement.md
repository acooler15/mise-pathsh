# 执行计划：mise bash 输出路径转换器

> 对应 prd.md / design.md（10-01-path-converter）。
> 命令一律 PATH 优先（`go ...` 直接执行；失败才 `mise install` +
> `mise exec --`，见 spec/go/toolchain-guidelines.md）。

---

## 有序清单

1. [ ] **初始化模块**：`go mod init mise-pathsh`；
      `go get github.com/mvdan/sh/v3@v3.14.1`。
2. [ ] **rules.go + 表驱动测试**：实现规则表（design.md §3）的检测与
      转换函数；测试覆盖每条规则正反例（含 `C:\Program Files\...`、
      base64 串、`stable` 裸值反例）。
3. [ ] **locate.go**：AST 遍历收集候选区间（Assign 值、命令字参数），
      单测用小型脚本片段断言区间与上下文（引号类型、展开段隔离）。
4. [ ] **converter.go**：`Convert` 编排（parse→定位→降序定点替换→输出）；
      解析错误包装行:列信息返回。
5. [ ] **main.go**：无参 stdin→stdout；文件参数逐个 cat 到 stdout；
      退出码 0/1/2；stderr 纪律（spec/go/logging-guidelines.md）。
6. [ ] **fixtures**：复制（勿改原件）`sample-activate.sh` →
      `testdata/activate-sample.sh`；`research/hook-env-sample.sh` →
      `testdata/hook-env-sample.sh`。
7. [ ] **端到端测试**（spec/go/quality-guidelines.md 断言集）：
      A1 全量转换 diff 仅在目标片段；A2 幂等 `f(f(x))==f(x)`；
      A5 结构无损（注释/引号/顺序）；解析失败→错误+退出码 1。
8. [ ] **全量验证**（见下）。

## 验证命令

```bash
gofmt -l .          # 输出必须为空
go vet ./...
go test ./...
go build ./...
# 手动集成（A3，Windows Git Bash）：
bash -c 'eval "$(mise activate --shell bash | ./mise-pathsh.exe)"; echo "$PATH"; mise --version'
bash -c 'eval "$(mise hook-env --shell bash | ./mise-pathsh.exe)"; go version'
```

## 风险文件与回滚点

- 每步（1→8）独立可提交；步骤 2/3/4 均为纯新增文件，任一步失败
  `git checkout -- <file>` 即回滚，不影响其余步骤。
- `sample-activate.sh` 与 `research/hook-env-sample.sh` 为只读原件，
  禁止修改（fixture 一律复制到 testdata/）。

## task.py start 前检查

- [x] prd.md 收敛（无未决开放问题）
- [x] design.md / implement.md 就绪
- [x] implement.jsonl（6 条）/ check.jsonl（5 条）含真实条目
- [ ] 用户对最终规划摘要的明确批准（本文件更新于摘要提出后）
