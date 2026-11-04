#!/system/bin/sh
# MiaM1ku 设备状态上报（KernelSU / Magisk / APatch）
# 协议：https://github.com/MiaM1ku/MiaM1ku/blob/main/devices/PROTOCOL.md
#
# 用法：
#   reporter.sh                    常驻运行（service.sh 启动）
#   reporter.sh --once             采样并上报一次，打印 JSON 与结果
#   reporter.sh --refresh-labels   先全量刷新应用名缓存（可与 --once 组合）
#   DRY_RUN=1 reporter.sh          采样一次并打印 JSON，跳过上报
# 测试时可用 MS_MODDIR / MS_DATA 覆盖模块目录与数据目录。

MODDIR="${MS_MODDIR:-${0%/*}}"
DATA="${MS_DATA:-/data/adb/miam1ku_status}"
CONF="$DATA/config.sh"
LOG="$DATA/reporter.log"
CACHE="$DATA/labels.cache"
LABELER="$MODDIR/bin/labeler.dex"
UA="miam1ku-status-ksu/1.0"

ONCE=0
REFRESH=0
for arg in "$@"; do
  case "$arg" in
    --once) ONCE=1 ;;
    --refresh-labels) REFRESH=1 ;;
  esac
done
[ "$DRY_RUN" = "1" ] && ONCE=1

mkdir -p "$DATA" 2>/dev/null

# ---------------------------------------------------------------- 日志
log() {
  echo "$(date '+%m-%d %H:%M:%S') $*" >> "$LOG"
  [ "$ONCE" = "1" ] && echo "$*" >&2
}

rotate_log() {
  [ -f "$LOG" ] || return 0
  size=$(wc -c < "$LOG" 2>/dev/null)
  if [ "${size:-0}" -gt 262144 ]; then
    tail -n 100 "$LOG" > "$LOG.tmp" 2>/dev/null && mv -f "$LOG.tmp" "$LOG"
  fi
}

# ---------------------------------------------------------------- 配置
load_config() {
  ENDPOINT="https://miam1ku.cn/api/report"
  TOKEN=""
  DEVICE="phone"
  MODEL=""
  INTERVAL=5
  HEARTBEAT=60
  MIN_INTERVAL=10
  HIDE_APPS=""
  # shellcheck disable=SC1090
  [ -f "$CONF" ] && . "$CONF"
  case "$INTERVAL" in ''|*[!0-9]*) INTERVAL=5 ;; esac
  case "$HEARTBEAT" in ''|*[!0-9]*) HEARTBEAT=60 ;; esac
  case "$MIN_INTERVAL" in ''|*[!0-9]*) MIN_INTERVAL=10 ;; esac
  [ "$INTERVAL" -lt 2 ] && INTERVAL=2
}

# ---------------------------------------------------------------- 采集
# 屏幕：on / off / locked
read_screen() {
  wake=$(dumpsys power 2>/dev/null | grep -m1 'mWakefulness=' | sed 's/.*mWakefulness=//' | tr -d '\r ')
  case "$wake" in
    Asleep*|Dozing*) echo off; return ;;
  esac

  policy=$(dumpsys window policy 2>/dev/null | grep -E 'isKeyguardShowing=|mKeyguardShowing=|KeyguardServiceDelegate|showing=' | head -n 20)
  if [ -n "$policy" ]; then
    if echo "$policy" | grep -qE 'isKeyguardShowing=true|mKeyguardShowing=true|showing=true'; then
      echo locked
    else
      echo on
    fi
    return
  fi

  if dumpsys window 2>/dev/null | grep -qE 'mDreamingLockscreen=true|isStatusBarKeyguard=true|mShowingLockscreen=true'; then
    echo locked
  else
    echo on
  fi
}

# 前台应用包名
read_fg_pkg() {
  line=$(dumpsys activity activities 2>/dev/null | grep -m1 -E 'topResumedActivity|mResumedActivity')
  if [ -z "$line" ]; then
    line=$(dumpsys window 2>/dev/null | grep -m1 -E 'mCurrentFocus=Window|mFocusedApp=')
  fi
  printf '%s\n' "$line" | grep -oE '[A-Za-z0-9_]+(\.[A-Za-z0-9_]+)+/' | head -n 1 | tr -d '/'
}

# 电量：设置 BAT（0-100 或空）与 CHG（true/false）
read_battery() {
  BAT=""
  CHG="false"
  ps_dir=/sys/class/power_supply/battery
  if [ -r "$ps_dir/capacity" ]; then
    BAT=$(tr -dc '0-9' < "$ps_dir/capacity" 2>/dev/null)
    st=$(cat "$ps_dir/status" 2>/dev/null)
    case "$st" in Charging*|Full*) CHG="true" ;; esac
  fi
  if [ -z "$BAT" ]; then
    dump=$(dumpsys battery 2>/dev/null)
    BAT=$(echo "$dump" | sed -n 's/^ *level: *\([0-9]*\).*/\1/p' | head -n 1)
    if echo "$dump" | grep -qE '(AC|USB|Wireless|Dock) powered: true'; then CHG="true"; fi
  fi
}

