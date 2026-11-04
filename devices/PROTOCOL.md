# 设备状态上报协议

首页保持纯静态。浏览器里的 `assets/js/devices.js` 在用户真实打开页面、设备卡片进入视口时请求 `/api/status`；
设备端（Windows 上报程序、Android KernelSU 模块）向 `/api/report` 上报。两个接口都由 EdgeOne Makers Edge Functions 处理，
数据存放在 Makers KV。

## 服务端配置

| 名称 | 类型 | 说明 |
| --- | --- | --- |
| `DEVICE_KV` | KV 命名空间绑定（变量名） | 控制台 → Makers → 存储 → KV：创建命名空间并绑定到项目，变量名填 `DEVICE_KV` |
| `DEVICE_TOKEN` | 项目环境变量 | 上报密钥，设备端以 `Authorization: Bearer <DEVICE_TOKEN>` 携带 |
| `DEVICE_IDS` | 项目环境变量（可选） | 允许上报的设备 ID，逗号分隔，默认 `pc,phone` |
| `OFFLINE_AFTER` | 项目环境变量（可选） | 超过多少秒没有上报视为离线，默认 `180` |

## POST /api/report

请求头：

```
Authorization: Bearer <DEVICE_TOKEN>
Content-Type: application/json
```

请求体（JSON，≤ 4 KB；除 `device` 外全部可选，未知字段会被丢弃）：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `device` | string | 设备 ID：`pc` 或 `phone` |
| `app` | string ≤ 64 | 前台应用的友好名称，例如 `Visual Studio Code`、`哔哩哔哩` |
| `pkg` | string ≤ 128 | 前台应用标识：PC 为进程名（`Code.exe`），手机为包名（`tv.danmaku.bili`） |
| `title` | string ≤ 120 | PC 前台窗口标题 |
| `model` | string ≤ 64 | 设备型号，例如 `Xiaomi 14`、`DESKTOP-XXXX` |
| `os` | string ≤ 64 | 系统版本，例如 `Windows 11 23H2`、`Android 15` |
| `battery` | int 0–100 | 电量百分比 |
| `charging` | bool | 是否在充电 |
| `cpu` | number 0–100 | CPU 占用百分比 |
| `mem` | number 0–100 | 内存占用百分比 |
| `screen` | `"on"` / `"off"` / `"locked"` | 手机屏幕状态 |
| `idle` | int ≥ 0 | PC 距离上次键鼠输入的秒数 |
| `offline` | bool | 设为 `true` 表示设备主动下线（关机、退出程序） |

成功响应：`200 {"ok":true,"ts":<毫秒时间戳>}`。
失败响应：`401` 密钥错误、`400` 参数错误、`413` 请求体过大、`503` KV 未绑定或 `DEVICE_TOKEN` 未配置。

### 上报节奏

- 状态变化时立即上报，两次上报间隔至少 10 秒。
- 状态没有变化时每 60 秒发送一次心跳。
- 服务端在 `OFFLINE_AFTER`（默认 180 秒）内没有收到上报时判定离线。KV 边缘缓存最长 60 秒，180 秒阈值留足余量。

## GET /api/status

```json
{
  "now": 1793700000000,
  "offlineAfter": 180,
  "devices": [
    { "id": "pc", "online": true, "ts": 1793699990000, "app": "Visual Studio Code", "title": "index.html - MiaM1ku", "cpu": 12.5, "mem": 48.2, "idle": 3 },
    { "id": "phone", "online": false, "ts": 1793690000000, "app": "哔哩哔哩", "pkg": "tv.danmaku.bili", "battery": 85, "charging": false, "screen": "off" }
  ]
}
```

KV 未绑定时返回 `503 {"error":"kv_unbound"}`，前端显示“状态服务暂不可用”。
