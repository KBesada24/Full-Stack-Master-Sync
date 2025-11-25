@echo off
REM Full Stack Master Sync Backend - Startup Script (Windows)

echo 🚀 Starting Full Stack Master Sync Backend

REM Check if .env file exists
if not exist ".env" (
    echo ⚠️  No .env file found. Creating from .env.example...
    if exist ".env.example" (
        copy .env.example .env
        echo ✅ Created .env file from .env.example
        echo ⚠️  Please update .env with your configuration
    ) else (
        echo ❌ No .env.example file found. Please create .env manually.
        exit /b 1
    )
)

REM Set default environment variables
if not defined PORT set PORT=8080
if not defined ENVIRONMENT set ENVIRONMENT=development
if not defined LOG_LEVEL set LOG_LEVEL=info

echo 📋 Configuration:
echo    Port: %PORT%
echo    Environment: %ENVIRONMENT%
echo    Log Level: %LOG_LEVEL%

REM Check if binary exists, if not build it
if not exist "full-stack-sync-backend.exe" (
    echo 📦 Building application...
    go build -o full-stack-sync-backend.exe .
    echo ✅ Build complete
)

REM Start the server
echo 🌐 Starting server on port %PORT%...
full-stack-sync-backend.exe
