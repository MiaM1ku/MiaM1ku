#!/bin/sh
# 打包 KernelSU 模块：dist/miam1ku_status-<version>.zip，并生成填好 TOKEN 的 dist/config.sh
set -eu
HERE=$(cd "$(dirname "$0")" && pwd)
MOD="$HERE/module"
DIST="$HERE/dist"
TOKEN_FILE="$HERE/../../.secrets/device-token"

[ -f "$MOD/bin/labeler.dex" ] || sh "$HERE/labeler/build.sh"

VERSION=$(sed -n 's/^version=//p' "$MOD/module.prop")
mkdir -p "$DIST"
ZIP="$DIST/miam1ku_status-$VERSION.zip"
rm -f "$ZIP"

# 脚本统一为 LF 换行
for f in "$MOD"/*.sh "$MOD"/*.conf "$MOD"/module.prop "$MOD"/config.sh.example; do
  sed -i 's/\r$//' "$f"
done

python3 - "$MOD" "$ZIP" <<'EOF'
import os, sys, zipfile
src, out = sys.argv[1], sys.argv[2]
with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED) as z:
    for root, _, files in os.walk(src):
        for name in sorted(files):
            path = os.path.join(root, name)
            rel = os.path.relpath(path, src)
            info = zipfile.ZipInfo(rel)
            info.compress_type = zipfile.ZIP_DEFLATED
            mode = 0o755 if name.endswith(".sh") or rel in ("bin/curl", "META-INF/com/google/android/update-binary") else 0o644
            info.external_attr = (0o100000 | mode) << 16
            with open(path, "rb") as f:
                z.writestr(info, f.read())
EOF

if [ -f "$TOKEN_FILE" ]; then
  token=$(tr -d '\r\n' < "$TOKEN_FILE")
  sed "s|^TOKEN=\"\"|TOKEN=\"$token\"|" "$MOD/config.sh.example" > "$DIST/config.sh"
  chmod 600 "$DIST/config.sh"
  echo "config: $DIST/config.sh"
fi

echo "zip: $ZIP ($(wc -c < "$ZIP") bytes)"
