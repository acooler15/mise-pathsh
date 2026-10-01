# 核心域约定：路径转换

> 本工具的唯一职责域。所有实现与评审必须对照本文件和真实样本
> `sample-activate.sh`。

---

## 适用范围（已确认）

- **只处理两条命令的 bash 输出**：
  `mise activate --shell bash` 与 `mise hook-env --shell bash`。
- 其他 shell（zsh / fish / pwsh）的输出不在范围内，明确不做。

## 解析方式（已确认）

- 使用**开源 Go 库解析 bash 脚本**，不手写解析器。
- 候选：`mvdan.sh/v3`（事实标准，支持保留注释的解析与回打印）。
  最终选型在实现任务 `design.md` 定稿，定稿后回填本节。

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
5. **规则表待定稿**：具体到"哪些变量名、哪些上下文、哪些边界情况
   （带空格路径如 `C:\Program Files\...`、大小写盘符等）"的完整转换规则表，
   在实现任务 `design.md` 定稿后回填本节。
