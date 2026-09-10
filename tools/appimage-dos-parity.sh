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
# 尺度：remake 的邏輯畫布是 640x400、視窗 1280x800（原版 320x200 的 4 倍）。
# 截圖用**最近鄰**降回 320x200 再比；平滑縮放會把差異抹掉。
#
# 視窗一定要是邏輯畫布的**整數倍**：原版的字是 8x15／16x15 的點陣，1.5 倍
# 放大會把某些像素行複製、某些丟掉，筆畫密的漢字看起來就像糊在一起，而降
# 取樣取到的也不是同一組邏輯像素。
#
# 逐格相同是**只對兩張**宣稱的：`title`（同一份 TITLE.DAX，位置與縮放都一樣）
# 與 `first-person`（spec 047／126，玩家整趟冒險看最久的一塊）。
#
# 建隊到進城那一段的每一張也拍、也比，但那幾張的版面是 remake 自己的
#（原版是 320x200 的文字排版，remake 是 640x400 加漢字），所以比出來的數字
# 是**版面差異的量度**，不是缺陷計數。看的是「框線對不對得上」與
#「哪一塊排在哪裡」，逐項判讀在 docs/audit/dos-parity-sample.md。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
FLAVOUR="${2:-full-local}"
LANG_MODE="${3:-zh}"
FONT_DIR="${ETEN_FONT_DIR:-/home/anr2/cht/etan_font}"
APPIMAGE="$ROOT/dist-all/$VERSION/$FLAVOUR/pool-of-radiance-remake-$VERSION-x86_64.AppImage"
ENGINE_DIR="${GOLDEN_BOX_REMAKE_ENGINE_DIR:-$ROOT/../golden-box-remake-engine}"
REF="$ROOT/workplace/dosgolem-ref"
# 市政廳那一格的基準是另一條鍵序產的（主基準走的是「導覽完往西撞遭遇」，
# 不經過市政廳）。缺了就少那兩項，不擋整份對拍——重生的方式：
#   POOL_DOSGOLEM_OUT=workplace/dosgolem-ref-cityhall \
#   POOL_DOSGOLEM_KEYS=<主鍵序把結尾換成 Left,Left,Up,Up,Up,Return,Up> \
#   tools/dosgolem-reference.sh
REF_CITYHALL="$ROOT/workplace/dosgolem-ref-cityhall"
test -d "$REF_CITYHALL" || REF_CITYHALL="$REF"
OUT="$ROOT/workplace/dos-parity-$LANG_MODE"

[[ -n "$VERSION" ]] || { echo "用法：tools/appimage-dos-parity.sh <版本> [patch|full-local] [zh|en]" >&2; exit 2; }
test -f "$APPIMAGE"
# **發行包要是當前原始碼建的。** 這一支比的是打包好的 AppImage，而它是現成的
# 檔案——改完程式碼直接跑對拍，量到的是上一版，數字看起來正常，結論整份是空的。
# 基準那一側早就有產地證明閘門（下面那段），這一側先前沒有：2026-09-10 因此
# 拿 9/9 建的包量了一整輪，還把別的 commit 的升幅記到這一輪頭上。
STALE_SOURCE="$(find "$ROOT/cmd" "$ROOT/internal" "$ENGINE_DIR" \
  -name '*.go' -newer "$APPIMAGE" -print -quit 2>/dev/null || true)"
[[ -z "$STALE_SOURCE" ]] || {
  echo "發行包比原始碼舊（$STALE_SOURCE 改過之後沒有重新打包）。" >&2
  echo "先跑 tools/package-release.sh $VERSION 再對拍。" >&2; exit 2; }
test -f "$ROOT/Pool of Radiance (1988).zip"
test -f "$REF/shots.json" || {
  echo "沒有基準畫面：先跑 tools/dosgolem-reference.sh" >&2; exit 2; }
# **基準一律由 dosgolem 產**（AGENTS.md §7）。這裡不是提醒是閘門：
# 一份放了幾天的 `workplace/` 目錄從外表看不出它是誰產的，而基準來自哪裡
# 是整份對拍結論的前提。缺產地證明就重跑 `tools/dosgolem-reference.sh`。
python3 - "$REF" <<'GATE'
import json, os, sys

path = os.path.join(sys.argv[1], "provenance.json")
if not os.path.exists(path):
    sys.exit(f"{path} 不在：基準不知道是誰產的。重跑 tools/dosgolem-reference.sh")
record = json.load(open(path, encoding="utf-8"))
if record.get("generator") != "dosgolem":
    sys.exit(f"基準的 generator 是 {record.get('generator')!r}，對拍只認 dosgolem")
