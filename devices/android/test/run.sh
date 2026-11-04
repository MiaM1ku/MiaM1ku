#!/bin/sh
# 本地模拟测试：用假的 dumpsys / getprop / app_process 运行 reporter.sh（DRY_RUN）
set -eu
HERE=$(cd "$(dirname "$0")" && pwd)
MOD="$HERE/../module"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/mod/bin" "$TMP/data"
cp "$MOD/reporter.sh" "$MOD/apps.conf" "$TMP/mod/"
: > "$TMP/mod/bin/labeler.dex"

# 场景由环境变量控制：FG（前台包名）、WAKE、KEYGUARD
cat > "$TMP/bin/dumpsys" <<'EOF'
#!/bin/sh
case "$1 $2" in
  "power "*) echo "  mWakefulness=${WAKE:-Awake}" ;;
  "window policy") echo "    KeyguardServiceDelegate"; echo "      showing=${KEYGUARD:-false}" ;;
  "activity activities") echo "  topResumedActivity=ActivityRecord{1a2b u0 ${FG:-tv.danmaku.bili}/.MainActivityV2 t123}" ;;
  "window "*) echo "  mCurrentFocus=Window{abc u0 ${FG:-tv.danmaku.bili}/tv.danmaku.bili.MainActivityV2}" ;;
  "battery "*) printf '  AC powered: false\n  USB powered: true\n  level: 77\n' ;;
esac
EOF
cat > "$TMP/bin/getprop" <<'EOF'
#!/bin/sh
case "$1" in
  ro.product.marketname) echo "Xiaomi 14" ;;
  ro.build.version.release) echo "15" ;;
esac
EOF
cat > "$TMP/bin/app_process" <<'EOF'
#!/bin/sh
shift 2
ALL='tv.danmaku.bili=哔哩哔哩
com.tencent.mm=微信
com.miui.home=系统桌面
com.example.quote=他说"你好"\路径'
if [ $# -eq 0 ]; then echo "$ALL"; else for p in "$@"; do echo "$ALL" | grep "^$p="; done; [ "$1" = "org.new.app" ] && echo "org.new.app=新应用"; fi
EOF
chmod +x "$TMP/bin/"*

run() {
  PATH="$TMP/bin:$PATH" MS_MODDIR="$TMP/mod" MS_DATA="$TMP/data" DRY_RUN=1 sh "$TMP/mod/reporter.sh" 2>/dev/null
}

check() { # $1=描述 $2=JSON $3=python 断言表达式（变量 j）
  printf '%s\n' "$2" | python3 -c "import json,sys; j=json.load(sys.stdin); assert $3, j" && echo "PASS $1"
}

j=$(run); check "labels.cache 命中" "$j" "j['app']=='哔哩哔哩' and j['pkg']=='tv.danmaku.bili' and j['screen']=='on' and j['model']=='Xiaomi 14' and j['os']=='Android 15'"
check "sysfs 无电池时回退 dumpsys battery" "$j" "('battery' not in j) or j['battery']==77"
j=$(FG=com.miui.home run); check "覆盖表优先于 labels.cache" "$j" "j['app']=='桌面'"
j=$(FG=com.example.quote run); check "JSON 转义" "$j" "j['app']=='他说\"你好\"\\\\路径'"
j=$(FG=org.new.app run); check "增量查询新应用" "$j" "j['app']=='新应用'"
grep -q '^org.new.app=新应用' "$TMP/data/labels.cache" && echo "PASS 增量结果写入缓存"
j=$(FG=com.unknown.pkg run); check "未知应用回退包名" "$j" "j['app']=='com.unknown.pkg'"
printf 'HIDE_APPS="com.tencent.mm"\n' > "$TMP/data/config.sh"
j=$(FG=com.tencent.mm run); check "HIDE_APPS" "$j" "j['app']=='隐私应用' and 'pkg' not in j"
printf 'com.tencent.mm=WeChat\n' > "$TMP/data/apps.conf"; rm -f "$TMP/data/config.sh"
j=$(FG=com.tencent.mm run); check "用户覆盖表" "$j" "j['app']=='WeChat'"
j=$(KEYGUARD=true run); check "锁屏" "$j" "j['screen']=='locked' and j['app']=='锁屏'"
j=$(WAKE=Asleep run); check "息屏" "$j" "j['screen']=='off' and j['app']=='锁屏'"
for sh in dash "busybox sh"; do
  j=$(PATH="$TMP/bin:$PATH" MS_MODDIR="$TMP/mod" MS_DATA="$TMP/data" DRY_RUN=1 $sh "$TMP/mod/reporter.sh" 2>/dev/null)
  check "在 $sh 下运行" "$j" "j['app']=='哔哩哔哩'"
done
echo "ALL PASS"
