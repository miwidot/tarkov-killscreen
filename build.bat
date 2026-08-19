@echo off
setlocal

:: ============================================================
:: Build + Sign Script for Tarkov Kill Screen Analyzer
:: ============================================================

set SIGNTOOL="C:\Program Files (x86)\Windows Kits\10\bin\10.0.28000.0\x64\signtool.exe"
set THUMBPRINT=9738D348735DCEFE13BAEEB1477B05315FC58767
set TIMESTAMP=http://time.certum.pl

:: Check signtool exists
if not exist %SIGNTOOL% (
    echo [ERROR] signtool.exe not found at %SIGNTOOL%
    echo         Install Windows SDK Signing Tools.
    pause
    exit /b 1
)

:: Parse argument
set BUILD_TYPE=%1
if "%BUILD_TYPE%"=="" set BUILD_TYPE=release

if /i "%BUILD_TYPE%"=="release" goto :build_release
if /i "%BUILD_TYPE%"=="debug" goto :build_debug
if /i "%BUILD_TYPE%"=="admin" goto :build_admin
echo [ERROR] Unknown build type: %BUILD_TYPE%
echo Usage: build.bat [release^|debug^|admin]
pause
exit /b 1

:build_release
echo [BUILD] Release build...
set OUTFILE=screenshoter.exe
set TAGS=
set LDFLAGS=-H=windowsgui -s -w
goto :do_build

:build_debug
echo [BUILD] Debug build...
set OUTFILE=screenshoter_debug.exe
set TAGS=-tags debug
set LDFLAGS=-H=windowsgui -s -w
goto :do_build

:build_admin
echo [BUILD] Admin+Debug build...
set OUTFILE=screenshoter_admin.exe
set TAGS=-tags "admin,debug"
set LDFLAGS=-H=windowsgui -s -w
goto :do_build

:do_build
set GOOS=windows
set GOARCH=amd64

:: Version resource is derived from version.go so the PE metadata can never
:: drift away from the version the app reports and updates against.
set VERSION=
for /f "tokens=2 delims==" %%A in ('findstr /c:"CurrentVersion =" version.go') do (
    for /f "tokens=1 delims= " %%B in ("%%A") do set VERSION=%%~B
)
if "%VERSION%"=="" (
    echo [ERROR] Could not read CurrentVersion from version.go
    pause
    exit /b 1
)

set GOWINRES=go-winres
where /q go-winres || set GOWINRES=%USERPROFILE%\go\bin\go-winres.exe

echo [RES] Generating version resource for %VERSION%...

:: The --file-version/--product-version flags cover RT_VERSION but not the
:: manifest's assemblyIdentity, so that one is substituted into a scratch copy.
:: The copy has to stay inside winres\ because icon paths are resolved relative
:: to the json file.
powershell -NoProfile -Command "(Get-Content 'winres\winres.json' -Raw) -replace '\"version\": \"0.0.0.0\"', '\"version\": \"%VERSION%.0\"' | Set-Content -NoNewline 'winres\.build.json'"

%GOWINRES% make --in winres\.build.json --arch amd64 --file-version %VERSION%.0 --product-version %VERSION%.0
set RES_STATUS=%errorlevel%
del /q winres\.build.json 2>nul
if not "%RES_STATUS%"=="0" (
    echo [ERROR] go-winres failed! Install with: go install github.com/tc-hib/go-winres@latest
    pause
    exit /b 1
)

go build %TAGS% -ldflags "%LDFLAGS%" -o %OUTFILE%
if errorlevel 1 (
    echo [ERROR] Build failed!
    pause
    exit /b 1
)
echo [BUILD] OK: %OUTFILE%

:: Sign
echo [SIGN] Signing %OUTFILE%...
%SIGNTOOL% sign /fd SHA256 /tr %TIMESTAMP% /td SHA256 /sha1 %THUMBPRINT% %OUTFILE%
if errorlevel 1 (
    echo [ERROR] Signing failed! Is SimplySign Desktop running?
    pause
    exit /b 1
)

:: Verify
echo [VERIFY] Checking signature...
%SIGNTOOL% verify /pa /v %OUTFILE%
if errorlevel 1 (
    echo [WARN] Verification failed - signature may not chain to a trusted root.
    echo        This is normal for OV certs until SmartScreen builds reputation.
) else (
    echo [VERIFY] OK
)

echo.
echo ============================================================
echo   Done: %OUTFILE% built and signed.
echo ============================================================
pause
