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
# `tools/appimage-dos-parity-repeat.sh` 會用固定骰子／ECL seed 與營火圖格連跑三次，
# 並逐位元組核對全部 remake 截圖；單跑仍由本腳本產生報表。
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
REF_CAMPQUIT="$ROOT/workplace/dosgolem-ref-campquit"
REF_TEMPLE="$ROOT/workplace/dosgolem-ref-temple"
REF_SHOP="$ROOT/workplace/dosgolem-ref-shop"
REF_SPELLS="$ROOT/workplace/dosgolem-ref-spells"
OUT="${POOL_PARITY_OUT:-$ROOT/workplace/dos-parity-$LANG_MODE}"

[[ -n "$VERSION" ]] || { echo "用法：tools/appimage-dos-parity.sh <版本> [patch|full-local] [zh|en]" >&2; exit 2; }
case "$OUT" in
  "$ROOT"/workplace/dos-parity-*) ;;
  *) echo "POOL_PARITY_OUT 必須位於 $ROOT/workplace/dos-parity-*" >&2; exit 2 ;;
esac
test -f "$APPIMAGE"
# **發行包要是當前原始碼建的。** 這一支比的是打包好的 AppImage，而它是現成的
# 檔案——改完程式碼直接跑對拍，量到的是上一版，數字看起來正常，結論整份是空的。
# 基準那一側早就有產地證明閘門（下面那段），這一側先前沒有：2026-09-10 因此
# 拿 9/9 建的包量了一整輪，還把別的 commit 的升幅記到這一輪頭上。
# 二分舊版時要**故意**量舊包（#51），所以留一個明講的出口：`POOL_PARITY_ALLOW_STALE=1`。
# 預設仍然拒跑——這道閘門擋的是「改完程式碼忘了重新打包」，不是「我知道自己在量哪一版」。
if [[ "${POOL_PARITY_ALLOW_STALE:-0}" == 1 ]]; then
  echo "POOL_PARITY_ALLOW_STALE=1：略過「發行包比原始碼舊」的檢查，量的是 $VERSION 這一包。" >&2
  STALE_SOURCE=""
else
STALE_SOURCE="$(docker run --rm --network none --memory 128m --cpus 1 --pids-limit 32 \
  -v "$ROOT:/src:ro" -v "$ENGINE_DIR:/engine:ro" -v "$APPIMAGE:/game.AppImage:ro" \
  debian:bookworm-slim sh -c \
  "find /src/cmd /src/internal /engine -name '*.go' -newer /game.AppImage -print -quit 2>/dev/null" || true)"
fi
[[ -z "$STALE_SOURCE" ]] || {
  echo "發行包比原始碼舊（$STALE_SOURCE 改過之後沒有重新打包）。" >&2
  echo "先跑 tools/package-release.sh $VERSION 再對拍。" >&2; exit 2; }
test -f "$ROOT/Pool of Radiance (1988).zip"
for ref_dir in "$REF" "$REF_CITYHALL" "$REF_CAMPQUIT" "$REF_TEMPLE" "$REF_SHOP" "$REF_SPELLS"; do
  test -f "$ref_dir/shots.json" || {
    echo "沒有基準畫面：$ref_dir/shots.json" >&2; exit 2; }
  test -f "$ref_dir/provenance.json" || {
    echo "沒有產地證明：$ref_dir/provenance.json" >&2; exit 2; }
done
# **基準一律由 dosgolem 產**（AGENTS.md §7）。這裡不是提醒是閘門：
# 一份放了幾天的 `workplace/` 目錄從外表看不出它是誰產的，而基準來自哪裡
# 是整份對拍結論的前提。缺產地證明就重跑 `tools/dosgolem-reference.sh`。
docker run --rm -i --network none --memory 128m --cpus 1 --pids-limit 32 \
  -v "$REF:/ref:ro" -v "$REF_CITYHALL:/ref-cityhall:ro" \
  -v "$REF_CAMPQUIT:/ref-campquit:ro" -v "$REF_TEMPLE:/ref-temple:ro" \
  -v "$REF_SHOP:/ref-shop:ro" -v "$REF_SPELLS:/ref-spells:ro" \
  python:3.12-slim python - /ref /ref-cityhall /ref-campquit /ref-temple /ref-shop /ref-spells <<'GATE'
