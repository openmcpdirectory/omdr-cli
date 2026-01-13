# Versioning

OMDR CLI uses **Calendar Versioning (CalVer)** with the format `YYYY.MM.DD`.

## Format

- **YYYY** - Full year (e.g., 2026)
- **MM** - Zero-padded month (e.g., 01 for January)
- **DD** - Zero-padded day (e.g., 13)

## Examples

- `2026.01.13` - Release on January 13, 2026
- `2026.01.13.1` - Hotfix on the same day
- `2026.02.01` - Release on February 1, 2026

## Why CalVer?

- **Clear release date** - Users know exactly when a version was released
- **Always incrementing** - No confusion about which version is newer
- **Simple** - No need to decide between major/minor/patch

## Releasing

```bash
# Create release tag with today's date
git tag $(date +%Y.%m.%d)
git push origin $(date +%Y.%m.%d)
```

GitHub Actions automatically builds and publishes to all platforms.

## Development Builds

Development builds show `dev-YYYYMMDD` format:

```bash
omdr version
# Output: dev-20260113 (commit: abc123, built: 2026-01-13T10:30:00Z)
```
