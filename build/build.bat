@echo off
setlocal enabledelayedexpansion

REM Main build script for JFVM on Windows
REM Usage: build.bat [executable_name] [version]

if "%1"=="" (
    set exe_name=jfvm.exe
) else (
    set exe_name=%1
)

REM Get version information
if not "%2"=="" (
    set version=%2
) else (
    REM Try to get version from git tag
    for /f "tokens=*" %%i in ('git describe --tags --exact-match HEAD 2^>nul') do set version=%%i
    if "!version!"=="" (
        for /f "tokens=*" %%i in ('powershell -Command "Get-Date -Format 'yyyyMMddHHmmss'"') do set version=dev-%%i
    )
)

REM Get build information
for /f "tokens=*" %%i in ('powershell -Command "Get-Date -Format 'yyyy-MM-dd_HH:mm:ss' -AsUTC"') do set build_date=%%i
for /f "tokens=*" %%i in ('git rev-parse --short HEAD 2^>nul') do set git_commit=%%i
if "!git_commit!"=="" set git_commit=unknown

REM Build flags
set ldflags=-w -extldflags "-static"
set ldflags=!ldflags! -X main.Version=!version!
set ldflags=!ldflags! -X main.BuildDate=!build_date!
set ldflags=!ldflags! -X main.GitCommit=!git_commit!

echo Building !exe_name!...
echo   Version: !version!
echo   Build Date: !build_date!
echo   Git Commit: !git_commit!
echo   GOOS: !GOOS!
echo   GOARCH: !GOARCH!

REM Ensure CGO is disabled for static compilation
set CGO_ENABLED=0

REM Build the binary
go build -o "!exe_name!" -ldflags "!ldflags!" main.go

if errorlevel 1 (
    echo Build failed!
    exit /b 1
)

echo The !exe_name! executable was successfully created.

REM Display binary information
for %%A in (!exe_name!) do echo Binary size: %%~zA bytes

endlocal


