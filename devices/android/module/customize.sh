#!/system/bin/sh
# 刷入时执行（KernelSU / Magisk / APatch 的安装环境会提供 MODPATH 与 ui_print）
DATA=/data/adb/miam1ku_status

set_perm_recursive "$MODPATH" 0 0 0755 0644
set_perm "$MODPATH/reporter.sh" 0 0 0755
set_perm "$MODPATH/service.sh" 0 0 0755
set_perm "$MODPATH/action.sh" 0 0 0755
set_perm "$MODPATH/uninstall.sh" 0 0 0755
[ -f "$MODPATH/bin/curl" ] && set_perm "$MODPATH/bin/curl" 0 0 0755

mkdir -p "$DATA"
chmod 0700 "$DATA"

if [ -f "$DATA/config.sh" ]; then
  ui_print "- 沿用已有配置：$DATA/config.sh"
else
  cp -f "$MODPATH/config.sh.example" "$DATA/config.sh"
  chmod 0600 "$DATA/config.sh"
  ui_print "- 已生成配置：$DATA/config.sh"
  ui_print "- 请在其中填写 TOKEN，保存后下个周期自动生效"
fi

# 刷新一次应用名缓存（新模块路径在重启后生效，这里先清理旧缓存）
rm -f "$DATA/labels.cache"
ui_print "- 日志：$DATA/reporter.log"
ui_print "- 重启后开始上报；也可在模块页点击“执行”立即上报一次"