import json, os, sys

original = None
for directory in sys.argv[1:]:
    path = os.path.join(directory, "provenance.json")
    record = json.load(open(path, encoding="utf-8"))
    if record.get("generator") != "dosgolem":
        sys.exit(f"{directory} 的 generator 是 {record.get('generator')!r}，對拍只認 dosgolem")
    digest = record.get("original_exe_sha256", "")
    if original is None:
        original = digest
    elif digest != original:
        sys.exit(f"{directory} 的原版 start.exe 與主基準不同")
    print(f"基準 {os.path.basename(directory)}：dosgolem "
          f"{record.get('generator_revision', '')[:12]} {record.get('frames', 0)} 幀　"
          f"原版 start.exe {digest[:12]}")
GATE
[[ "$LANG_MODE" != zh ]] || test -f "$FONT_DIR/stdfont.15"

# 容器裡那段鍵序走到市政廳外那一格（spec 082）：導覽結束在 (0,4) 朝西，
# 左轉兩次朝東再走三步就到 (3,4)。轉向與移動不會改變畫面識別字，所以那一段用
# turn／east_step（看 `.sync` 的朝向與座標），不能用 step——step 會因為識別字
# 早就符合而一次都不按。
#
# 路上的格子會印字，而事件 pending 的時候方向鍵按不動，所以每走一步要把純文字
# 的那種按掉（east_step）。它只吃 adventure-cell-text：市政廳第一段是腳本自己
# 放的單選項選單（adventure-cell-menu），那一張正是要拍的，不能一起清掉。
#
# **容器腳本裡不要放多行中文註解，也不要在註解裡放反引號**：那一整段是
# `bash -c '…'` 的單引號字串，
# 中文註解在裡面曾經讓主機這一側報 "指令找不到"（對拍照樣跑完、exit 0，
# 只是尾巴多一行雜訊，看起來像對拍壞了）。註解留在主機端這裡。
#
# **送鍵與擷圖不靠 sleep（#92）。** 遊戲在 `-screen-state` 旁邊寫一份
# `<檔名>.sync`：Update 次數、讀到的按下數與放開數、畫面識別字、地圖、座標、
# 朝向（cmd/pool-game/screen_state.go 的 publishAutomationSync）。
#   * pulse：按住直到按下數變了才放開，放開後等放開數變了才回來。Ebiten 每幀
#     才輪詢一次鍵盤，舊版「按 0.12 秒就放」在一幀拖長時整下不見；回來時那一格
#     的處理結果也已經寫出來了，不必再睡。
#   * turn／east_step：按完要看到朝向或座標真的變了，否則當場 die；走完以
#     Update 次數等畫面與位置連續 90 拍（比速度 4 的一拍 54 格長）不動，再清
#     格子文字。舊版睡 0.3 秒就讀畫面，格子字還沒出來就被當成沒有，下一步
#     被擋、少走一格，停在 adventure-move 等不到 adventure-cell-menu。
#   * shot：先等 30 拍不動，再連拍到兩張逐位元組相同才收（§7 同一條做法），
#     拍完確認畫面識別字沒換；次數記在 shot-attempts.txt，二十張都不一致的
#     記在 unstable.txt。
# 法術兩張的數字會跳（spells 52055／52065）也是掉鍵：牧師流程在種族頁按五下
# End（游標往下、會繞回），掉一下就換了種族，擲出來的人物不同，法術頁那一行
# 「已記 0/1,還能記 1」變成「0/0,還能記 0」。不是游標閃爍，畫面本身沒有閃的東西。
rm -rf "$OUT"; mkdir -p "$OUT"
# POOL_PARITY_CONTAINER 可替這個容器命名，並行跑好幾份時分得出是誰的。
docker run --rm ${POOL_PARITY_CONTAINER:+--name "$POOL_PARITY_CONTAINER"} \
  --network none --memory 3g --cpus "${PARITY_CPUS:-2}" --pids-limit 384 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" --tmpfs /tmp/.X11-unix:rw,mode=1777 \
  -e HOME=/tmp/home -e LANG_MODE="$LANG_MODE" \
  -v "$APPIMAGE:/game.AppImage:ro" -v "$FONT_DIR:/fonts:ro" \
  -v "$ROOT/Pool of Radiance (1988).zip:/zip/pool.zip:ro" \
  -v "$REF:/ref:ro" -v "$REF_CITYHALL:/ref-cityhall:ro" \
  -v "$REF_CAMPQUIT:/ref-campquit:ro" -v "$REF_TEMPLE:/ref-temple:ro" \
  -v "$REF_SHOP:/ref-shop:ro" -v "$REF_SPELLS:/ref-spells:ro" \
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
rm -f "$STATE" "$STATE.sync"
if test "$LANG_MODE" = zh; then
  set -- -lang zh -eten-font /fonts/stdfont.15
