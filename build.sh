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

# Version resource is derived from version.go so the PE metadata can never
# drift away from the version the app reports and updates against.
VERSION=$(sed -n 's/.*CurrentVersion = "\([^"]*\)".*/\1/p' version.go)
if [ -z "$VERSION" ]; then
    echo "[ERROR] Could not read CurrentVersion from version.go"
    exit 1
fi

GO_WINRES="go-winres"
if ! command -v "$GO_WINRES" >/dev/null 2>&1; then
    GO_WINRES="$(go env GOPATH)/bin/go-winres"
fi

echo "[RES] Generating version resource for $VERSION..."

# The --file-version/--product-version flags cover RT_VERSION but not the
# manifest's assemblyIdentity, so that one is substituted into a scratch copy.
# The copy has to stay inside winres/ because icon paths are resolved relative
# to the json file.
BUILD_JSON="winres/.build.json"
sed "s/\"version\": \"0.0.0.0\"/\"version\": \"$VERSION.0\"/" winres/winres.json > "$BUILD_JSON"

"$GO_WINRES" make --in "$BUILD_JSON" --arch amd64 --file-version "$VERSION.0" --product-version "$VERSION.0"
RES_STATUS=$?
rm -f "$BUILD_JSON"
if [ $RES_STATUS -ne 0 ]; then
    echo "[ERROR] go-winres failed! Install with: go install github.com/tc-hib/go-winres@latest"
    exit 1
fi

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
