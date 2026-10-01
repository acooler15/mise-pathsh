# cygpath 语义对照（oracle 实测记录）

> 实测环境：Git for Windows 内置 cygpath（cygwin 3.6.10），2026-10-01。
> 用途：为 rules.go 表驱动测试提供黄金值；记录与 cygpath 的有意偏差。

## 实测输入 → 输出

| 输入（单引号内原样） | `cygpath -u/-up` 输出 | 备注 |
|---|---|---|
| `C:\Users\tester\bin` | `/c/Users/tester/bin` | 基准：小写盘符、正斜杠 |
| `C:\Program Files\Git\cmd` | `/cmd` | **挂载**：Git 根 = `/` |
| `C:\Program Files\Git\mingw64\bin` | `/mingw64/bin` | 同上 |
| `C:\Program Files\Git\usr\bin` | `/usr/bin` | 同上 |
| `e:\develop\mise\shims` | `/e/develop/mise/shims` | 小写盘符输入同样处理 |
| `C:\Users\tester\bin\` | `/c/Users/tester/bin/` | 尾随斜杠保留 |
| `C:/Users/tester/bin` | `/c/Users/tester/bin` | 正斜杠 Windows 形态也接受 |
| `C:\Users\tester\bin;E:\develop\mise\shims;C:\Program Files\dotnet`（-up） | `/c/Users/tester/bin:/e/develop/mise/shims:/c/Program Files/dotnet` | 列表：`;`→`:`，逐段转换 |
| `\server\share\dir`（UNC 经 bash 传参后） | `/e/server/share/dir` | **失真**：MSYS 参数转换把
`\\server\share` 塌缩为当前盘相对路径——UNC 不可经 shell 传参测 oracle；工具按字节处理不受此影响，但 v1 对 UNC 保守跳过 |
| `/already/posix` | `/already/posix` | POSIX 输入原样通过（幂等依据） |

## 完整 PATH 黄金对照

`cygpath-expected-hookenv-path.txt`：hook-env 样本第一条 PATH 值
（46 段）经 `cygpath -up` 的完整输出。其中 Git 安装目录段为挂载规范形
（`/cmd`、`/mingw64/bin`、`/usr/bin`），**与本项目 v1 的有意偏差**：
v1 输出 `/c/Program Files/Git/cmd` 等盘符形（见 design.md「cygpath 对齐
策略」）。测试断言：非 Git 根下路径与该黄金文件逐段一致；Git 根下路径
断言为文档化的盘符形。