else
  set -- -lang en
fi
(exec ./squashfs-root/AppRun -zip /zip/pool.zip -screen-state "$STATE" \
  -dice-seed 136 -capture-camp-fire-frame 0 "$@") >/tmp/game.log 2>&1 &
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

sync_read() {
  s_tick=0; s_down=0; s_up=0; s_screen=; s_map=; s_x=; s_y=; s_face=
  if test -f "$STATE.sync"; then read -r s_tick s_down s_up s_screen s_map s_x s_y s_face < "$STATE.sync" || true; fi
}
screen() { sync_read; printf "%s" "$s_screen"; }
where() { sync_read; printf "%s %s %s %s" "$s_map" "$s_x" "$s_y" "$s_face"; }
die() {
  echo "$1" >&2; sync_read
  echo "目前畫面：$s_screen 位置：$s_map $s_x $s_y $s_face 第 $s_tick 拍" >&2
  tail -20 /tmp/game.log >&2 || true; exit 1
}
wait_ticks() {
  local t0 w=0; sync_read; t0=$s_tick
  while sync_read; test $((s_tick - t0)) -lt "$1"; do
    sleep 0.02; w=$((w+1)); test "$w" -lt 3000 || die "遊戲停住了（等 $1 拍）"
  done
}
settle() {
  local need last since now n
  need=${1:-90}; sync_read; last="$s_screen $s_map $s_x $s_y $s_face"; since=$s_tick; n=0
  while :; do
    sleep 0.03; sync_read; now="$s_screen $s_map $s_x $s_y $s_face"
    if test "$now" != "$last"; then last=$now; since=$s_tick; fi
    test $((s_tick - since)) -lt "$need" || return 0
    n=$((n+1)); test "$n" -lt 4000 || die "畫面一直在變"
  done
}
await() {
  want=$1; rounds=${2:-150}; n=0
  while test "$(screen)" != "$want"; do
    sleep 0.1; n=$((n+1))
    test "$n" -lt "$rounds" || die "等不到畫面 $want"
  done
}
await_adventure() {
  rounds=${1:-150}; n=0
  while test "$(screen)" != adventure-move && test "$(screen)" != adventure-cell-done; do
    sleep 0.1; n=$((n+1))
    test "$n" -lt "$rounds" || die "等不到自由移動畫面"
  done
}
pulse() {
  local d0 u0 n
  sync_read; d0=$s_down; u0=$s_up; n=0
  xdotool keydown "$1"
  while sync_read; test "$s_down" -le "$d0"; do
    sleep 0.02; n=$((n+1))
    test "$n" -lt 1500 || { xdotool keyup "$1"; die "遊戲讀不到按鍵 $1"; }
  done
  xdotool keyup "$1"
  n=0
  while sync_read; test "$s_up" -le "$u0"; do
    sleep 0.02; n=$((n+1))
    test "$n" -lt 1500 || die "遊戲讀不到放開 $1"
  done
}
step() {
  key=$1; want=$2; limit=${3:-40}; attempt=0
  while test "$(screen)" != "$want"; do
    attempt=$((attempt+1))
    test "$attempt" -le "$limit" || die "按 $key 走不到 $want（試了 $limit 次）"
    pulse "$key"
    sync_read; t0=$s_tick; n=0
    while test "$s_screen" != "$want" && test $((s_tick - t0)) -lt 90; do
      sleep 0.03; sync_read; n=$((n+1)); test "$n" -lt 3000 || die "遊戲停住了"
    done
  done
}
wait_moved() {
  local before t0
  before=$1; sync_read; t0=$s_tick
  while test "$s_map $s_x $s_y $s_face" = "$before"; do
    test $((s_tick - t0)) -lt 60 || return 1
    sleep 0.02; sync_read
  done
}
turn() {
  local before
  before=$(where); pulse "$1"
  wait_moved "$before" || die "按 $1 沒有轉向（停在 $before）"
}
east_step() {
  local before n
  before=$(where); pulse Up
  wait_moved "$before" || die "往前走不動（停在 $before）"
  settle 90
  n=0
  while test "$(screen)" = "adventure-cell-text"; do
    pulse Return
    settle 90
    n=$((n+1))
    test "$n" -lt 20 || die "清不掉格子事件"
  done
}
shot() {
  local first prev sum n
  settle 30
  first=$(screen); prev=; n=0
  while :; do
    ffmpeg -y -hide_banner -loglevel error -f x11grab -video_size "${WIDTH}x${HEIGHT}" \
      -i ":99+${X},${Y}" -frames:v 1 /tmp/shot.png
    sum=$(md5sum < /tmp/shot.png)
    if test "$sum" = "$prev"; then break; fi
    prev=$sum; n=$((n+1))
    if test "$n" -ge 20; then echo "$1 連拍 20 張都不一致，收最後一張" >&2; echo "$1" >> /out/unstable.txt; break; fi
    wait_ticks 6
  done
  test "$(screen)" = "$first" || die "拍 $1 的時候畫面從 $first 換掉了"
  mv /tmp/shot.png "/out/$1.png"
  echo "$1 $n" >> /out/shot-attempts.txt
}

