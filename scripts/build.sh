#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
build_file="$root_dir/build_number.txt"

if [[ ! -f "$build_file" ]]; then
  echo "0" > "$build_file"
fi

read -r num < "$build_file"
if ! [[ "$num" =~ ^[0-9]+$ ]]; then
  num=0
fi

num=$((num + 1))
echo "$num" > "$build_file"

export GOCACHE="$root_dir/.gocache"
export GOMODCACHE="$root_dir/.gomodcache"

go build -ldflags "-X github.com/akyairhashvil/SSPT/internal/tui.AppVersion=$num" \
  -o "$root_dir/sspt" \
  "$root_dir/cmd/app/main.go"

echo "Built sspt (version $num)"