read_model() {
  if [ -n "$MODEL" ]; then echo "$MODEL"; return; fi
  for prop in ro.product.marketname ro.product.vendor.marketname ro.product.model; do
    v=$(getprop "$prop" 2>/dev/null)
    if [ -n "$v" ]; then echo "$v"; return; fi
  done
}

# ---------------------------------------------------------------- 应用名
LABEL_TRIED=""

# 包名只含字母、数字、下划线和点；把点转义后作为 grep 前缀
pkg_regex() {
  echo "$1" | sed 's/\./\\./g'
}

lookup_label() { # $1=文件 $2=包名
  [ -f "$1" ] || return 1
  hit=$(grep -m1 "^$(pkg_regex "$2")=" "$1" 2>/dev/null | cut -d= -f2- | tr -d '\r')
  [ -n "$hit" ] || return 1
  printf '%s\n' "$hit"
}

run_labeler() {
  [ -f "$LABELER" ] || return 1
  if command -v timeout >/dev/null 2>&1; then
    CLASSPATH="$LABELER" timeout 15 app_process /system/bin Labeler "$@" 2>>"$LOG"
  else
    CLASSPATH="$LABELER" app_process /system/bin Labeler "$@" 2>>"$LOG"
  fi
}

refresh_labels() {
  if run_labeler > "$CACHE.tmp" && [ -s "$CACHE.tmp" ]; then
    mv -f "$CACHE.tmp" "$CACHE"
    log "应用名缓存已刷新：$(wc -l < "$CACHE" | tr -d ' ') 个应用"
  else
    rm -f "$CACHE.tmp"
    log "应用名缓存刷新失败（labeler 不可用或输出为空）"
  fi
}

# 缓存里没有的包名：增量查询，同一包名 10 分钟内最多尝试一次。
# 结果写入 LABEL_OUT；在当前 shell 中调用，保证 LABEL_TRIED 的更新得以保留。
label_incremental() { # $1=包名
  LABEL_OUT=""
  re=$(pkg_regex "$1")
  last=$(printf '%s\n' "$LABEL_TRIED" | grep -m1 "^$re " | cut -d' ' -f2)
  if [ -n "$last" ] && [ $((NOW - last)) -lt 600 ]; then return 1; fi
  LABEL_TRIED=$(printf '%s\n' "$LABEL_TRIED" | grep -v "^$re "; echo "$1 $NOW")
  out=$(run_labeler "$1" | grep -m1 "^$re=" | tr -d '\r')
  [ -n "$out" ] || return 1
  printf '%s\n' "$out" >> "$CACHE"
  LABEL_OUT=$(printf '%s\n' "$out" | cut -d= -f2-)
  [ -n "$LABEL_OUT" ]
}

app_name() { # $1=包名；设置 APP 与 PKG
  PKG="$1"
  for hidden in $HIDE_APPS; do
    if [ "$hidden" = "$1" ]; then APP="隐私应用"; PKG=""; return; fi
  done
  APP=$(lookup_label "$DATA/apps.conf" "$1") && return
  APP=$(lookup_label "$MODDIR/apps.conf" "$1") && return
  APP=$(lookup_label "$CACHE" "$1") && return
  if label_incremental "$1"; then APP="$LABEL_OUT"; return; fi
  APP="$1"
}

