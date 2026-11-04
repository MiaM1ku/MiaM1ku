# Android 状态上报模块（KernelSU / Magisk / APatch）

模块在手机开机后以 root 身份后台运行，每 5 秒采样一次，向 `https://miam1ku.cn/api/report` 上报：

- 前台应用名称与包名（锁屏、息屏时显示“锁屏”）
- 屏幕状态：亮屏 / 锁屏 / 息屏
- 电量与充电状态
- 机型与 Android 版本

状态变化时立即上报（两次间隔至少 10 秒），状态不变时每 60 秒发一次心跳。协议见 [PROTOCOL.md](../PROTOCOL.md)。

## 安装

1. 在本仓库运行 `sh devices/android/build.sh`，得到 `devices/android/dist/miam1ku_status-v1.0.0.zip`，以及填好 TOKEN 的 `dist/config.sh`。
2. 在 KernelSU / Magisk / APatch 管理器里刷入 zip。
3. 把 TOKEN 写进配置文件 `/data/adb/miam1ku_status/config.sh`。两种做法任选其一：
   - 用 `dist/config.sh` 覆盖它：`adb push dist/config.sh /data/local/tmp/ && adb shell su -c cp /data/local/tmp/config.sh /data/adb/miam1ku_status/config.sh`
   - 用 MT 管理器等 root 文件管理器编辑，填入 `TOKEN="..."`。
4. 重启手机。也可以在模块页点击“执行”，立即上报一次并启动后台进程。

修改配置后无需重启，下一个采样周期自动生效。升级模块时沿用已有配置。

## 应用名称

名称按以下顺序查找：

1. `HIDE_APPS` 中的包名显示为“隐私应用”，同时隐藏包名；
2. 用户覆盖表 `/data/adb/miam1ku_status/apps.conf`（格式 `包名=名称`，升级时保留）；
3. 模块内置覆盖表 `apps.conf`（各厂商桌面统一显示为“桌面”）；
4. 系统真实应用名：`bin/labeler.dex` 通过 `app_process` 调用系统 PackageManager 读取，结果缓存在 `/data/adb/miam1ku_status/labels.cache`，原理见 [labeler/README.md](labeler/README.md)；
5. 以上都查不到时显示包名。

应用名缓存在启动时生成，每 6 小时全量刷新；新安装的应用会在第一次出现在前台时单独查询（同一应用 10 分钟内最多查询一次）。在模块页点击“执行”会立即全量刷新。

## 隐私

在 `config.sh` 里设置 `HIDE_APPS`，用空格分隔多个包名：

```sh
HIDE_APPS="com.tencent.mm org.telegram.messenger"
```

## 日志与调试

- 日志：`/data/adb/miam1ku_status/reporter.log`（超过 256 KB 时保留最后 100 行）
- 查看将要发送的数据：`su -c 'DRY_RUN=1 sh /data/adb/modules/miam1ku_status/reporter.sh'`
- 立即上报一次：`su -c 'sh /data/adb/modules/miam1ku_status/reporter.sh --once'`

## HTTPS 客户端

模块按顺序使用：

1. `/data/adb/modules/miam1ku_status/bin/curl`（自行放入）
2. 系统 PATH 里的 `curl`
3. KernelSU / Magisk / APatch 自带的 `busybox wget`

日志里出现 HTTPS 或 TLS 相关的失败时，下载一个 arm64 静态 curl（例如 [stunnel/static-curl](https://github.com/stunnel/static-curl/releases) 的 `curl-linux-aarch64`），放到 `/data/adb/modules/miam1ku_status/bin/curl` 并 `chmod 755`，下个周期自动切换。

## 卸载

在管理器里移除模块并重启。配置、缓存和日志保留在 `/data/adb/miam1ku_status/`，需要清理时执行 `su -c rm -rf /data/adb/miam1ku_status`。

## 开发

- `sh devices/android/test/run.sh`：用假的 `dumpsys` / `getprop` / `app_process` 在本机运行 reporter.sh，覆盖包名解析、锁屏、覆盖表、缓存、隐私应用与 JSON 转义。
- `sh devices/android/labeler/build.sh`：重新编译 `bin/labeler.dex`。
