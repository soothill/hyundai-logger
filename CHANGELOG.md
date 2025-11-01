# Changelog

All notable changes to Hyundai Logger will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Multi-architecture Docker image support (amd64, arm64, arm/v7)
- Native Go cross-compilation for 20x faster ARM builds
- Optimized build system using BUILDPLATFORM
- Docker Hub push automation with `make docker-push`
- Automatic Docker/Podman runtime detection with permissions checking
- Comprehensive Docker deployment documentation

### Changed
- Optimized Dockerfile for faster multi-platform builds
- Improved Makefile with intelligent build system detection
- Enhanced error messages for Docker permission issues

### Performance
- ARM v7 builds reduced from 20+ minutes to ~1 minute
- Total multi-arch build time: ~2 minutes (vs 40+ minutes previously)

## [1.0.0] - 2025-01-01

### Added
- Initial release
- Hyundai Bluelink API integration (US, CA, EU regions)
- Intelligent time-based polling with configurable schedules
- Enhanced charging detection with increased polling frequency
- Exponential backoff retry logic for network resilience
- Email alerting for extended problem periods
- Rate limiting to protect vehicle 12V battery
- Comprehensive vehicle data logging:
  - Engine status and range
  - Climate control settings
  - Door lock status
  - Battery voltage and level
  - Tire pressure monitoring
  - Fuel level and odometer
  - EV-specific data (battery level, charging status, range)
  - GPS location tracking
- InfluxDB v2 integration with 90-day retention
- Automatic bucket creation
- Graceful shutdown handling
- YAML and environment variable configuration
- Docker Compose deployment setup
- Grafana dashboard integration
- Log rotation support
- Systemd service configuration

### Security
- Non-root container execution
- Read-only configuration mounts
- Secure credential management via .env

---

## Version History Format

### Added
- New features that have been added

### Changed
- Changes in existing functionality

### Deprecated
- Features that will be removed in upcoming releases

### Removed
- Features that have been removed

### Fixed
- Bug fixes

### Security
- Security improvements and vulnerability fixes

### Performance
- Performance improvements
