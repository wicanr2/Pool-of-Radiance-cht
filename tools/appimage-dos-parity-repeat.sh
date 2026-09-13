#!/usr/bin/env bash
# 同一個 AppImage 對同一組 dosgolem 基準連跑三次，逐位元組核對全部 remake 截圖。
# 用法：tools/appimage-dos-parity-repeat.sh <版本> [patch|full-local] [zh|en]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
FLAVOUR="${2:-full-local}"
LANG_MODE="${3:-zh}"
RUN_ROOT="$ROOT/workplace/dos-parity-repeat-$LANG_MODE"
APPIMAGE="$ROOT/dist-all/$VERSION/$FLAVOUR/pool-of-radiance-remake-$VERSION-x86_64.AppImage"
SOURCE_COMMIT="$(git -C "$ROOT" rev-parse HEAD)"

[[ -n "$VERSION" ]] || {
  echo "用法：tools/appimage-dos-parity-repeat.sh <版本> [patch|full-local] [zh|en]" >&2
  exit 2
}
case "$RUN_ROOT" in
  "$ROOT"/workplace/dos-parity-repeat-*) ;;
  *) echo "重跑輸出必須位於 $ROOT/workplace/dos-parity-repeat-*" >&2; exit 2 ;;
esac

test -f "$APPIMAGE"
if [[ "${POOL_PARITY_VERIFY_ONLY:-0}" != 1 ]]; then
  rm -rf "$RUN_ROOT"
  mkdir -p "$RUN_ROOT"
  for run in 1 2 3; do
    POOL_PARITY_OUT="$RUN_ROOT/run-$run" \
      "$ROOT/tools/appimage-dos-parity.sh" "$VERSION" "$FLAVOUR" "$LANG_MODE"
  done
fi

docker run --rm --network none --memory 256m --cpus 1 --pids-limit 64 \
  -u "$(id -u):$(id -g)" -v "$RUN_ROOT:/runs" -v "$APPIMAGE:/game.AppImage:ro" \
  python:3.12-slim python - "$VERSION" "$FLAVOUR" "$LANG_MODE" "$SOURCE_COMMIT" <<'PY'
import hashlib
import json
import pathlib
import sys

version, flavour, language, source_commit = sys.argv[1:]
root = pathlib.Path("/runs")
runs = []
expected = None
for number in range(1, 4):
    directory = root / f"run-{number}"
    files = sorted(directory.glob("remake-*.png"))
    if not files:
        raise SystemExit(f"run-{number} 沒有 remake 截圖")
    hashes = {path.name: hashlib.sha256(path.read_bytes()).hexdigest() for path in files}
    if expected is None:
        expected = hashes
    elif hashes != expected:
        missing = sorted(set(expected) - set(hashes))
        extra = sorted(set(hashes) - set(expected))
        changed = sorted(name for name in set(expected) & set(hashes)
                         if expected[name] != hashes[name])
        raise SystemExit(
            f"run-{number} 截圖不一致：missing={missing}, extra={extra}, changed={changed}")
    manifest = json.dumps(hashes, sort_keys=True, separators=(",", ":")).encode()
    runs.append({
        "run": number,
        "screenshots": len(hashes),
        "manifest_sha256": hashlib.sha256(manifest).hexdigest(),
    })

receipt = {
    "schema": "pool-appimage-parity-reproducibility/1",
    "version": version,
    "flavour": flavour,
    "language": language,
    "source_commit": source_commit,
    "appimage_sha256": hashlib.sha256(pathlib.Path("/game.AppImage").read_bytes()).hexdigest(),
    "screenshot_count_each_run": len(expected),
    "screenshots_sha256": expected,
    "runs": runs,
    "all_screenshots_byte_identical": True,
}
(root / "reproducibility.json").write_text(
    json.dumps(receipt, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print(f"三次各 {len(expected)} 張 remake 截圖逐位元組相同")
PY

echo "重跑收據 → $RUN_ROOT/reproducibility.json"
