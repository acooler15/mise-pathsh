# Bootstrap Task: Fill Project Development Guidelines

**You (the AI) are running this task. The developer does not read this file.**

The developer just ran `trellis init` on this project for the first time.
`.trellis/` now exists with empty spec scaffolding, and this bootstrap task
exists under `.trellis/tasks/`. When they want to work on it, they should start
this task from a session that provides Trellis session identity.

**Your job**: help them populate `.trellis/spec/` with the team's real
coding conventions. Every future AI session — this project's
`trellis-implement` and `trellis-check` sub-agents — auto-loads spec files
listed in per-task jsonl manifests. Empty spec = sub-agents write generic
code. Real spec = sub-agents match the team's actual patterns.

Don't dump instructions. Open with a short greeting, figure out if the repo
has any existing convention docs (CLAUDE.md, .cursorrules, etc.), and drive
the rest conversationally.

---

## 定位澄清（2026-10-01，开发者确认）

- **项目形态**：Go 命令行工具（编译产物为单个 Go 工具）。
- **工具职责**：将 Windows 上 `mise activate --shell bash` /
  `mise hook-env --shell bash` 输出的 bash 脚本中的 Windows 风格路径，
  转换为 Git-Bash（MSYS2）可用的 POSIX 路径；**只处理这两条命令的输出**。
- **技术路线**：使用开源 Go 依赖解析 bash 脚本内容，不手写解析器。
- **spec 结构决定**：trellis init 生成的 backend/frontend 全栈模板与本
  项目不符，已重构为单一 `go` 层（frontend 删除；原 backend 中不适用的
  database 等模板不保留），另保留 `guides/` 通用思维指南不动。
- **文档语言**：按开发者要求，项目文档尽可能使用中文。
- **工具链执行顺序（2026-10-01 修正）**：Go 命令**优先使用 PATH 中的
  `go`**（mise 激活后会把其管理的工具注入 PATH，PATH 里通常就是 mise
  版本）；PATH 中没有时才回退 mise（`mise install` 后
  `mise exec -- <命令>`）。版本安装/切换仍一律走 mise。
  原"一切操作统一经 mise exec"的写法据此废弃。

## Status（按重构后范围更新）

- [x] 重构 spec 目录（backend/frontend → go 单层）
- [x] 填写 go 层规范（index / toolchain / directory-structure /
      path-conversion / error-handling / logging / quality，共 7 个文件）
- [x] 引用真实代码示例（`sample-activate.sh` 三种路径形态，见
      `spec/go/path-conversion.md`；仓库尚无 Go 源码，代码级示例以真实
      样本代替，实现任务定稿后回填）
- [ ] 实现任务产出 `design.md` 后回填两处"待定稿"：bash 解析库选型、
      完整转换规则表（移交后续任务，不阻塞本任务归档）

---

## Spec files populated（重构后的实际清单）

### Go 层（唯一代码规范层，全中文）

| 文件 | 记录内容 |
|------|----------|
| `.trellis/spec/go/index.md` | 项目定位、规范索引、命令速查、greenfield 现状说明 |
| `.trellis/spec/go/toolchain-guidelines.md` | mise 管理 Go 运行时（源自全局 AGENTS.md + mise.toml 现状） |
| `.trellis/spec/go/directory-structure.md` | Go CLI 目录布局；`sample-activate.sh` 只读约定 |
| `.trellis/spec/go/path-conversion.md` | **核心域**：适用范围、解析方式、样本三种路径形态、转换原则 |
| `.trellis/spec/go/error-handling.md` | 错误包装、定位信息透传、CLI 退出码 |
| `.trellis/spec/go/logging-guidelines.md` | stdout/stderr 分工、静默成功 |
| `.trellis/spec/go/quality-guidelines.md` | 检查命令、测试断言（结构无损/转换正确/幂等）、禁止事项 |

原 init 模板的 backend / frontend 表格已随目录重构删除（决策见上方
"定位澄清"一节）；database-guidelines 等与本项目无关的模板不再保留。

### Thinking guides（保持不动）

`.trellis/spec/guides/` 中的通用思维指南为 init 预填，适用于本项目，未改动。

---

## How to fill the spec

### Step 1: Import from existing convention files first (preferred)

Search the repo for existing convention docs. If any exist, read them and
extract the relevant rules into the matching `.trellis/spec/` files —
usually much faster than documenting from scratch.

| File / Directory | Tool |
|------|------|
| `CLAUDE.md` / `CLAUDE.local.md` | Claude Code |
| `AGENTS.md` | Codex / Claude Code / agent-compatible tools |
| `.cursorrules` | Cursor |
| `.cursor/rules/*.mdc` | Cursor (rules directory) |
| `.windsurfrules` | Windsurf |
| `.clinerules` | Cline |
| `.roomodes` | Roo Code |
| `.github/copilot-instructions.md` | GitHub Copilot |
| `.vscode/settings.json` → `github.copilot.chat.codeGeneration.instructions` | VS Code Copilot |
| `CONVENTIONS.md` / `.aider.conf.yml` | aider |
| `CONTRIBUTING.md` | General project conventions |
| `.editorconfig` | Editor formatting rules |

### Step 2: Analyze the codebase for anything not covered by existing docs

Scan real code to discover patterns. Before writing each spec file:
- Find 2-3 real examples of each pattern in the codebase.
- Reference real file paths (not hypothetical ones).
- Document anti-patterns the team clearly avoids.

### Step 3: Document reality, not ideals

**Critical**: write what the code *actually does*, not what it should do.
Sub-agents match the spec, so aspirational patterns that don't exist in the
codebase will cause sub-agents to write code that looks out of place.

If the team has known tech debt, document the current state — improvement
is a separate conversation, not a bootstrap concern.

---

## Quick explainer of the runtime (share when they ask "why do we need spec at all")

- Every AI coding task spawns two sub-agents: `trellis-implement` (writes
  code) and `trellis-check` (verifies quality).
- Each task has `implement.jsonl` / `check.jsonl` manifests listing which
  spec files to load.
- The platform hook auto-injects those spec files + the task's `prd.md`
  into every sub-agent prompt, so the sub-agent codes/reviews per team
  conventions without anyone pasting them manually.
- Source of truth: `.trellis/spec/`. That's why filling it well now pays
  off forever.

---

## Completion

When the developer confirms the checklist items above are done with real
examples (not placeholders), guide them to run:

```bash
python ./.trellis/scripts/task.py finish
python ./.trellis/scripts/task.py archive 00-bootstrap-guidelines
```

After archive, every new developer who joins this project will get a
`00-join-<slug>` onboarding task instead of this bootstrap task.

---

## Suggested opening line

"Welcome to Trellis! Your init just set me up to help you fill the project
spec — a one-time setup so every future AI session follows the team's
conventions instead of writing generic code. Before we start, do you have
any existing convention docs (CLAUDE.md, .cursorrules, CONTRIBUTING.md,
etc.) I can pull from, or should I scan the codebase from scratch?"
