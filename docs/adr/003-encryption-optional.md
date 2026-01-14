# 003 - Optional SQLCipher Encryption

## Status
Accepted

## Context
Some users want encrypted local data, while others prefer minimal dependencies and simpler builds. SQLCipher is not always available in all build environments.

## Decision
Support SQLCipher encryption as an optional feature. The app should run with plain SQLite by default and enable encryption when SQLCipher is available.

## Consequences
- Builds without SQLCipher still work and store data unencrypted.
- Users can choose to encrypt, and encryption errors are surfaced clearly.
- The codebase must handle both encrypted and unencrypted workflows.
