# Windows 状态上报程序

单个 exe，登录 Windows 后在后台运行，每 2 秒采样一次，向 `https://miam1ku.cn/api/report` 上报：

- 前台应用：友好名称（exe 版本信息里的 FileDescription，例如 `Visual Studio Code`）、进程名、窗口标题
- 键鼠空闲秒数（首页超过 5 分钟显示“挂机中”）
- CPU、内存占用；笔记本额外上报电量与充电状态
- 计算机名与 Windows 版本

前台变化时立即上报（两次间隔至少 10 秒），状态不变时每 60 秒心跳一次；退出、注销或关机时发送一次“离线”。协议见 [PROTOCOL.md](../PROTOCOL.md)。

## 构建

```sh
sh devices/windows/build.sh
```

产物在 `devices/windows/dist/`：

| 文件 | 用途 |
| --- | --- |
| `miam1ku-status.exe` | 正式版，无控制台窗口 |
| `miam1ku-status-console.exe` | 调试版，带控制台输出 |
| `miam1ku-status.json` | 已从 `.secrets/device-token` 填好 token 的配置 |

## 安装

1. 新建目录 `%LOCALAPPDATA%\MiaM1kuStatus\`，放入 `miam1ku-status.exe` 和 `miam1ku-status.json`。
2. 打开 `miam1ku-status.json` 确认 `token` 已填写。
3. 在该目录打开终端执行：

   ```powershell
   .\miam1ku-status.exe -install
   ```

   程序写入 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\MiaM1kuStatus`，之后每次登录自动启动，并立即在后台运行一个实例。

升级时替换 exe 后再执行一次 `-install`：程序会先通知旧实例退出，再启动新版本。

## 卸载

```powershell
.\miam1ku-status.exe -uninstall
```

删除开机自启项，并结束正在运行的实例（先请求优雅退出，8 秒后强制结束）。之后删除整个目录即可。

## 调试

```powershell
.\miam1ku-status-console.exe -once      # 采样一次并打印将要发送的 JSON
.\miam1ku-status-console.exe -console   # 前台运行，日志输出到控制台
```

正式版的日志写在 exe 同目录的 `miam1ku-status.log`，超过 1 MB 时轮转为 `.log.1`。

## 配置

`miam1ku-status.json`（不存在时程序会生成模板并退出）：

```json
{
  "endpoint": "https://miam1ku.cn/api/report",
  "token": "",
  "device": "pc",
  "model": "",
  "show_title": true,
  "hide_title_apps": ["KeePassXC.exe", "1Password.exe"],
  "app_aliases": { "Code.exe": "Visual Studio Code" },
  "heartbeat_seconds": 60,
  "min_interval_seconds": 10
}
```

| 字段 | 说明 |
| --- | --- |
| `model` | 首页显示的设备名，留空时使用计算机名 |
| `show_title` | 设为 `false` 时整体隐藏窗口标题 |
| `hide_title_apps` | 这些进程只上报应用名，隐藏窗口标题（大小写不敏感）；建议把浏览器隐私窗口、密码管理器、聊天软件加进来 |
| `app_aliases` | 进程名 → 显示名，优先级高于 FileDescription |

修改配置后重新执行 `-install`（或注销再登录）即可生效。
