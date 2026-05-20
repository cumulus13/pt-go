@echo off

echo Building pt.exe for Windows...

for /f "tokens=* delims=" %%a in ('date /t') do set current_date=%%a
for /f "tokens=* delims=" %%b in ('time /t') do set current_time=%%b

REM Clean previous build
if exist pt.exe del pt.exe

REM Build
:: pt -b main.go -c
:: pt -b monitor.go -c
:: pt -b pdiff2.go -c
:: pt -b unix.go -c
:: pt -b windows.go -c
for /f "delims=" %%i in ('dir /b pt\*.go') do (
    echo "%%i"
    pt -b %%i -c
)

go build -v -ldflags="-s -w" -o pt.exe ./pt
:: call gobuild -v --ldflags="-s -w" -o pt.exe ./pt

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✅ Build successful: pt.exe
    sendgrowl.exe "GoBuilder" build "SUCCESS %current_date% %current_time%" "[%current_date% %current_time%] ✅ Build successful: pt.exe" -i "%~dp0notif.png" -H 127.0.0.1
    echo.
    dir pt.exe
    cpa pt.exe c:\TOOLS\EXE\ -c
    copy /y VERSION %USERPROFILE%\.pt
) else (
    echo.
    echo ❌ Build failed!
    sendgrowl.exe "GoBuilder" build "FAILED %current_date% %current_time%" "[%current_date% %current_time%] ❌ Build failed!: pt.exe" -i "%~dp0notif.png" -H 127.0.0.1
    exit /b 1
)