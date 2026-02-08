# Installation Guide

## npm (macOS/Linux/Windows)

```bash
npm install -g @legacycodehq/sanity
```

## Homebrew (macOS/Linux)

```bash
brew install LegacyCodeHQ/tap/sanity
```

## Download Pre-built Binary

Download the latest release for your platform from
the [releases page](https://github.com/LegacyCodeHQ/sanity/releases/latest).

### macOS

```bash
# For Apple Silicon (M1/M2/M3)
curl -L https://github.com/LegacyCodeHQ/sanity/releases/latest/download/sanity_VERSION_darwin_arm64.tar.gz | tar xz

# For Intel Macs
curl -L https://github.com/LegacyCodeHQ/sanity/releases/latest/download/sanity_VERSION_darwin_amd64.tar.gz | tar xz

# Move to PATH
sudo mv sanity /usr/local/bin/

# Verify installation
sanity --version
```

Replace `VERSION` with the actual version number (e.g., `0.2.1`).

### Linux

```bash
# For ARM64
curl -L https://github.com/LegacyCodeHQ/sanity/releases/latest/download/sanity_VERSION_linux_arm64.tar.gz | tar xz

# For AMD64/x86_64
curl -L https://github.com/LegacyCodeHQ/sanity/releases/latest/download/sanity_VERSION_linux_amd64.tar.gz | tar xz

# Move to PATH
sudo mv sanity /usr/local/bin/

# Verify installation
sanity --version
```

Replace `VERSION` with the actual version number (e.g., `0.2.1`).

### Windows

1. Download the Windows zip file from the [releases page](https://github.com/LegacyCodeHQ/sanity/releases/latest)
2. Extract the zip file
3. Add the directory containing `sanity.exe` to your PATH

## Build from Source

Requires Go 1.21+ and CGO enabled:

```bash
git clone https://github.com/LegacyCodeHQ/sanity.git
cd sanity
make build-dev
sudo mv sanity /usr/local/bin/
```

## Go Install

If you have Go installed:

```bash
go install github.com/LegacyCodeHQ/sanity@latest
```
