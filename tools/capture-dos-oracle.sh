#!/usr/bin/env bash
# 由固定 DOS ZIP 重生標題與主選單 oracle；所有工作均在一次性 Docker 內。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXTRACTED="$ROOT/workplace/oracle/dos"
OUT="$ROOT/docs/reference/original-dos"
UID_NOW="$(id -u)"
GID_NOW="$(id -g)"
test -f "$ROOT/Pool of Radiance (1988).zip"
mkdir -p "$EXTRACTED" "$OUT"

docker run --rm --network none --memory 512m --cpus 1 --pids-limit 64 \
  -u "$UID_NOW:$GID_NOW" -v "$ROOT:/pool" python:3.12-slim python -c '
import pathlib, zipfile
root = pathlib.Path("/pool")
out = root / "workplace/oracle/dos"
with zipfile.ZipFile(root / "Pool of Radiance (1988).zip") as archive:
    for entry in archive.infolist():
        if entry.is_dir():
            continue
        relative = pathlib.PurePosixPath(entry.filename).relative_to("poolrad")
        target = out / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(archive.read(entry))
'

docker run --rm --network none --memory 1g --cpus 1 --pids-limit 128 \
  -u "$UID_NOW:$GID_NOW" \
  --tmpfs /tmp/.X11-unix:rw,mode=1777 --tmpfs /run-game:rw,size=64m \
  -e HOME=/tmp/home -v "$EXTRACTED:/source:ro" -v "$ROOT:/repo" \
  dosbox-run:latest sh -c '
set -eu
mkdir -p "$HOME" /repo/docs/reference/original-dos /run-game/POOLRAD
cp -R /source/. /run-game/POOLRAD/
Xvfb :99 -screen 0 800x600x24 >/tmp/xvfb.log 2>&1 &
xvfb=$!
trap "kill $xvfb 2>/dev/null || true" EXIT
until test -S /tmp/.X11-unix/X99; do sleep 0.1; done
DISPLAY=:99 dosbox -c "mount c /run-game" -c "c:" -c "cd POOLRAD" -c "start" \
  >/tmp/dosbox.log 2>&1 &
dosbox_pid=$!
sleep 3
await_colorful_title() {
  probe=/tmp/title-probe.png
  retries=0
  while :; do
    DISPLAY=:99 import -window root "$probe"
    colors=$(identify -format "%k" "$probe")
    if test "$colors" -ge 10; then
      break
    fi
    sleep 1
    retries=$((retries + 1))
    test "$retries" -lt 30 || exit 1
  done
}
stable_capture() {
  name=$1
  previous=/tmp/${name}-previous.png
  current=/tmp/${name}-current.png
  DISPLAY=:99 import -window root "$previous"
  retries=0
  while :; do
    sleep 1
    DISPLAY=:99 import -window root "$current"
    changed=$(compare -metric AE "$previous" "$current" null: 2>&1 || true)
    if test "$changed" = 0; then
      convert "$current" -crop 640x400+0+0 +repage \
        "/repo/docs/reference/original-dos/$name.png"
      break
    fi
    cp "$current" "$previous"
    retries=$((retries + 1))
    test "$retries" -lt 30 || exit 1
  done
}
await_colorful_title
stable_capture title-intro
window=$(DISPLAY=:99 xdotool search --name DOSBox | head -1)
DISPLAY=:99 xdotool windowfocus "$window"
for _ in $(seq 1 18); do
  DISPLAY=:99 xdotool key space
  sleep 0.5
done
DISPLAY=:99 xdotool key Return
sleep 2
stable_capture main-menu
kill "$dosbox_pid" 2>/dev/null || true
wait "$dosbox_pid" 2>/dev/null || true
'
sha256sum "$OUT/title-intro.png" "$OUT/main-menu.png"