# ---------------------------------------------------------------- JSON
json_str() {
  printf '%s' "$1" | tr -d '\000-\037' | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

build_json() {
  j="{\"device\":\"$(json_str "$DEVICE")\""
  j="$j,\"app\":\"$(json_str "$APP")\""
  [ -n "$PKG" ] && j="$j,\"pkg\":\"$(json_str "$PKG")\""
  j="$j,\"screen\":\"$SCREEN\""
  [ -n "$BAT" ] && j="$j,\"battery\":$BAT,\"charging\":$CHG"
  [ -n "$MODEL_STR" ] && j="$j,\"model\":\"$(json_str "$MODEL_STR")\""
  [ -n "$OS_STR" ] && j="$j,\"os\":\"$(json_str "$OS_STR")\""
  printf '%s}\n' "$j"
}

# ---------------------------------------------------------------- HTTP
HTTP=""
HTTP_CHECKED=0

detect_http() {
  HTTP=""
  if [ -x "$MODDIR/bin/curl" ]; then
    HTTP="curl:$MODDIR/bin/curl"
  elif command -v curl >/dev/null 2>&1; then
    HTTP="curl:$(command -v curl)"
  else
    for bb in /data/adb/ksu/bin/busybox /data/adb/magisk/busybox /data/adb/ap/bin/busybox; do
      if [ -x "$bb" ] && "$bb" wget --help 2>&1 | grep -q 'post-data'; then
        HTTP="wget:$bb"
        break
      fi
    done
  fi
  HTTP_CHECKED=$NOW
  if [ -n "$HTTP" ]; then log "HTTP 客户端：$HTTP"; else log "找不到 curl 或支持 --post-data 的 busybox wget，5 分钟后重试"; fi
}

# 发送 $1；成功返回 0，结果写入 SEND_RESULT
send_json() {
  SEND_RESULT=""
  case "$HTTP" in
    curl:*)
      SEND_RESULT=$("${HTTP#curl:}" -sS -m 10 -X POST \
        -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -H "User-Agent: $UA" \
        --data-binary "$1" -w ' HTTP %{http_code}' "$ENDPOINT" 2>&1)
      case "$SEND_RESULT" in *' HTTP 200') return 0 ;; esac
      return 1
      ;;
    wget:*)
      SEND_RESULT=$("${HTTP#wget:}" wget -q -O - -T 10 -U "$UA" \
        --header "Authorization: Bearer $TOKEN" --header 'Content-Type: application/json' \
        --post-data "$1" "$ENDPOINT" 2>&1)
      return $?
      ;;
  esac
  SEND_RESULT="no http client"
  return 1
}

# ---------------------------------------------------------------- 主循环
sample() {
  SCREEN=$(read_screen)
  if [ "$SCREEN" = "on" ]; then
    fg=$(read_fg_pkg)
    [ -z "$fg" ] && fg="$LAST_FG"
    if [ -n "$fg" ]; then
      LAST_FG="$fg"
      app_name "$fg"
    else
      APP="系统界面"
      PKG=""
    fi
  else
    APP="锁屏"
    PKG=""
  fi
  read_battery
}

LAST_FG=""
LAST_SIG=""
LAST_SEND=0
PENDING=0
LAST_FULL=0
TOKEN_WARNED=0

load_config
NOW=$(date +%s)
MODEL_STR=$(read_model)
OS_STR="Android $(getprop ro.build.version.release 2>/dev/null)"
[ "$OS_STR" = "Android " ] && OS_STR=""

if [ "$REFRESH" = "1" ] || [ ! -s "$CACHE" ]; then
  refresh_labels
  LAST_FULL=$NOW
fi

if [ "$ONCE" = "1" ]; then
  sample
  body=$(build_json)
  printf '%s\n' "$body"
  [ "$DRY_RUN" = "1" ] && exit 0
  if [ -z "$TOKEN" ]; then echo "TOKEN 为空，请编辑 $CONF" >&2; exit 1; fi
  detect_http
  if send_json "$body"; then echo "OK $SEND_RESULT"; exit 0; fi
  echo "FAILED $SEND_RESULT" >&2
  exit 1
fi

log "启动 pid=$$"
while true; do
  if [ -f "$MODDIR/disable" ] || [ -f "$MODDIR/remove" ]; then
    sleep 30
    continue
  fi

  load_config
  NOW=$(date +%s)
  rotate_log

  if [ $((NOW - LAST_FULL)) -ge 21600 ]; then
    refresh_labels
    LAST_FULL=$NOW
  fi

  sample
  SIG="$PKG|$APP|$SCREEN|$CHG"
  if [ "$SIG" != "$LAST_SIG" ]; then
    LAST_SIG="$SIG"
    PENDING=1
  fi

  due=0
  if [ "$PENDING" = "1" ] && [ $((NOW - LAST_SEND)) -ge "$MIN_INTERVAL" ]; then due=1; fi
  if [ $((NOW - LAST_SEND)) -ge "$HEARTBEAT" ]; then due=1; fi

  if [ "$due" = "1" ]; then
    if [ -z "$TOKEN" ]; then
      if [ $((NOW - TOKEN_WARNED)) -ge 600 ]; then
        log "TOKEN 为空，跳过上报；请编辑 $CONF"
        TOKEN_WARNED=$NOW
      fi
    else
      if [ -z "$HTTP" ] && [ $((NOW - HTTP_CHECKED)) -ge 300 ]; then detect_http; fi
      if [ -n "$HTTP" ]; then
        body=$(build_json)
        if send_json "$body"; then
          PENDING=0
        else
          log "上报失败：$SEND_RESULT"
        fi
      fi
    fi
    # 失败时同样按 MIN_INTERVAL 节流重试
    LAST_SEND=$NOW
  fi

  if [ "$SCREEN" = "off" ]; then sleep 15; else sleep "$INTERVAL"; fi
done
