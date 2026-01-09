#!/usr/bin/env bash
set -euo pipefail

bin_name="sspt"

usage() {
  cat <<'EOF'
Usage: scripts/uninstall.sh [--user] [--prefix DIR]

Options:
  --user         Uninstall from ~/.local/bin (no sudo).
  --prefix DIR   Uninstall from DIR/bin (defaults to /usr/local).
EOF
}

prefix="/usr/local"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --user)
      prefix="$HOME/.local"
      shift
      ;;
    --prefix)
      prefix="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage
      exit 1
      ;;
  esac
done

install_path="$prefix/bin/$bin_name"

if [[ ! -e "$install_path" ]]; then
  echo "No installed binary at $install_path"
  exit 0
fi

rm_cmd=(rm -f "$install_path")
if [[ "$prefix" == /usr/* ]]; then
  rm_cmd=(sudo "${rm_cmd[@]}")
fi

"${rm_cmd[@]}"
echo "Removed $install_path"
