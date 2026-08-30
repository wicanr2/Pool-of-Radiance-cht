#!/usr/bin/env bash
# 以真實 Ebitengine 視窗與逐鍵輸入重生姓名／肖像／戰鬥圖示建角截圖。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE="$(cd "$ROOT/../golden-box-remake-engine" && pwd)"
UID_NOW="$(id -u)"
GID_NOW="$(id -g)"
test -f "$ROOT/Pool of Radiance (1988).zip"

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  -u "$UID_NOW:$GID_NOW" --tmpfs /tmp/.X11-unix:rw,mode=1777 \
  -e HOME=/tmp/home -e GOCACHE=/src/workplace/go-build-cache \
  -e GOMODCACHE=/src/workplace/go-mod-cache \
  -v "$ROOT:/src" -v "$ENGINE:/engine:ro" -w /src \
  wasteland-go:1.24-x11-record-r1 bash -c '
set -eu
mkdir -p "$HOME" docs/screenshots
Xvfb :99 -screen 0 1200x800x24 >/tmp/xvfb.log 2>&1 &
xvfb_pid=$!
game_pid=
finish() {
  test -z "$game_pid" || kill "$game_pid" 2>/dev/null || true
  kill "$xvfb_pid" 2>/dev/null || true
}
trap finish EXIT
export DISPLAY=:99
until test -S /tmp/.X11-unix/X99; do sleep 0.1; done
cp go.mod /tmp/pool.mod
cp go.sum /tmp/pool.sum
printf "\nreplace github.com/wicanr2/golden-box-remake-engine => /engine\n" >> /tmp/pool.mod
go build -modfile=/tmp/pool.mod -o /tmp/pool-game ./cmd/pool-game
(cd /tmp && exec /tmp/pool-game -zip "/src/Pool of Radiance (1988).zip") >/tmp/game.log 2>&1 &
game_pid=$!
retries=0
window=
until test -n "$window"; do
  window=$(xdotool search --name "Pool of Radiance Remake" 2>/dev/null | head -1 || true)
  sleep 0.2
  retries=$((retries + 1))
  test "$retries" -lt 100
done
xdotool windowfocus "$window"
eval "$(xdotool getwindowgeometry --shell "$window")"
pulse() {
  xdotool keydown "$1"
  sleep 0.18
  xdotool keyup "$1"
  sleep 0.28
}
for key in Return c Return Return Return Return; do
  pulse "$key"
done
sleep 0.8
pulse Return
sleep 0.5
ffmpeg -y -hide_banner -loglevel error -f x11grab -video_size "${WIDTH}x${HEIGHT}" \
  -i ":99+${X},${Y}" -frames:v 1 docs/screenshots/pool-remake-character-name.png
xdotool type --delay 120 HERO
pulse Return
sleep 0.5
ffmpeg -y -hide_banner -loglevel error -f x11grab -video_size "${WIDTH}x${HEIGHT}" \
  -i ":99+${X},${Y}" -frames:v 1 docs/screenshots/pool-remake-portrait-editor.png
pulse k
sleep 0.8
for key in h w p 1 2 s; do
  pulse "$key"
done
sleep 0.5
ffmpeg -y -hide_banner -loglevel error -f x11grab -video_size "${WIDTH}x${HEIGHT}" \
  -i ":99+${X},${Y}" -frames:v 1 docs/screenshots/pool-remake-combat-icon-editor.png
pulse Return
sleep 0.5
ffmpeg -y -hide_banner -loglevel error -f x11grab -video_size "${WIDTH}x${HEIGHT}" \
  -i ":99+${X},${Y}" -frames:v 1 docs/screenshots/pool-remake-icon-confirm.png
pulse y
sleep 0.8
pulse a
sleep 0.8
ffmpeg -y -hide_banner -loglevel error -f x11grab -video_size "${WIDTH}x${HEIGHT}" \
  -i ":99+${X},${Y}" -frames:v 1 docs/screenshots/pool-remake-party-menu.png
sha256sum docs/screenshots/pool-remake-character-name.png \
  docs/screenshots/pool-remake-portrait-editor.png \
  docs/screenshots/pool-remake-combat-icon-editor.png \
  docs/screenshots/pool-remake-icon-confirm.png \
  docs/screenshots/pool-remake-party-menu.png
if cmp -s docs/screenshots/pool-remake-portrait-editor.png docs/screenshots/pool-remake-combat-icon-editor.png; then
  echo "combat icon capture did not leave the portrait screen" >&2
  exit 1
fi
if cmp -s docs/screenshots/pool-remake-combat-icon-editor.png docs/screenshots/pool-remake-icon-confirm.png || \
   cmp -s docs/screenshots/pool-remake-icon-confirm.png docs/screenshots/pool-remake-party-menu.png; then
  echo "creation completion capture did not advance through confirmation and party menu" >&2
  exit 1
fi
'
