# Shared Terminal Bridge Tools

<p align="center"><a href="https://github.com/sharpbai/shared-terminal-bridge">STB Core</a> · <a href="https://github.com/sharpbai/shared-terminal-bridge-rdc">RDC Adapter</a> · <a href="https://github.com/sharpbai/shared-terminal-bridge-tools"><strong>Tools</strong></a> · <a href="https://github.com/sharpbai/shared-terminal-bridge-docs">Documentation</a></p>

这里集中维护可独立使用的辅助工具。每个工具在自己的目录中管理实现语言、依赖、构建和使用说明；根目录不要求统一使用 Go。

## 子工具导航

| 子工具 | 用途 | 使用与构建 |
| --- | --- | --- |
| [SSH](ssh/README.md) | macOS 上的 `stb-tool-ssh` 连接 Windows 上的 `stb-tool-sshd.exe`，进入内置 PowerShell 7 | [SSH README](ssh/README.md) |

工具源码位于各自目录，发布产物集中在 `dist/<工具名>/`。目前只有 `ssh/`；它与 STB、STB-RDC 的运行时没有依赖关系。
