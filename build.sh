#!/bin/bash
# ============================================================
# Build + Sign Script for Tarkov Kill Screen Analyzer
# ============================================================

SIGNTOOL="/c/Program Files (x86)/Windows Kits/10/bin/10.0.28000.0/x64/signtool.exe"
THUMBPRINT="9738D348735DCEFE13BAEEB1477B05315FC58767"
TIMESTAMP="http://time.certum.pl"

if [ ! -f "$SIGNTOOL" ]; then
    echo "[ERROR] signtool.exe not found"
    exit 1
fi

BUILD_TYPE="${1:-release}"

case "$BUILD_TYPE" in
    release)
        echo "[BUILD] Release build..."
        OUTFILE="screenshoter.exe"
        TAGS=""
        ;;
    debug)
        echo "[BUILD] Debug build..."
        OUTFILE="screenshoter_debug.exe"
        TAGS="-tags debug"
        ;;
    admin)
        echo "[BUILD] Admin+Debug build..."
        OUTFILE="screenshoter_admin.exe"
        TAGS="-tags admin,debug"
        ;;
    *)
        echo "[ERROR] Unknown build type: $BUILD_TYPE"
        echo "Usage: build.sh [release|debug|admin]"
        exit 1
        ;;
esac

export GOOS=windows
export GOARCH=amd64

echo "[BUILD] go build $TAGS -o $OUTFILE"
go build $TAGS -ldflags "-H=windowsgui -s -w" -o "$OUTFILE"
if [ $? -ne 0 ]; then
    echo "[ERROR] Build failed!"
    exit 1
fi
echo "[BUILD] OK: $OUTFILE"

echo "[SIGN] Signing $OUTFILE..."
"$SIGNTOOL" sign //fd SHA256 //tr "$TIMESTAMP" //td SHA256 //sha1 "$THUMBPRINT" "$OUTFILE"
if [ $? -ne 0 ]; then
    echo "[ERROR] Signing failed! Is SimplySign Desktop running?"
    exit 1
fi

echo "[VERIFY] Checking signature..."
"$SIGNTOOL" verify //pa "$OUTFILE"

echo ""
echo "============================================================"
echo "  Done: $OUTFILE built and signed."
echo "============================================================"
