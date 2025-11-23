@echo off
setlocal enabledelayedexpansion

:: GLive Installer for Windows (CMD)
:: Usage: curl -sSL https://raw.githubusercontent.com/yasinhessnawi1/glive/main/install.bat -o install.bat && install.bat

set "REPO=yasinhessnawi1/glive"
set "BINARY_NAME=glive.exe"
set "INSTALL_DIR=%LOCALAPPDATA%\glive\bin"

echo Installing GLive...

:: Create install directory
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

:: Detect architecture
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    set "ARCH=amd64"
) else if "%PROCESSOR_ARCHITECTURE%"=="ARM64" (
    set "ARCH=arm64"
) else (
    set "ARCH=386"
)

echo Detected: windows/%ARCH%

:: Check if curl is available
where curl >nul 2>&1
if %errorlevel% neq 0 (
    echo curl is not available. Please install curl or use PowerShell installer.
    echo PowerShell: irm https://raw.githubusercontent.com/yasinhessnawi1/glive/main/install.ps1 ^| iex
    exit /b 1
)

:: Get latest release version
echo Fetching latest release...
for /f "tokens=*" %%i in ('curl -sL "https://api.github.com/repos/%REPO%/releases/latest" ^| findstr /C:"tag_name"') do set "TAG_LINE=%%i"

:: Parse version from tag_name line
set "TAG_LINE=!TAG_LINE:~15!"
for /f "tokens=1 delims=," %%a in ("!TAG_LINE!") do set "VERSION=%%a"
set "VERSION=!VERSION:"=!"

if "!VERSION!"=="" (
    echo No releases found. Building from source...
    goto :build_from_source
)

echo Latest version: !VERSION!

:: Remove 'v' prefix for download URL
set "VERSION_NO_V=!VERSION:v=!"

set "DOWNLOAD_URL=https://github.com/%REPO%/releases/download/!VERSION!/glive_!VERSION_NO_V!_windows_%ARCH%.zip"

echo Downloading from: !DOWNLOAD_URL!

:: Create temp directory
set "TEMP_DIR=%TEMP%\glive_install_%RANDOM%"
mkdir "%TEMP_DIR%"

:: Download
curl -sL "!DOWNLOAD_URL!" -o "%TEMP_DIR%\glive.zip"
if %errorlevel% neq 0 (
    echo Download failed. Building from source...
    rmdir /s /q "%TEMP_DIR%" 2>nul
    goto :build_from_source
)

:: Extract using PowerShell (available on all modern Windows)
powershell -Command "Expand-Archive -Path '%TEMP_DIR%\glive.zip' -DestinationPath '%TEMP_DIR%' -Force"
if %errorlevel% neq 0 (
    echo Extraction failed. Building from source...
    rmdir /s /q "%TEMP_DIR%" 2>nul
    goto :build_from_source
)

:: Find and move binary
for /r "%TEMP_DIR%" %%f in (glive.exe) do (
    copy "%%f" "%INSTALL_DIR%\%BINARY_NAME%" >nul
    goto :installed
)

echo Binary not found in archive. Building from source...
rmdir /s /q "%TEMP_DIR%" 2>nul
goto :build_from_source

:build_from_source
:: Check if Go is installed
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo Go is not installed. Please install Go from https://go.dev/dl/
    exit /b 1
)

set "TEMP_DIR=%TEMP%\glive_build_%RANDOM%"
mkdir "%TEMP_DIR%"
cd /d "%TEMP_DIR%"

echo Cloning repository...
git clone --depth 1 "https://github.com/%REPO%.git"
cd glive\cmd\glive

echo Building...
go build -o %BINARY_NAME%

copy "%BINARY_NAME%" "%INSTALL_DIR%\%BINARY_NAME%" >nul
cd /d "%TEMP%"
rmdir /s /q "%TEMP_DIR%" 2>nul
goto :installed

:installed
:: Cleanup temp if exists
if exist "%TEMP_DIR%" rmdir /s /q "%TEMP_DIR%" 2>nul

:: Check if already in PATH
echo %PATH% | findstr /C:"%INSTALL_DIR%" >nul
if %errorlevel% neq 0 (
    echo.
    echo Adding %INSTALL_DIR% to PATH...
    setx PATH "%PATH%;%INSTALL_DIR%" >nul 2>&1
    set "PATH=%PATH%;%INSTALL_DIR%"
)

echo.
echo ============================================
echo GLive installed successfully!
echo Location: %INSTALL_DIR%\%BINARY_NAME%
echo.
echo Please restart your terminal, then run:
echo   glive --help
echo ============================================

endlocal
