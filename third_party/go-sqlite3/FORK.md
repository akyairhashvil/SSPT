# go-sqlite3 Fork

This is a fork of github.com/mattn/go-sqlite3 with SQLCipher support.

## Upstream Version
- Base: v1.14.32
- Forked: unknown

## Changes from Upstream
1. Added SQLCipher build tags
2. Modified SQLCipher-related build configuration

## Updating from Upstream
```bash
cd third_party/go-sqlite3
git remote add upstream https://github.com/mattn/go-sqlite3
git fetch upstream
git merge upstream/master
# Resolve conflicts, test, commit
```

## Build Requirements
- libsqlcipher-dev (Debian/Ubuntu)
- sqlcipher (macOS via Homebrew)