# 標題：不用按任何鍵，那一張本來就是開場。
await title
shot remake-title

# 建隊到進城，每一站拍一張。原版同一段由 tools/dosgolem-reference.sh 拍，
# 兩邊逐張對得起來，版面差在哪就看得出來。
step Return menu
shot remake-menu-empty
step c creation-race
shot remake-race
step Return creation-gender
shot remake-gender
step Return creation-class
shot remake-class
step Return creation-alignment
shot remake-alignment
step Return creation-roll
shot remake-sheet
step Return creation-name
shot remake-name
xdotool type --delay 120 HERO
wait_ticks 10
step Return creation-portrait
shot remake-portrait
step k creation-icon-0
shot remake-icon
step e creation-icon-confirm
shot remake-icon-confirm
step y menu
pulse a
shot remake-menu-party
step b adventure-intro
shot remake-intro
# 導覽是自己會跑完的：tourActive 底下走的是 ECL，每一步靠 tourDelay 逐幀
# 遞減推進（Game Speed 是「等一拍」的統一單位）。按 Return 只是推過中間需要
# 按鍵的地方，所以這個上限量的是「等多久」不是「按幾次」——機器忙的時候
# Ebiten 的幀率掉下來，同樣的步數要等更久。60 次（約 90 秒）在負載高時不夠，
# 症狀是「按 Return 走不到 adventure-move、目前畫面 adventure-tour」。
step Return adventure-move 200
shot remake-first-person
# 平面圖：A 把第一人稱視野換成俯視圖（原版指令列的 AREA）。基準那一側
# 也是在導覽結束的同一格按 a 拍的，所以兩邊站的位置一樣。
step a adventure-map
shot remake-map
step a adventure-move
# 檢視人物：原版指令列的 VIEW。**兩邊多了一層**——原版有「選定角色」那個
# 全域（ds:5CF0h），按 v 直接開資料頁；remake 沒有那個全域，所以先挑人
# （view-pick）再按 Return 才進資料頁。退出鍵也不同：原版那一頁底下是
# VIEW: TRADE DROP EXIT（按 e），remake 是 ESC 返回，而且要按兩次
# （先從資料頁回挑人那一層，再退出去）。
step v view-pick
step Return view-sheet
shot remake-view-sheet
pulse Escape
step Escape adventure-move
# 戰鬥畫面（spec 129）。原版那一側是走到第一場遭遇拍的；remake 這一側用 F5
# 叫出同一支繪製——盤面內容本來就不同，這一項看的是版面。
# 市政廳外那一格：見主機端那一段註解。
turn Left
turn Left
east_step
east_step
east_step
await adventure-cell-menu
shot remake-city-hall-first
pulse Return
await adventure-cell-done
shot remake-city-hall-second

