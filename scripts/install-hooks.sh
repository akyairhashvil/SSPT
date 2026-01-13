#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
git -C "${root}" config core.hooksPath .githooks
chmod +x "${root}/.githooks/pre-commit"
echo "Installed git hooks in ${root}/.githooks"
