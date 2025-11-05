# Release v0.1.1

## 发布包说明

本目录包含 Message Mirror v0.1.1 的所有平台发布包。

## 文件列表

- `message-mirror-v0.1.1-linux-amd64.tar.gz` - Linux AMD64 平台
- `message-mirror-v0.1.1-linux-arm64.tar.gz` - Linux ARM64 平台
- `message-mirror-v0.1.1-darwin-amd64.tar.gz` - macOS Intel 平台
- `message-mirror-v0.1.1-darwin-arm64.tar.gz` - macOS Apple Silicon 平台
- `message-mirror-v0.1.1-windows-amd64.zip` - Windows AMD64 平台

## 安装说明

### Linux / macOS
```bash
tar -xzf message-mirror-v0.1.1-linux-amd64.tar.gz
sudo mv message-mirror-linux-amd64 /usr/local/bin/message-mirror
```

### Windows
```powershell
Expand-Archive message-mirror-v0.1.1-windows-amd64.zip
# 将 message-mirror-windows-amd64.exe 添加到 PATH
```

## 验证安装
```bash
message-mirror version
```

## 更多信息

请参阅项目根目录的 [README.md](../README.md) 和 [RELEASE_NOTES.md](../RELEASE_NOTES.md)。

