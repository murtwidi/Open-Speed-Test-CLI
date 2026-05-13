# Changelog

## [v1.0.0] — Initial Release

### Added
- Ping test with min / avg / max / jitter / packet-loss metrics
- Download test with live progress bar and real-time speed estimate
- Upload test with live progress bar and real-time speed estimate
- 6 parallel connections per test (matches OpenSpeedTest browser default)
- +4% overhead compensation (matches OpenSpeedTest browser behaviour)
- ANSI colour output with `--no-color` flag to disable
- Single-flag mode: `-ping`, `-download`, `-upload`
- Configurable test duration (`-duration`) and thread count (`-threads`)
- Pre-built binaries for Linux (amd64, arm64, armv7), Windows (amd64), macOS (amd64, arm64)
