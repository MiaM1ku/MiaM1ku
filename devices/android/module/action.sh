#!/system/bin/sh
# 模块页“执行”按钮：刷新应用名缓存并立即上报一次
MODDIR=${0%/*}
echo "MiaM1ku Device Status"
echo "--------------------------------"
sh "$MODDIR/reporter.sh" --refresh-labels --once 2>&1
echo "--------------------------------"
DATA=/data/adb/miam1ku_status
pid=$(cat "$DATA/reporter.pid" 2>/dev/null)
if [ -n "$pid" ] && grep -q reporter.sh "/proc/$pid/cmdline" 2>/dev/null; then
  echo "后台上报进程运行中：pid $pid"
else
  echo "后台上报进程未运行，正在启动"
  nohup sh "$MODDIR/reporter.sh" >/dev/null 2>&1 &
  echo $! > "$DATA/reporter.pid"
fi