# 紮營（spec 135）：原版指令列的 ENCAMP，兩邊的指令列一樣
# SAVE VIEW MAGIC REST ALTER EXIT，都按 e 退出。
# **放在市政廳之後**：紮營會推進遊戲時間，擺在前面會讓市政廳那幾格的事件
# 狀態變掉，症狀是「等不到畫面 adventure-cell-menu」。兩邊只要各自走到同一
# 個畫面就好，先後順序不必跟基準那一側一致。
step e camp
shot remake-camp
# REST 是紮營底下最直接的一層；兩側都按 R 進去、E 回到最外層。
step r camp-rest
shot remake-camp-rest
step e camp
# ALTER 底下七個狀態都用原版的字母／數字鍵走，不用 direct-entry。
step a camp-alter
shot remake-camp-alter
step s camp-speed
shot remake-camp-speed
step e camp-alter
step p camp-pics
shot remake-camp-pics
step e camp-alter
step i camp-icon
shot remake-camp-icon
step e camp-alter
step o camp-order-select
shot remake-camp-order-select
step 1 camp-order-place
shot remake-camp-order-place
step 1 camp-order-select
step e camp-alter
step d camp-drop
shot remake-camp-drop
step n camp-alter
step e camp
step s camp-quit
shot remake-camp-quit
step n camp
step e adventure-cell-done

turn Right
turn Right
east_step
east_step
turn Right
east_step
await adventure-cell-menu
pulse Return
await temple
shot remake-temple
pulse Left
pulse Return
await_adventure
turn Right
turn Right
for unused in 1 2 3 4 5 6 7 8; do east_step; done
turn Left
for unused in 1 2 3 4 5 6 7; do east_step; done
await adventure-cell-menu
pulse Return
await shop
shot remake-shop
pulse Escape
await_adventure

step F5 tactical
shot remake-tactical

kill "$game" 2>/dev/null || true
wait "$game" 2>/dev/null || true
game=
STATE=/tmp/pool-screen-caster
rm -f "$STATE" "$STATE.sync"
(exec ./squashfs-root/AppRun -zip /zip/pool.zip -screen-state "$STATE" \
  -dice-seed 136 -capture-camp-fire-frame 0 "$@") >/tmp/game-caster.log 2>&1 &
game=$!
window=
n=0
until test -n "$window"; do
  window=$(xdotool search --name "Pool of Radiance Remake" 2>/dev/null | head -1 || true)
  sleep 0.2
  n=$((n+1))
  test "$n" -lt 150 || die "牧師流程的視窗沒有出現"
