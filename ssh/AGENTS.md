# 维护说明

这两个工具是独立 SSH 程序；不得引入 STB、STB-RDC 或 tmux 依赖。

- macOS 连接、认证与主机指纹：`cmd/stb-tool-ssh/main.go`
- Windows 监听、认证、主机密钥：`cmd/stb-tool-sshd/main.go`
- Windows ConPTY 与 PowerShell 会话：`cmd/stb-tool-sshd/session.go`
- PowerShell ZIP 验证与解包：`cmd/stb-tool-sshd/powershell.go`
- 私有配对和双平台构建：`cmd/package/main.go`

认证只接受构建时配对的 Ed25519 公钥。不要把 `private/`、`../dist/`、`vendor-pwsh/` 或 `internal/builddata/generated.go` 提交到 Git。用户和端口均以命令行参数为准；默认逻辑用户名为 `stb`。Windows 进程始终在启动服务端的账户权限下运行。

在 `ssh/` 目录检查：`go test ./...`，再用 `GOOS=windows GOARCH=amd64 go vet ./cmd/stb-tool-sshd` 检查 Windows 目标代码；完整跨平台构建用 README 中的打包命令。真实 SSH 和 ConPTY 行为需要 Windows x64 实机验收。
