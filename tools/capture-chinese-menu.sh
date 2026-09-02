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
# C 進建角：種族 → 性別 → 職業，各拍一張能證明選單項目也是中文的。
pulse c
sleep 0.6
shot docs/screenshots/pool-remake-chinese-race.png
if cmp -s docs/screenshots/pool-remake-chinese-menu.png docs/screenshots/pool-remake-chinese-race.png; then
  echo "C did not open character creation" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
for key in Return Return; do
  pulse "$key"
done
sleep 0.6
shot docs/screenshots/pool-remake-chinese-class.png
if cmp -s docs/screenshots/pool-remake-chinese-race.png docs/screenshots/pool-remake-chinese-class.png; then
  echo "the creation flow did not reach the class picker" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
for key in Return Return; do
  pulse "$key"
done
sleep 0.8
shot docs/screenshots/pool-remake-chinese-sheet.png
if cmp -s docs/screenshots/pool-remake-chinese-class.png docs/screenshots/pool-remake-chinese-sheet.png; then
  echo "the creation flow did not reach the character sheet" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
# 走完建角、加入隊伍、開始冒險，拍下羅夫導覽的第一頁——那是原版敘述文字
# 第一次以中文出現在畫面上的地方。
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
sleep 1.5
shot docs/screenshots/pool-remake-chinese-tour.png
if cmp -s docs/screenshots/pool-remake-chinese-sheet.png docs/screenshots/pool-remake-chinese-tour.png; then
  echo "B did not begin the adventure" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
# F5 開戰術盤面，確認那一頁在漢字字型下四行資訊與功能鍵列都不相疊。
pulse F5
sleep 0.8
shot docs/screenshots/pool-remake-chinese-tactical.png
if cmp -s docs/screenshots/pool-remake-chinese-tour.png docs/screenshots/pool-remake-chinese-tactical.png; then
  echo "F5 did not open the tactical board" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
# F5 關掉盤面，J 開探險者手冊，再打 46 + ENTER 跳到遊戲文字實際引用的那一條——
# 「抄進手冊，成為線索報導 46」在畫面上說得出口，這裡就要翻得到。
pulse F5
sleep 0.5
pulse j
sleep 0.8
shot docs/screenshots/pool-remake-chinese-journal.png
if cmp -s docs/screenshots/pool-remake-chinese-tour.png docs/screenshots/pool-remake-chinese-journal.png; then
  echo "J did not open the journal" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
pulse 4
sleep 0.4
pulse 6
sleep 0.4
pulse Return
sleep 0.8
shot docs/screenshots/pool-remake-chinese-journal-46.png
if cmp -s docs/screenshots/pool-remake-chinese-journal.png docs/screenshots/pool-remake-chinese-journal-46.png; then
  echo "typing 46 did not jump to clue 46" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
sha256sum docs/screenshots/pool-remake-chinese-*.png
'
