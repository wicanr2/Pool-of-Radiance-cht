#!/usr/bin/env bash
# 以真實 Ebitengine 視窗與逐鍵輸入拍下戰術地圖預覽畫面。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE="$(cd "$ROOT/../golden-box-remake-engine" && pwd)"
UID_NOW="$(id -u)"
GID_NOW="$(id -g)"
test -f "$ROOT/Pool of Radiance (1988).zip"

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$UID_NOW:$GID_NOW" --tmpfs /tmp/.X11-unix:rw,mode=1777 \
  -e HOME=/tmp/home -e GOCACHE=/src/workplace/go-build-cache \
  -e GOMODCACHE=/src/workplace/go-mod-cache \
  -v "$ROOT:/src" -v "$ENGINE:/engine:ro" -w /src \
  wasteland-go:1.24-x11-record-r1 bash -c '
set -eu
mkdir -p "$HOME" docs/screenshots
Xvfb :99 -screen 0 1400x900x24 >/tmp/xvfb.log 2>&1 &
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
# 滑鼠移到螢幕右下角，**要在遊戲視窗外面**：視窗現在是 1280x800，
# 舊的 (1190,790) 會落在視窗裡。
xdotool mousemove 1390 890
pulse() {
  xdotool keydown "$1"
  sleep 0.18
  xdotool keyup "$1"
  sleep 0.28
}
shot() {
  ffmpeg -y -hide_banner -loglevel error -f x11grab -video_size "${WIDTH}x${HEIGHT}" \
    -i ":99+${X},${Y}" -frames:v 1 "$1"
}
for key in Return c Return Return Return Return; do
  pulse "$key"
done
sleep 0.8
pulse Return
sleep 0.5
xdotool type --delay 120 HERO
pulse Return
sleep 0.5
pulse k
sleep 0.8
for key in h w p 1 2 s; do
  pulse "$key"
done
sleep 0.5
pulse Return
sleep 0.5
pulse y
sleep 0.8
pulse a
sleep 0.8
pulse b
sleep 1.2
shot /tmp/before-tactical.png
pulse F5
sleep 0.8
shot /tmp/tactical-fresh.png
if cmp -s /tmp/before-tactical.png /tmp/tactical-fresh.png; then
  echo "F5 did not change the screen" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
for key in m m i; do
  pulse "$key"
done
sleep 0.6
shot docs/screenshots/pool-remake-tactical-preview.png
if cmp -s /tmp/tactical-fresh.png docs/screenshots/pool-remake-tactical-preview.png; then
  echo "the movement keys did not change the tactical screen" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
pulse Return
sleep 0.6
shot /tmp/tactical-next-round.png
if cmp -s docs/screenshots/pool-remake-tactical-preview.png /tmp/tactical-next-round.png; then
  echo "ending the turn did not advance the round" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
sha256sum docs/screenshots/pool-remake-tactical-preview.png
'