done
xdotool windowfocus "$window"
eval "$(xdotool getwindowgeometry --shell "$window")"
xdotool mousemove 1390 890
await title
step Return menu
step c creation-race
for unused in 1 2 3 4 5; do pulse End; done
step Return creation-class
step Return creation-alignment
step Return creation-roll
step Return creation-name
xdotool type --delay 120 HERO
wait_ticks 10
step Return creation-portrait
step k creation-icon-0
step e creation-icon-confirm
step y menu
pulse a
step b adventure-intro
step Return adventure-move 500
step i equipment
shot remake-equipment
step i adventure-move
step k spells-cleric-1
shot remake-spells
pulse m
step k adventure-move
step e camp
step r camp-rest
pulse h
pulse i
pulse i
pulse r
# **城區街上休息會被城衛隊攔下來**（原版就是這樣，收據
# `docs/audit/dos-tour-cell-rest.json`：排兩小時，00:05 被打斷，選項 GO／STAY）。
# 攔下來時畫面是 `adventure-cell-menu`；答 `GO`（走人，原版 STAY 會開打）。
# **選單是方向鍵移游標、ENTER 選**，不是按首字母——按 `g` 什麼都不會發生。
# `GO` 是第一個選項，所以游標不用動。
n=0
while test "$(screen)" = adventure-cell-menu; do
  n=$((n+1)); test "$n" -lt 10 || die "城衛隊那一問答不掉"
  pulse Return; sleep 0.3
done
await_adventure 60
# 休息被攔就記不成法術，`C）施法` 那一頁因此可能開不起來。開不起來就**跳過這兩張**
# ——比對程式會把它們記成「未量」，不會把整份報表弄丟（#42／#51）。
# 要拍到它們得找一個不會被攔的地方休息（貧民窟的房間），那是另一件事。
n=0
while test "$(screen)" != field-cast; do
  n=$((n+1))
  if test "$n" -gt 6; then echo "沒有拍到野外施法那兩張（休息被城衛隊攔下來，記不成法術）"; break; fi
  pulse c; sleep 0.3
done
if test "$(screen)" = field-cast; then
  shot remake-field-cast
  # 挑人那一頁開得起來，**法術清單那一頁要真的有記好的法術**才開得了；
  # 休息被城衛隊攔掉就沒有。開不了就只少這一張。
  n=0
  while test "$(screen)" != field-cast-spell; do
    n=$((n+1))
    if test "$n" -gt 6; then echo "沒有拍到法術清單那一張（沒有記好的法術）"; break; fi
    pulse Return; sleep 0.3
  done
  if test "$(screen)" = field-cast-spell; then
    shot remake-field-cast-spell
  fi
fi

python3 /tools/dos-parity-compare.py /ref /out /ref-cityhall /ref-campquit /ref-temple /ref-shop /ref-spells
'
echo "報告 → $OUT"

# **比較的對象是基準表，不是上一次跑的結果。** 與上一次比在每一輪都成立，卻永遠
# 不會發現「表上的數字與現在的程式差了十項」（2026-09-18 就是這樣漂開的，#51）。
# 擷圖停在半路時上面那段會非零退出，所以這一步照樣要能單獨跑：
#   tools/go.sh run ./cmd/pool-parity-check -run <OUT>/parity.json
if [[ -f "$OUT/parity.json" ]]; then
  # `tools/go.sh` 在容器裡跑，看得到的是 repo 相對路徑；傳絕對主機路徑會
  # 「no such file」。
  "$ROOT/tools/go.sh" run ./cmd/pool-parity-check -run "${OUT#$ROOT/}/parity.json" || {
    echo "與基準表對不上（上面列出哪幾欄）。確認是改好還是改壞之後，" >&2
    echo "用 tools/go.sh run ./cmd/pool-parity-check -run ${OUT#$ROOT/}/parity.json -write 更新表。" >&2
    exit 1; }
else
  echo "沒有 $OUT/parity.json，這一次沒有對到基準表。" >&2
  exit 1
fi
