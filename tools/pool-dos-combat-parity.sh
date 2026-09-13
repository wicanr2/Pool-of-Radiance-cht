#!/usr/bin/env bash
# 以正式基準版本的 dosgolem 重生第一場戰鬥的數值收據。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE="$ROOT/workplace/oracle/dos"
PROBE="$ROOT/tools/pool-dos-combat-parity"
DOSGOLEM_SOURCE="${DOSGOLEM_DIR:-/home/anr2/cht/dosgolem}"
REVISION="d351681ba86d97aab571d00b979c36e2336486f3"
OUTPUT="${1:-$ROOT/workplace/dos-combat-parity.json}"
case "$OUTPUT" in
  /*) ;;
  *) OUTPUT="$ROOT/$OUTPUT" ;;
esac
OUTPUT_DIR="${OUTPUT%/*}"
OUTPUT_NAME="${OUTPUT##*/}"

test -f "$SOURCE/start.exe" || { echo "缺少 $SOURCE/start.exe" >&2; exit 2; }
test -f "$PROBE/main.go" || { echo "缺少 $PROBE/main.go" >&2; exit 2; }
test -f "$DOSGOLEM_SOURCE/go.mod" || { echo "找不到 dosgolem：$DOSGOLEM_SOURCE" >&2; exit 2; }
test -d "$OUTPUT_DIR" || { echo "輸出目錄不存在：$OUTPUT_DIR" >&2; exit 2; }

TMP="$(mktemp -d /tmp/pool-dos-combat-parity-XXXXXX)"
trap 'rm -rf "${TMP:?}"' EXIT
git clone --quiet --no-hardlinks "$DOSGOLEM_SOURCE" "$TMP/dosgolem"
git -C "$TMP/dosgolem" checkout --quiet --detach "$REVISION"
mkdir -p "$TMP/scratch" "$TMP/gocache" "$TMP/gomodcache"

timeout 600 docker run --rm --network none --memory 4g --cpus 2 --pids-limit 256 \
  -u "$(id -u):$(id -g)" -e HOME=/tmp -e GOCACHE=/gocache -e GOMODCACHE=/gomodcache \
  -v "$PROBE:/probe:ro" -v "$TMP/dosgolem:/dosgolem:ro" \
  -v "$TMP/gocache:/gocache" -v "$TMP/gomodcache:/gomodcache" \
  -v "$SOURCE:/orig:ro" -v "$TMP/scratch:/scratch" -v "$OUTPUT_DIR:/out" \
  -w /probe golang:1.24-bookworm go run . \
  -revision "$REVISION" -out "/out/$OUTPUT_NAME"

echo "戰鬥對拍收據 → $OUTPUT"
