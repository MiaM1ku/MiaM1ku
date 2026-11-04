#!/usr/bin/env bash
# 交叉编译 Windows 上报程序，并在 dist/ 生成填好 token 的配置文件。
set -euo pipefail

cd "$(dirname "$0")"
TOKEN_FILE="$(cd ../.. && pwd)/.secrets/device-token"

mkdir -p dist
export GOOS=windows GOARCH=amd64 CGO_ENABLED=0

go build -trimpath -ldflags "-s -w -H windowsgui" -o dist/miam1ku-status.exe .
go build -trimpath -ldflags "-s -w" -o dist/miam1ku-status-console.exe .

# 配置模板与 config.go 中的 ConfigTemplate 保持一致；token 从 .secrets 读取，只写入 dist/。
token=""
if [[ -f "$TOKEN_FILE" ]]; then
  token="$(tr -d '\r\n' < "$TOKEN_FILE")"
else
  echo "警告：$TOKEN_FILE 不存在，dist/miam1ku-status.json 中 token 留空" >&2
fi
# JSON 字符串转义：反斜杠与双引号。
token_json="${token//\\/\\\\}"
token_json="${token_json//\"/\\\"}"

umask 077
cat > dist/miam1ku-status.json <<EOF
{
  "endpoint": "https://miam1ku.cn/api/report",
  "token": "${token_json}",
  "device": "pc",
  "model": "",
  "show_title": true,
  "hide_title_apps": ["KeePassXC.exe", "1Password.exe"],
  "app_aliases": { "Code.exe": "Visual Studio Code" },
  "heartbeat_seconds": 60,
  "min_interval_seconds": 10
}
EOF
chmod 600 dist/miam1ku-status.json

ls -l dist/
