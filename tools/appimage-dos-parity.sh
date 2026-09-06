#!/usr/bin/env bash
# 拿**打包好的 AppImage**（不是原始碼建置）對原版 DOS 畫面抽樣對拍。
# 用法：tools/appimage-dos-parity.sh <版本> [patch|full-local] [zh|en]
#
# 為什麼要對「發行包」而不是對原始碼建置：spec 126 那組 100% 是單元測試在比
# remake 內部合成出來的那張圖。**「測試綠」與「我們寄出去的那個檔案畫得對」
# 是兩件事**——中間隔著打包、資源內嵌、視窗縮放與實際的繪圖路徑。
#
# 基準畫面由 **dosgolem** 產（`tools/dosgolem-reference.sh`），不是 DOSBox。
# dosgolem 直接吐 320x200 的色號陣列，走位靠程式自己的訊號；DOSBox 只能從
# 外面看畫面，送鍵靠 xdotool、等畫面靠 sleep，而猜對與猜錯在截圖上長得一樣。
#
# 尺度：remake 的邏輯畫布是 640x400、視窗 960x600（3 倍）。截圖用**最近鄰**
# 降回 320x200 再比；平滑縮放會把差異抹掉。
#
# 抽樣，不是全程（使用者 2026-09-07 指定）。取的是兩張最說明問題的：
#   * `title`：整張畫面。原版與 remake 畫的是同一份 TITLE.DAX，
#     位置與縮放也一樣，所以這一張**應該逐格相同**（扣掉 remake 自己加的
#     按鍵提示那一行）。
#   * `first-person`：第一人稱那一框的 88x88 內容。**那是唯一宣稱過逐格
#     相同的東西**（spec 047／126），也是玩家整趟冒險看最久的一塊。
#
# 其餘畫面（人物管理選擇項、種族選單、人物資料頁）的版面是 remake 自己的，
# 不是對拍目標；它們的中文化由 `tools/capture-chinese-menu.sh` 顧。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
FLAVOUR="${2:-full-local}"
LANG_MODE="${3:-zh}"
FONT_DIR="${ETEN_FONT_DIR:-/home/anr2/cht/etan_font}"
APPIMAGE="$ROOT/dist-all/$VERSION/$FLAVOUR/pool-of-radiance-remake-$VERSION-x86_64.AppImage"
REF="$ROOT/workplace/dosgolem-ref"
OUT="$ROOT/workplace/dos-parity-$LANG_MODE"

[[ -n "$VERSION" ]] || { echo "用法：tools/appimage-dos-parity.sh <版本> [patch|full-local] [zh|en]" >&2; exit 2; }
test -f "$APPIMAGE"
test -f "$ROOT/Pool of Radiance (1988).zip"
test -f "$REF/shots.json" || {
  echo "沒有基準畫面：先跑 tools/dosgolem-reference.sh" >&2; exit 2; }
[[ "$LANG_MODE" != zh ]] || test -f "$FONT_DIR/stdfont.15"

rm -rf "$OUT"; mkdir -p "$OUT"
docker run --rm --network none --memory 3g --cpus "${PARITY_CPUS:-2}" --pids-limit 384 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" --tmpfs /tmp/.X11-unix:rw,mode=1777 \
  -e HOME=/tmp/home -e LANG_MODE="$LANG_MODE" \
  -v "$APPIMAGE:/game.AppImage:ro" -v "$FONT_DIR:/fonts:ro" \
  -v "$ROOT/Pool of Radiance (1988).zip:/zip/pool.zip:ro" \
  -v "$REF:/ref:ro" -v "$OUT:/out" -v "$ROOT/tools:/tools:ro" -w /tmp \
  wasteland-go:1.24-x11-record-r1 bash -c '
set -eu
mkdir -p "$HOME" /tmp/run
cd /tmp/run
cp /game.AppImage ./game.AppImage
./game.AppImage --appimage-extract >/tmp/extract.log 2>&1
Xvfb :99 -screen 0 1200x800x24 >/tmp/xvfb.log 2>&1 &
xvfb=$!
game=
finish() { test -z "$game" || kill "$game" 2>/dev/null || true; kill "$xvfb" 2>/dev/null || true; }
trap finish EXIT
export DISPLAY=:99
until test -S /tmp/.X11-unix/X99; do sleep 0.1; done

STATE=/tmp/pool-screen
rm -f "$STATE"
if test "$LANG_MODE" = zh; then
  set -- -lang zh -eten-font /fonts/stdfont.15
else
  set -- -lang en
fi
(exec ./squashfs-root/AppRun -zip /zip/pool.zip -screen-state "$STATE" "$@") >/tmp/game.log 2>&1 &
game=$!
n=0
window=
until test -n "$window"; do
  window=$(xdotool search --name "Pool of Radiance Remake" 2>/dev/null | head -1 || true)
  sleep 0.2; n=$((n+1))
  test "$n" -lt 150 || { echo "視窗沒有出現" >&2; tail -20 /tmp/game.log >&2; exit 1; }
done
xdotool windowfocus "$window"
eval "$(xdotool getwindowgeometry --shell "$window")"
xdotool mousemove 1190 790

screen() { cat "$STATE" 2>/dev/null | tr -d "\n"; }
die() { echo "$1" >&2; echo "目前畫面：$(screen)" >&2; tail -20 /tmp/game.log >&2 || true; exit 1; }
await() {
  want=$1; rounds=${2:-150}; n=0
  while test "$(screen)" != "$want"; do
    sleep 0.1; n=$((n+1))
    test "$n" -lt "$rounds" || die "等不到畫面 $want"
  done
}
pulse() { xdotool keydown "$1"; sleep 0.12; xdotool keyup "$1"; sleep 0.12; }
step() {
  key=$1; want=$2; attempt=0
  while test "$(screen)" != "$want"; do
    attempt=$((attempt+1))
    test "$attempt" -le 40 || die "按 $key 走不到 $want"
    pulse "$key"
    waited=0
    while test "$(screen)" != "$want" && test "$waited" -lt 15; do
      sleep 0.1; waited=$((waited+1))
    done
  done
}
shot() {
  ffmpeg -y -hide_banner -loglevel error -f x11grab -video_size "${WIDTH}x${HEIGHT}" \
    -i ":99+${X},${Y}" -frames:v 1 "/out/$1.png"
}

# 標題：不用按任何鍵，那一張本來就是開場。
await title
sleep 0.6
shot remake-title

# 走完建角與導覽到自由移動，那時第一人稱框裡是乾淨的視野。
step Return menu
step c creation-race
step Return creation-class
step Return creation-alignment
step Return creation-roll
step Return creation-name
xdotool type --delay 120 HERO
sleep 0.3
step Return creation-portrait
step k creation-icon-0
step e creation-icon-confirm
step y menu
sleep 0.4
pulse a
sleep 0.6
step b adventure-intro
step Return adventure-move 60
sleep 0.6
shot remake-first-person

python3 /tools/dos-parity-compare.py /ref /out
'
echo "報告 → $OUT"
