#!/bin/sh
# 联调：reporter.sh 常驻模式 + 本地 dev-server（需先运行 DEVICE_TOKEN=tok node tests/dev-server.mjs 8788）
set -eu
HERE=$(cd "$(dirname "$0")" && pwd)
TMP=$(mktemp -d)
mkdir -p "$TMP/bin" "$TMP/data"
cp -r "$HERE/../module" "$TMP/mod"
cat > "$TMP/bin/dumpsys" <<'EOF'
#!/bin/sh
case "$1 $2" in
  "power "*) echo "  mWakefulness=Awake" ;;
  "window policy") echo "      showing=false" ;;
  "activity activities") echo "  topResumedActivity=ActivityRecord{1 u0 $(cat "$FGFILE")/.Main t1}" ;;
  "battery "*) printf '  USB powered: true\n  level: 64\n' ;;
esac
EOF
printf '#!/bin/sh\n[ "$1" = ro.product.marketname ] && echo "Test Phone"; [ "$1" = ro.build.version.release ] && echo 15; exit 0\n' > "$TMP/bin/getprop"
printf '#!/bin/sh\necho "tv.danmaku.bili=哔哩哔哩"; echo "com.tencent.mm=微信"\n' > "$TMP/bin/app_process"
chmod +x "$TMP/bin/"*
cat > "$TMP/data/config.sh" <<'EOF'
ENDPOINT="http://127.0.0.1:8788/api/report"
TOKEN="tok"
INTERVAL=2
MIN_INTERVAL=3
HEARTBEAT=60
EOF
export FGFILE="$TMP/fg"
echo tv.danmaku.bili > "$FGFILE"
PATH="$TMP/bin:$PATH" MS_MODDIR="$TMP/mod" MS_DATA="$TMP/data" sh "$TMP/mod/reporter.sh" &
PID=$!
sleep 3
curl -s http://127.0.0.1:8788/api/status; echo
echo com.tencent.mm > "$FGFILE"
sleep 5
curl -s http://127.0.0.1:8788/api/status; echo
kill $PID
cat "$TMP/data/reporter.log"
rm -rf "$TMP"
