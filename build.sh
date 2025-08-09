#!/bin/bash

# FortiFi Build Script
# This script creates a deployable package with the executable and necessary files

set -e  # Exit on any error

echo "Building FortiFi..."

# Create builds directory if it doesn't exist
BUILD_DIR="builds"
if [ ! -d "$BUILD_DIR" ]; then
    mkdir -p "$BUILD_DIR"
    echo "Created builds directory"
fi

# Clean previous build
echo "Cleaning previous build..."
rm -rf "$BUILD_DIR"/*

# Create raw directory in builds
mkdir -p "$BUILD_DIR/raw"
echo "Created raw directory"

# Get version info
VERSION=$(grep 'Version = ' internal/types/version.go | cut -d'"' -f2)

# Build the executable with version info
echo "Compiling executable..."
go build -ldflags "-X github.com/HadeZForge/FortiFi/internal/types.Version=$VERSION" -o "$BUILD_DIR/FortiFi" ./cmd/fortifi-cli

# Copy config template to builds directory
echo "Setting up config files..."
cp "internal\config\import_config.template.json" "$BUILD_DIR/import_config.json"
cp "README.md" "$BUILD_DIR/README.md"


echo "Build complete! Deployable package created in $BUILD_DIR/"