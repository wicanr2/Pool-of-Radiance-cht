#!/usr/bin/env bash
# 拿發行包拍 F1 說明頁與作弊選單（spec 141）：真的 Ebitengine 視窗、逐鍵輸入、
# 畫面識別字等畫面（-screen-state），ffmpeg 抓視窗。
# 用法：tools/capture-help-cheats.sh <版本> [zh|en] [out 目錄]
#   CAPTURE_CHEATS=0 只拍說明頁（作弊選單還沒做的版本用）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"; LANG_MODE="${2:-zh}"
[[ -n "$VERSION" ]] || { echo "用法：tools/capture-help-cheats.sh <版本> [zh|en] [out]" >&2; exit 2; }
APPIMAGE="$ROOT/dist-all/$VERSION/patch/pool-of-radiance-remake-$VERSION-x86_64.AppImage"
FONT_DIR="${ETEN_FONT_DIR:-/home/anr2/cht/etan_font}"
OUT="${3:-$ROOT/workplace/capture-help-cheats-$LANG_MODE}"
test -f "$APPIMAGE" || { echo "找不到 $APPIMAGE" >&2; exit 2; }
test -f "$ROOT/Pool of Radiance (1988).zip"
test -d "$FONT_DIR"
mkdir -p "$OUT"
# docker -v 要絕對路徑：相對路徑會被當成具名 volume 而拒跑。
OUT="$(cd "$OUT" && pwd)"
docker run --rm --network none --memory 3g --cpus 2 --pids-limit 384 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" --tmpfs /tmp/.X11-unix:rw,mode=1777 \
  -e HOME=/tmp/home -e LANG_MODE="$LANG_MODE" -e CAPTURE_CHEATS="${CAPTURE_CHEATS:-1}" \
  -v "$APPIMAGE:/game.AppImage:ro" -v "$FONT_DIR:/fonts:ro" \
  -v "$ROOT/Pool of Radiance (1988).zip:/zip/pool.zip:ro" -v "$OUT:/out" -w /tmp \
  wasteland-go:1.24-x11-record-r1 bash -c '
set -eu
mkdir -p "$HOME" /tmp/run
cd /tmp/run
cp /game.AppImage ./game.AppImage
./game.AppImage --appimage-extract >/tmp/extract.log 2>&1
Xvfb :99 -screen 0 1400x900x24 >/tmp/xvfb.log 2>&1 &
xvfb=$!
game=
finish() { test -z "$game" || kill "$game" 2>/dev/null || true; kill "$xvfb" 2>/dev/null || true; }
trap finish EXIT
export DISPLAY=:99
until test -S /tmp/.X11-unix/X99; do sleep 0.1; done
STATE=/tmp/pool-screen
if test "$LANG_MODE" = zh; then set -- -lang zh -eten-font /fonts/stdfont.15; else set -- -lang en; fi
(exec ./squashfs-root/AppRun -zip /zip/pool.zip -screen-state "$STATE" -dice-seed 136 "$@") >/tmp/game.log 2>&1 &
game=$!
n=0; window=
until test -n "$window"; do
  window=$(xdotool search --name "Pool of Radiance Remake" 2>/dev/null | head -1 || true)
  sleep 0.2; n=$((n+1)); test "$n" -lt 150 || { echo "視窗沒有出現" >&2; exit 1; }
done
xdotool windowfocus "$window"
eval "$(xdotool getwindowgeometry --shell "$window")"
xdotool mousemove 1390 890
screen() { cat "$STATE" 2>/dev/null | tr -d "\n"; }
die() { echo "$1；目前畫面：$(screen)" >&2; tail -20 /tmp/game.log >&2 || true; exit 1; }
await() { want=$1; rounds=${2:-150}; n=0; while test "$(screen)" != "$want"; do sleep 0.1; n=$((n+1)); test "$n" -lt "$rounds" || die "等不到畫面 $want"; done; }
pulse() { xdotool keydown "$1"; sleep 0.12; xdotool keyup "$1"; sleep 0.12; }
step() {
  key=$1; want=$2; limit=${3:-40}; attempt=0
  while test "$(screen)" != "$want"; do
    attempt=$((attempt+1)); test "$attempt" -le "$limit" || die "按 $key 走不到 $want"
    pulse "$key"; waited=0
    while test "$(screen)" != "$want" && test "$waited" -lt 15; do sleep 0.1; waited=$((waited+1)); done
  done
}
shot() { ffmpeg -y -hide_banner -loglevel error -f x11grab -video_size "${WIDTH}x${HEIGHT}" -i ":99+${X},${Y}" -frames:v 1 "/out/$1.png"; }
await title
step Return menu
step c creation-race
for unused in 1 2 3 4 5; do pulse End; done
step Return creation-class
step Return creation-alignment
step Return creation-roll
step Return creation-name
xdotool type --delay 120 HERO
step Return creation-portrait
step k creation-icon-0
step e creation-icon-confirm
step y menu
pulse a
step b adventure-intro
step Return adventure-move 500
sleep 0.5
step F1 help
sleep 0.5
shot help
step Right help-2
sleep 0.5
shot help-2
step F1 adventure-move
if test "$CAPTURE_CHEATS" = 1; then
  step F6 cheat-menu
  sleep 0.4
  shot cheat-menu-off
  pulse l
  pulse o
  sleep 0.4
  shot cheat-menu-on
  step Escape adventure-move
  sleep 0.5
  shot adventure-cheats-marked
  step F5 tactical
  sleep 0.6
  shot tactical-cheats-marked
fi
echo ok
'
echo "截圖 → $OUT"
