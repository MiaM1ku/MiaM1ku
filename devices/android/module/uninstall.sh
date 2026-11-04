#!/system/bin/sh
# 卸载时停止上报进程；配置与日志保留在 /data/adb/miam1ku_status/
PIDFILE=/data/adb/miam1ku_status/reporter.pid
pid=$(cat "$PIDFILE" 2>/dev/null)
if [ -n "$pid" ] && grep -q reporter.sh "/proc/$pid/cmdline" 2>/dev/null; then
  kill "$pid" 2>/dev/null
fi
rm -f "$PIDFILE"
