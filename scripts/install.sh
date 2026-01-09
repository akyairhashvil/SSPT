#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bin_name="sspt"
src_bin="$root_dir/$bin_name"

usage() {
  cat <<'EOF'
Usage: scripts/install.sh [--user] [--prefix DIR] [--no-build] [--sqlcipher]

Options:
  --user         Install to ~/.local/bin (no sudo).
  --prefix DIR   Install to DIR/bin (defaults to /usr/local).
  --no-build     Skip build step and install existing ./sspt.
  --sqlcipher    Build with SQLCipher support.
EOF
}

prefix="/usr/local"
do_build=1
build_flags=()
install_dir=""

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
    --no-build)
      do_build=0
      shift
      ;;
    --sqlcipher)
      build_flags+=("--sqlcipher")
      shift
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

install_dir="$prefix/bin"

if [[ $do_build -eq 1 ]]; then
  "$root_dir/scripts/build.sh" "${build_flags[@]}"
fi

if [[ ! -f "$src_bin" ]]; then
  echo "Binary not found at $src_bin. Build first or pass --no-build." >&2
  exit 1
fi

install_cmd=(install -m 0755 "$src_bin" "$install_dir/$bin_name")
if [[ "$install_dir" == /usr/* ]]; then
  mkdir_cmd=(sudo mkdir -p "$install_dir")
  install_cmd=(sudo "${install_cmd[@]}")
else
  mkdir_cmd=(mkdir -p "$install_dir")
fi
"${mkdir_cmd[@]}"
"${install_cmd[@]}"
echo "Installed $bin_name to $install_dir/$bin_name"
