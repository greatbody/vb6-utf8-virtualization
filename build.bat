@echo off
echo Building Windows File System Transparent Encoding Conversion Proxy...
go build -o utf8proxy.exe main.go
if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b %errorlevel%
)
echo Build success: utf8proxy.exe
pause
