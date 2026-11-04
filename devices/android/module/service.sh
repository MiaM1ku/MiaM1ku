#!/system/bin/sh
# late_start service：开机完成后在后台启动 reporter.sh（单实例）
MODDIR=${0%/*}
DATA=/data/adb/miam1ku_status
PIDFILE=$DATA/reporter.pid

until [ "$(getprop sys.boot_completed)" = "1" ]; do
  sleep 5
done
sleep 15

mkdir -p "$DATA"

if [ -f "$PIDFILE" ]; then
  pid=$(cat "$PIDFILE" 2>/dev/null)
  if [ -n "$pid" ] && grep -q reporter.sh "/proc/$pid/cmdline" 2>/dev/null; then
    exit 0
  fi
fi

nohup sh "$MODDIR/reporter.sh" >/dev/null 2>&1 &
echo $! > "$PIDFILE"
