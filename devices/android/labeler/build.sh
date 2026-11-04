#!/usr/bin/env bash
# Builds Labeler.java into ../module/bin/labeler.dex.
# Requirements: JDK 11+ (javac --release 8), curl, network access for the first r8 download.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOLS_DIR="$SCRIPT_DIR/.tools"
BUILD_DIR="$SCRIPT_DIR/build"
OUT_DIR="$SCRIPT_DIR/../module/bin"
R8_JAR="$TOOLS_DIR/r8.jar"
MAVEN_BASE="https://dl.google.com/dl/android/maven2/com/android/tools/r8"

mkdir -p "$TOOLS_DIR"

# 1. Fetch r8 (contains d8) once; later runs reuse .tools/r8.jar.
if [[ ! -s "$R8_JAR" ]]; then
    echo "[labeler] resolving latest stable r8 from Google Maven..."
    # Stable versions consist of digits and dots only (skips -alpha/-beta/-dev builds).
    R8_VERSION="$(curl -fsSL "$MAVEN_BASE/maven-metadata.xml" \
        | grep -o '<version>[^<]*</version>' \
        | sed -e 's/<version>//' -e 's/<\/version>//' \
        | grep -E '^[0-9]+(\.[0-9]+)*$' \
        | sort -V | tail -n 1)"
    if [[ -z "$R8_VERSION" ]]; then
        echo "[labeler] failed to resolve r8 version" >&2
        exit 1
    fi
    echo "[labeler] downloading r8 $R8_VERSION"
    curl -fsSL -o "$R8_JAR.tmp" "$MAVEN_BASE/$R8_VERSION/r8-$R8_VERSION.jar"
    mv "$R8_JAR.tmp" "$R8_JAR"
    echo "$R8_VERSION" > "$TOOLS_DIR/r8.version"
fi
echo "[labeler] using r8 $(cat "$TOOLS_DIR/r8.version" 2>/dev/null || echo '(unknown version)')"

# 2. Compile. All Android APIs are reflective, so the JDK alone is enough.
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/classes" "$BUILD_DIR/dex"
javac --release 8 -d "$BUILD_DIR/classes" "$SCRIPT_DIR/src/Labeler.java"

# 3. Dex. Without --lib, d8 may warn about missing library classes; the code only
#    touches java.* types directly, so the warnings are expected and harmless.
java -cp "$R8_JAR" com.android.tools.r8.D8 \
    --release --min-api 26 \
    --output "$BUILD_DIR/dex" \
    "$BUILD_DIR"/classes/*.class

# 4. Install the artifact into the module.
mkdir -p "$OUT_DIR"
cp "$BUILD_DIR/dex/classes.dex" "$OUT_DIR/labeler.dex"
echo "[labeler] wrote $(cd "$OUT_DIR" && pwd)/labeler.dex ($(wc -c < "$OUT_DIR/labeler.dex") bytes)"
