@echo off
REM GLive Initialization Script for Windows
REM This script sets up the development environment

echo.
echo ================================
echo GLive - Initialization Script
echo ================================
echo.

REM Check Go installation
where go >nul 2>nul
if %errorlevel% neq 0 (
    echo X Go is not installed!
    echo Please install Go from https://go.dev/dl/
    echo.
    echo Windows: choco install golang
    exit /b 1
)

echo ✓ Go found
go version

REM Check Node.js installation
where node >nul 2>nul
if %errorlevel% neq 0 (
    echo X Node.js is not installed!
    echo Please install Node.js from https://nodejs.org/
    exit /b 1
)

echo ✓ Node.js found
node --version
echo.

REM Initialize Go modules
echo Initializing Go modules...

echo   - pkg/core
cd pkg\core
go mod tidy

echo   - pkg/agent
cd ..\agent
go mod tidy

echo   - cmd/glive
cd ..\..\cmd\glive
go mod tidy

cd ..\..

echo ✓ Go modules initialized
echo.

REM Install web dependencies
echo Installing web dependencies...
cd web
call npm install
cd ..

echo ✓ Web dependencies installed
echo.

REM Create workspace directory
set WORKSPACE_DIR=%USERPROFILE%\glive-workspace
if not exist "%WORKSPACE_DIR%" (
    mkdir "%WORKSPACE_DIR%"
    echo ✓ Created workspace directory: %WORKSPACE_DIR%
)

REM Create state directory
set STATE_DIR=%USERPROFILE%\.glive
if not exist "%STATE_DIR%" (
    mkdir "%STATE_DIR%"
    echo ✓ Created state directory: %STATE_DIR%
)

echo.
echo ================================
echo Initialization complete!
echo ================================
echo.
echo Next steps:
echo 1. Configure your API key:
echo    cd cmd\glive ^&^& go run main.go config set api-key YOUR_DEEPSEEK_API_KEY
echo.
echo 2. Build the CLI:
echo    cd cmd\glive ^&^& go build -o glive.exe
echo.
echo 3. Run the agent (optional, for web dashboard):
echo    cd pkg\agent ^&^& go run main.go
echo.
echo 4. Run the web dashboard (optional):
echo    cd web ^&^& npm run dev
echo.
echo 5. Test the CLI:
echo    cd cmd\glive ^&^& go run main.go version
echo.

pause
