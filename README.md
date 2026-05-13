# Open Speed Test CLI

A single-binary command-line speed test client for **self-hosted [OpenSpeedTest](https://openspeedtest.com)** servers.  
No runtime, no dependencies — just download and run.

```
  ospeedtest  —  OpenSpeedTest CLI  →  http://192.168.1.5:3000

  ✓ Server reachable

  [1/3] Ping Test  (20 samples)
  ✓  avg 1.23 ms  jitter 0.18 ms

  [2/3] Download Test  (10s / 6 threads)
  Download  [████████████████████████████]  100%  942.17 Mbps
  ✓  942.17 Mbps

  [3/3] Upload Test  (10s / 6 threads)
  Upload    [████████████████████████████]  100%  487.33 Mbps
  ✓  487.33 Mbps

  ────────────────────────────────────────────────────────────
    Test          Result            Detail
  ────────────────────────────────────────────────────────────
  📡  Ping (avg)   1.23 ms           min 0.91 ms / max 2.14 ms
  〰   Jitter       0.18 ms           packet loss: 0.0%
  ⬇   Download     942.17 Mbps       187.3 MB in 10.0s via 6 threads
  ⬆   Upload       487.33 Mbps       96.8 MB in 10.0s via 6 threads
  ────────────────────────────────────────────────────────────
    +4% overhead compensation applied (matches browser behaviour)
```

---

## Table of Contents

- [Requirements](#requirements)
- [Download & Install](#download--install)
  - [Linux (x86\_64 / amd64)](#linux-x86_64--amd64)
  - [Linux (ARM64 — Raspberry Pi 4/5, Oracle ARM)](#linux-arm64--raspberry-pi-45-oracle-arm)
  - [Linux (ARMv7 — Raspberry Pi 2/3)](#linux-armv7--raspberry-pi-23)
  - [macOS (Apple Silicon — M1/M2/M3/M4)](#macos-apple-silicon--m1m2m3m4)
  - [macOS (Intel)](#macos-intel)
  - [Windows (x86\_64)](#windows-x86_64)
- [Usage](#usage)
- [Flags](#flags)
- [Examples](#examples)
- [Build from Source](#build-from-source)
- [How It Works](#how-it-works)
- [Server Setup](#server-setup)

---

## Requirements

- A running **OpenSpeedTest server** on your network or the internet.  
  The easiest way is Docker: `docker run -d -p 3000:3000 openspeedtest/latest`  
  See [Server Setup](#server-setup) for full instructions.
- No other requirements on the client side — the binary is fully self-contained.

---

## Download & Install

Go to the [**Releases**](../../releases/latest) page and download the binary for your platform,  
or follow the platform-specific steps below.

---

### Linux (x86\_64 / amd64)

Suitable for: **desktop PCs, laptops, servers, VMs, WSL2**

```bash
# Download
curl -L https://github.com/murtwidi/Open-Speed-Test-CLI/releases/latest/download/ospeedtest-linux-amd64 \
  -o ospeedtest

# Make executable
chmod +x ospeedtest

# (Optional) Install system-wide so you can call it from anywhere
sudo mv ospeedtest /usr/local/bin/ospeedtest

# Run
ospeedtest -host 192.168.1.5
```

---

### Linux (ARM64 — Raspberry Pi 4/5, Oracle ARM)

Suitable for: **Raspberry Pi 4 & 5, Raspberry Pi CM4, Oracle Cloud ARM instances, AWS Graviton**

```bash
# Download
curl -L https://github.com/murtwidi/Open-Speed-Test-CLI/releases/latest/download/ospeedtest-linux-arm64 \
  -o ospeedtest

# Make executable
chmod +x ospeedtest

# (Optional) Install system-wide
sudo mv ospeedtest /usr/local/bin/ospeedtest

# Run
ospeedtest -host 192.168.1.5
```

> **Check your Pi model:**  
> Run `uname -m` — if it prints `aarch64`, use this ARM64 binary.

---

### Linux (ARMv7 — Raspberry Pi 2/3)

Suitable for: **Raspberry Pi 2 & 3 (32-bit OS), older ARM boards**

```bash
# Download
curl -L https://github.com/murtwidi/Open-Speed-Test-CLI/releases/latest/download/ospeedtest-linux-armv7 \
  -o ospeedtest

# Make executable
chmod +x ospeedtest

# (Optional) Install system-wide
sudo mv ospeedtest /usr/local/bin/ospeedtest

# Run
ospeedtest -host 192.168.1.5
```

> **Check your Pi model:**  
> Run `uname -m` — if it prints `armv7l`, use this ARMv7 binary.

---

### macOS (Apple Silicon — M1/M2/M3/M4)

Suitable for: **MacBook Air/Pro with M-series chip, Mac Mini M-series, Mac Studio, Mac Pro M2**

```bash
# Download
curl -L https://github.com/murtwidi/Open-Speed-Test-CLI/releases/latest/download/ospeedtest-darwin-arm64 \
  -o ospeedtest

# Make executable
chmod +x ospeedtest

# Remove macOS quarantine flag (required for unsigned binaries)
xattr -d com.apple.quarantine ospeedtest

# (Optional) Install system-wide
sudo mkdir -p /usr/local/bin
sudo mv ospeedtest /usr/local/bin/ospeedtest

# Run
ospeedtest -host 192.168.1.5
```

> **Gatekeeper prompt:** If macOS shows *"cannot be opened because the developer cannot be verified"*,  
> go to **System Settings → Privacy & Security → Security** and click **"Allow Anyway"**, then re-run.  
> Alternatively, right-click the file in Finder and choose **Open**.

---

### macOS (Intel)

Suitable for: **MacBook / iMac / Mac Mini / Mac Pro with Intel processor (pre-2020)**

```bash
# Download
curl -L https://github.com/murtwidi/Open-Speed-Test-CLI/releases/latest/download/ospeedtest-darwin-amd64 \
  -o ospeedtest

# Make executable
chmod +x ospeedtest

# Remove macOS quarantine flag
xattr -d com.apple.quarantine ospeedtest

# (Optional) Install system-wide
sudo mkdir -p /usr/local/bin
sudo mv ospeedtest /usr/local/bin/ospeedtest

# Run
ospeedtest -host 192.168.1.5
```

---

### Windows (x86\_64)

Suitable for: **Windows 10 / 11, Windows Server 2016+**

**Option A — PowerShell (recommended)**

```powershell
# Download
Invoke-WebRequest -Uri "https://github.com/murtwidi/Open-Speed-Test-CLI/releases/latest/download/ospeedtest-windows-amd64.exe" `
  -OutFile "ospeedtest.exe"

# Run from current folder
.\ospeedtest.exe -host 192.168.1.5
```

**Option B — Add to PATH for global use**

1. Download `ospeedtest-windows-amd64.exe` from the [Releases](../../releases/latest) page.
2. Rename it to `ospeedtest.exe`.
3. Move it to a folder that is in your `PATH`, for example `C:\Windows\System32\` or create a dedicated folder like `C:\tools\` and add it to your PATH:
   - Open **Start → Search → "Edit the system environment variables"**
   - Click **Environment Variables**
   - Under **System variables**, find `Path` → click **Edit → New**
   - Add `C:\tools\` → click **OK**
4. Open a new **Command Prompt** or **PowerShell** window and run:

```cmd
ospeedtest -host 192.168.1.5
```

> **Windows Defender warning:** Windows may flag unsigned executables. Click **More info → Run anyway**  
> or add an exclusion in Windows Security → Virus & threat protection settings.

---

## Usage

```
ospeedtest [flags]
```

Run all three tests (ping, download, upload) against a server:

```bash
ospeedtest -host 192.168.1.5
ospeedtest -host 192.168.1.5 -port 8080
ospeedtest -host http://192.168.1.5:3000
ospeedtest -host https://speedtest.example.com
```

---

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-host` | `localhost` | Server IP address, hostname, or full URL |
| `-port` | `3000` | Server port |
| `-duration` | `10s` | Duration of each test. Accepts Go duration strings: `10s`, `30s`, `1m` |
| `-threads` | `6` | Number of parallel HTTP connections (same default as the browser client) |
| `-ping` | `false` | Run **ping test only** |
| `-download` | `false` | Run **download test only** |
| `-upload` | `false` | Run **upload test only** |
| `-no-color` | `false` | Disable ANSI colour output (useful for logging or CI) |

---

## Examples

```bash
# Full test against a local server
ospeedtest -host 192.168.1.5

# Server on a custom port
ospeedtest -host 192.168.1.5 -port 8080

# Full URL (HTTP or HTTPS)
ospeedtest -host http://192.168.1.5:3000
ospeedtest -host https://speedtest.example.com

# Ping only
ospeedtest -host 192.168.1.5 -ping

# Download only
ospeedtest -host 192.168.1.5 -download

# Upload only
ospeedtest -host 192.168.1.5 -upload

# Longer 30-second test with more threads
ospeedtest -host 192.168.1.5 -duration 30s -threads 12

# Plain text output (no colours) — good for scripts / logging
ospeedtest -host 192.168.1.5 -no-color

# Test a remote server over the internet
ospeedtest -host speedtest.mycompany.com -port 443
```

---

## Build from Source

You need **Go 1.21 or newer** installed. Download it from [go.dev/dl](https://go.dev/dl/).

```bash
# Clone the repository
git clone https://github.com/murtwidi/Open-Speed-Test-CLI.git
cd ospeedtest

# Build for your current platform
go build -o ospeedtest .

# Run
./ospeedtest -host 192.168.1.5
```

**Cross-compile for other platforms:**

```bash
# Linux amd64
GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o ospeedtest-linux-amd64 .

# Linux arm64 (Raspberry Pi 4/5)
GOOS=linux   GOARCH=arm64 go build -ldflags="-s -w" -o ospeedtest-linux-arm64 .

# Linux armv7 (Raspberry Pi 2/3)
GOOS=linux   GOARCH=arm   GOARM=7 go build -ldflags="-s -w" -o ospeedtest-linux-armv7 .

# Windows amd64
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ospeedtest-windows-amd64.exe .

# macOS Intel
GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o ospeedtest-darwin-amd64 .

# macOS Apple Silicon
GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o ospeedtest-darwin-arm64 .
```

---

## How It Works

`ospeedtest` replicates the exact measurement method used by the OpenSpeedTest browser client:

| Phase | Method | Detail |
|-------|--------|--------|
| **Ping** | HTTP `HEAD` to `/downloading` | 20 samples; reports min, avg, max, jitter, packet loss |
| **Download** | HTTP `GET` stream from `/downloading` | 6 parallel connections for `duration` seconds; counts bytes received |
| **Upload** | HTTP `POST` stream to `/upload` | 6 parallel connections for `duration` seconds; streams random data; counts bytes sent |
| **Speed calculation** | `(bytes × 8) / elapsed / 1 000 000` | Then `×1.04` (+4% overhead compensation — identical to browser) |

Results are therefore directly comparable to what you see in the OpenSpeedTest web UI.

---

## Server Setup

You need an **OpenSpeedTest server** to test against. The simplest option is Docker:

```bash
# Pull and run (HTTP on port 3000, HTTPS on port 3001)
docker run -d --restart unless-stopped \
  --name openspeedtest \
  -p 3000:3000 \
  -p 3001:3001 \
  openspeedtest/latest
```

Then point `ospeedtest` at your server:

```bash
ospeedtest -host 192.168.1.5           # HTTP  port 3000 (default)
ospeedtest -host 192.168.1.5 -port 3001  # HTTPS port 3001
```

For other deployment options (bare Nginx, Helm, custom SSL, Let's Encrypt), see the  
[OpenSpeedTest documentation](https://github.com/openspeedtest/Speed-Test).

---

## License

[MIT](LICENSE) — © 2025 ospeedtest contributors  
OpenSpeedTest™ is a separate project by [openspeedtest.com](https://openspeedtest.com); this CLI is an independent client.
