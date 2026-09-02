#!/usr/bin/env bash
# 以真實 Ebitengine 視窗拍下繁中人物管理選擇項。
#
# 倚天字型是第三方資產，不進 repo：這裡以唯讀掛載從主機路徑帶進容器，
# 路徑可由 ETEN_FONT_DIR 覆寫。沒有字型時遊戲會失敗即關閉，不會默默用英文跑，
# 所以這支腳本拍不到圖就是拍不到，不會拍出一張看起來對的英文畫面。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE="$(cd "$ROOT/../golden-box-remake-engine" && pwd)"
FONT_DIR="${ETEN_FONT_DIR:-/home/anr2/cht/etan_font}"
test -f "$ROOT/Pool of Radiance (1988).zip"
test -f "$FONT_DIR/stdfont.15"

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" --tmpfs /tmp/.X11-unix:rw,mode=1777 \
  -e HOME=/tmp/home -e GOCACHE=/src/workplace/go-build-cache \
  -e GOMODCACHE=/src/workplace/go-mod-cache \
  -v "$ROOT:/src" -v "$ENGINE:/engine:ro" -v "$FONT_DIR:/fonts:ro" -w /src \
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
(cd /tmp && exec /tmp/pool-game -zip "/src/Pool of Radiance (1988).zip" \
   -lang zh -eten-font /fonts/stdfont.15) >/tmp/game.log 2>&1 &
game_pid=$!
retries=0
window=
until test -n "$window"; do
  window=$(xdotool search --name "Pool of Radiance Remake" 2>/dev/null | head -1 || true)
  sleep 0.2
  retries=$((retries + 1))
  if test "$retries" -ge 100; then
    echo "the game window never appeared" >&2
    tail -20 /tmp/game.log >&2 || true
    exit 1
  fi
done
xdotool windowfocus "$window"
eval "$(xdotool getwindowgeometry --shell "$window")"
xdotool mousemove 1190 790
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
sleep 1.0
shot /tmp/title.png
pulse Return
sleep 0.8
shot docs/screenshots/pool-remake-chinese-menu.png
if cmp -s /tmp/title.png docs/screenshots/pool-remake-chinese-menu.png; then
  echo "ENTER did not leave the title screen" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
sha256sum docs/screenshots/pool-remake-chinese-menu.png
'
