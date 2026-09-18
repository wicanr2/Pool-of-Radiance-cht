#!/usr/bin/env bash
# 拿發行包拍戰利品畫面（spec 034／040、#47）：真的 Ebitengine 視窗、逐鍵輸入、
# 畫面識別字等畫面（-screen-state），ffmpeg 抓視窗。
# 用法：tools/capture-treasure.sh <版本> [zh|en] [out 目錄]
#
# 怎麼走到戰利品：建一個人的隊伍、進貧民窟、開作弊（鎖 HP + 一擊斃命）之後往前走，
# 遇到隨機遭遇就整隊 Q）UICK 交給 AI 打。打贏就是戰利品畫面。
# 走幾步會遇到不是固定的，所以腳本用「等畫面識別字」推進，不用 sleep 猜。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"; LANG_MODE="${2:-zh}"
[[ -n "$VERSION" ]] || { echo "用法：tools/capture-treasure.sh <版本> [zh|en] [out]" >&2; exit 2; }
APPIMAGE="$ROOT/dist-all/$VERSION/patch/pool-of-radiance-remake-$VERSION-x86_64.AppImage"
FONT_DIR="${ETEN_FONT_DIR:-/home/anr2/cht/etan_font}"
OUT="${3:-$ROOT/workplace/capture-treasure-$LANG_MODE}"
test -f "$APPIMAGE" || { echo "找不到 $APPIMAGE" >&2; exit 2; }
test -f "$ROOT/Pool of Radiance (1988).zip"
test -d "$FONT_DIR"
mkdir -p "$OUT"
OUT="$(cd "$OUT" && pwd)"
docker run --rm --network none --memory 3g --cpus 2 --pids-limit 384 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" --tmpfs /tmp/.X11-unix:rw,mode=1777 \
  -e HOME=/tmp/home -e LANG_MODE="$LANG_MODE" \
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
die() { echo "$1；目前畫面：$(screen)" >&2; tail -30 /tmp/game.log >&2 || true; exit 1; }
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
# **六個人，不是一個。** 一人隊伍在戰術盤上一個 tick 內就被打光，而鎖 HP 的寫回是
# tick 結束才做（spec 141 的限制），於是畫面直接跳回標題，拍不到戰利品。
make_character() {
  step c creation-race
  for unused in 1 2 3 4 5; do pulse End; done
  step Return creation-class
  step Return creation-alignment
  step Return creation-roll
  step Return creation-name
  xdotool type --delay 120 "$1"
  step Return creation-portrait
  step k creation-icon-0
  step e creation-icon-confirm
  step y menu
}
for name in HEROA HEROB HEROC HEROD HEROE HEROF; do
  make_character "$name"
done
# `a` 把「還沒進隊伍的第一個角色」加進來，所以按六次就是六個人。
for unused in 1 2 3 4 5 6; do pulse a; sleep 0.2; done
step b adventure-intro
step Return adventure-move 500
# 作弊：鎖 HP + 一擊斃命。一個人的隊伍照原版規則打不贏隨機遭遇，
# 而打輸了拍不到戰利品。
step F6 cheat-menu
pulse l
pulse o
# 穿牆也開：只按 Up 會一直撞牆，走不動就不會有隨機遭遇（第一次跑就是這樣，
# 走了 400 步一場都沒遇到）。
pulse w
sleep 0.3
shot cheats-menu-on
step Escape adventure-move
sleep 0.4
shot cheats-on
# 走路 → 打 → 看有沒有戰利品，沒有就再走。一場遭遇不一定有戰利品（這一段的
# 隨機遭遇很多場打完就沒了），所以要打到出現為止，不是打一場就期待它出現。
battle=0
while test "$(screen)" != treasure; do
  battle=$((battle+1)); test "$battle" -lt 12 || die "打了 $((battle-1)) 場都沒有戰利品"
  # 往前走到遇到遭遇為止。事件框照按 Return，`combat-staged` 按 ENTER 進戰術盤。
  n=0
  walked=
  while test "$(screen)" != tactical && test "$(screen)" != treasure; do
    n=$((n+1)); test "$n" -lt 400 || die "走了 400 步沒有遇到隨機遭遇"
    now="$(screen)"; if test "$now" != "$walked"; then echo "第 $battle 場・走路第 $n 步 → $now"; walked="$now"; fi
    case "$now" in
      # `adventure-cell-done` 也是可以走的（格子事件跑完、還停在同一格），
      # 只認 `adventure-move` 會一直按 Return 而一步都不走。
      adventure-move|adventure-cell-done) pulse Up ;;
      treasure*) break ;;
      # 遭遇排好了還沒進戰術盤：按 ENTER 進去。
      combat-staged) pulse Return ;;
      *) pulse Return ;;
    esac
    sleep 0.05
  done
  # 整隊交給 AI 打（Q）UICK）。`n` 回答「Y 繼續戰鬥　N 結束戰鬥」那個提示——
  # 敵方清光時原版會多問一次（spec 062），答 `y` 是再開一個空回合，於是提示再
  # 出現一次，整場永遠不會結束。
  n=0
  last=
  while test "$(screen)" = tactical; do
    n=$((n+1)); test "$n" -lt 120 || { shot tactical-stuck; die "戰鬥沒有結束"; }
    pulse q; pulse Return; pulse n
    now="$(screen)"
    if test "$now" != "$last"; then echo "第 $battle 場・戰鬥第 $n 圈 → $now"; last="$now"; fi
  done
  echo "第 $battle 場結束，畫面是 $(screen)"
  # 全滅就沒戲了（鎖 HP 開著，理論上不會發生）。
  test "$(screen)" != title || die "全滅回到標題"
done
await treasure 600
shot treasure-top
# TAKE：幣種清單。選單是方向鍵移動、ENTER 選。
pulse Right
step Return treasure-take
shot treasure-take
pulse Return
sleep 0.4
shot treasure-take-money
step Escape treasure-take 20
step Escape treasure 20
# VIEW：人物頁。
step Return view-sheet 20
shot treasure-view
step Escape treasure 20
# SHARE：分錢之後選項會變。
for unused in 1 2 3; do pulse Right; done
pulse Return
sleep 0.6
shot treasure-after-share
echo ok
'
echo "截圖 → $OUT"