print(f"基準：dosgolem {record.get('generator_revision', '')[:12]} "
      f"{record.get('frames', 0)} 幀　原版 start.exe "
      f"{record.get('original_exe_sha256', '')[:12]}")
GATE
[[ "$LANG_MODE" != zh ]] || test -f "$FONT_DIR/stdfont.15"

# 容器裡那段鍵序走到市政廳外那一格（spec 082）：導覽結束在 (0,4) 朝西，
# 左轉兩次朝東再走三步就到 (3,4)。轉向與移動不會改變畫面識別字，所以那一段用
# pulse 直接按，不能用 step——step 會因為識別字早就符合而一次都不按。
#
# 路上的格子會印字，而事件 pending 的時候方向鍵按不動，所以每走一步要把純文字
# 的那種按掉（east_step）。它只吃 adventure-cell-text：市政廳第一段是腳本自己
# 放的單選項選單（adventure-cell-menu），那一張正是要拍的，不能一起清掉。
#
# **容器腳本裡不要放多行中文註解**：那一整段是 `bash -c '…'` 的單引號字串，
# 中文註解在裡面曾經讓主機這一側報 "指令找不到"（對拍照樣跑完、exit 0，
# 只是尾巴多一行雜訊，看起來像對拍壞了）。註解留在主機端這裡。
rm -rf "$OUT"; mkdir -p "$OUT"
docker run --rm --network none --memory 3g --cpus "${PARITY_CPUS:-2}" --pids-limit 384 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" --tmpfs /tmp/.X11-unix:rw,mode=1777 \
  -e HOME=/tmp/home -e LANG_MODE="$LANG_MODE" \
  -v "$APPIMAGE:/game.AppImage:ro" -v "$FONT_DIR:/fonts:ro" \
  -v "$ROOT/Pool of Radiance (1988).zip:/zip/pool.zip:ro" \
  -v "$REF:/ref:ro" -v "$REF_CITYHALL:/ref-cityhall:ro" \
  -v "$OUT:/out" -v "$ROOT/tools:/tools:ro" \
  -w /tmp \
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
# 滑鼠移到螢幕右下角，**要在遊戲視窗外面**：視窗現在是 1280x800，
# 舊的 (1190,790) 會落在視窗裡。
xdotool mousemove 1390 890

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

# 建隊到進城，每一站拍一張。原版同一段由 tools/dosgolem-reference.sh 拍，
# 兩邊逐張對得起來，版面差在哪就看得出來。
step Return menu
sleep 0.4
shot remake-menu-empty
step c creation-race
sleep 0.4
shot remake-race
step Return creation-gender
sleep 0.4
shot remake-gender
step Return creation-class
sleep 0.4
shot remake-class
step Return creation-alignment
sleep 0.4
shot remake-alignment
step Return creation-roll
sleep 0.5
shot remake-sheet
step Return creation-name
sleep 0.3
shot remake-name
xdotool type --delay 120 HERO
sleep 0.4
step Return creation-portrait
sleep 0.4
shot remake-portrait
step k creation-icon-0
sleep 0.4
shot remake-icon
step e creation-icon-confirm
sleep 0.4
shot remake-icon-confirm
step y menu
sleep 0.4
pulse a
sleep 0.6
shot remake-menu-party
step b adventure-intro
sleep 0.6
shot remake-intro
step Return adventure-move 60
sleep 0.6
shot remake-first-person
# 平面圖：`A` 把第一人稱視野換成俯視圖（原版指令列的 `AREA`）。基準那一側
# 也是在導覽結束的同一格按 `a` 拍的，所以兩邊站的位置一樣。
step a adventure-map
sleep 0.6
shot remake-map
step a adventure-move
sleep 0.4
# 戰鬥畫面（spec 129）。原版那一側是走到第一場遭遇拍的；remake 這一側用 F5
# 叫出同一支繪製——盤面內容本來就不同，這一項看的是版面。
# 市政廳外那一格：見主機端那一段註解。
east_step() {
  pulse Up
  sleep 0.3
  n=0
  while test "$(screen)" = "adventure-cell-text"; do
    pulse Return
    sleep 0.2
    n=$((n+1))
    test "$n" -lt 20 || die "走到市政廳的路上清不掉格子事件"
  done
}
pulse Left
pulse Left
east_step
east_step
east_step
await adventure-cell-menu
sleep 0.6
shot remake-city-hall-first
pulse Return
await adventure-cell-done
sleep 0.6
shot remake-city-hall-second

step F5 tactical
sleep 0.6
shot remake-tactical

python3 /tools/dos-parity-compare.py /ref /out /ref-cityhall
'
echo "報告 → $OUT"
