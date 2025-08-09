@echo off
REM FortiFi Build Script for Windows
REM This script creates a deployable package with the executable and necessary files

echo Building FortiFi...

REM Create builds directory if it doesn't exist
if not exist "builds" (
    mkdir builds
    echo Created builds directory
)

REM Clean previous build
echo Cleaning previous build...
if exist "builds\*" (
    del /q "builds\*"
    for /d %%i in ("builds\*") do rmdir /s /q "%%i"
)

REM Create raw directory in builds
mkdir "builds\raw"
echo Created raw directory

REM Build the executable
echo Compiling executable...
go build -o "builds\FortiFi.exe" ./cmd/fortifi-cli

REM Copy config template to builds directory
echo Setting up config files...
copy "internal\config\import_config.template.json" "builds\import_config.json"
copy "README.md" "builds\README.md"

echo Build complete! Deployable package created in builds/