# Labeler

在 root 的 Android 手机上读取已安装应用的真实名称（用户可见 label，跟随系统语言），供 KernelSU 模块使用。

## 原理

与 scrcpy 相同：用 `app_process` 直接运行一个 dex，在 shell/root 进程里拿到系统 Context，再调用系统 `PackageManager`：

1. `Looper.prepareMainLooper()`
2. `ActivityThread.systemMain().getSystemContext()`（失败时依次回退到 `currentActivityThread()`、`new ActivityThread()`）
3. `getPackageManager()` → `getApplicationLabel(ApplicationInfo)`

所有 Android API 都通过反射调用，因此用本机 JDK（`javac --release 8`）即可编译，再由 d8 转成 dex。

输出为 UTF-8，每行一个 `包名=名称`。名称中的换行、制表符和 `=` 会替换为空格，连续空白合并为一个；查询失败或名称为空的包会跳过。

## 手机上调用

```sh
CLASSPATH=/data/adb/modules/miam1ku_status/bin/labeler.dex app_process /system/bin Labeler            # 全部应用
CLASSPATH=/data/adb/modules/miam1ku_status/bin/labeler.dex app_process /system/bin Labeler tv.danmaku.bili com.tencent.mm   # 指定包名
```

退出码：成功为 0，初始化失败为 1（错误信息写到 stderr）。

## 重新构建

```sh
./build.sh
```

脚本会：

- 首次运行时从 Google Maven 下载最新稳定版 r8 到 `.tools/r8.jar`，之后复用；
- `javac --release 8` 编译 `src/Labeler.java`；
- 用 d8（`--release --min-api 26`）生成 dex；
- 复制到 `../module/bin/labeler.dex`。

需要 JDK 11 及以上版本和 curl。要更新 r8，删除 `.tools/` 后重新运行即可。
