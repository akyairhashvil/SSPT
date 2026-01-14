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
commit="$(git rev-parse --short HEAD)"
build_time="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

export GOCACHE="$root_dir/.gocache"
export GOMODCACHE="$root_dir/.gomodcache"

build_tags=()
run_tests=false
run_race=false
skip_build=false

for arg in "$@"; do
  case "$arg" in
    --sqlcipher)
      export CGO_ENABLED=1
      export CGO_CFLAGS="${CGO_CFLAGS:-} -I/usr/include/sqlcipher"
      export CGO_LDFLAGS="${CGO_LDFLAGS:-} -lsqlcipher"
      build_tags+=("sqlcipher" "libsqlite3")
      ;;
    --test)
      run_tests=true
      ;;
    --race)
      run_tests=true
      run_race=true
      ;;
    --test-only)
      run_tests=true
      skip_build=true
      ;;
    *)
      echo "Unknown flag: $arg" >&2
      echo "Usage: $0 [--sqlcipher] [--test] [--race] [--test-only]" >&2
      exit 2
      ;;
  esac
done

tags_arg=()
if [[ ${#build_tags[@]} -gt 0 ]]; then
  tags_arg=(-tags "$(IFS=,; echo "${build_tags[*]}")")
fi

if [[ "$skip_build" == "false" ]]; then
  go build "${tags_arg[@]}" -ldflags "\
    -X github.com/akyairhashvil/SSPT/internal/tui.AppVersion=$num \
    -X github.com/akyairhashvil/SSPT/internal/tui.GitCommit=$commit \
    -X github.com/akyairhashvil/SSPT/internal/tui.BuildTime=$build_time" \
    -o "$root_dir/sspt" \
    "$root_dir/cmd/app/main.go"
fi

if [[ "$run_tests" == "true" ]]; then
  if [[ "$run_race" == "true" ]]; then
    go test -race ./...
  else
    go test ./...
  fi
fi

if [[ "$skip_build" == "false" ]]; then
  echo "Built sspt (version $num)"
fi
