#!/usr/bin/env bash
# 以真實 Ebitengine 視窗拍下繁中畫面。
#
# 倚天字型是第三方資產、不進 repo：這裡以唯讀掛載從主機路徑帶進容器，
# 路徑可由 ETEN_FONT_DIR 覆寫。沒有字型時遊戲會失敗即關閉，不會默默用英文跑，
# 所以這支腳本拍不到圖就是拍不到，不會拍出一張看起來對的英文畫面。
#
# **走位靠遊戲自己回報的畫面識別字**（`-screen-state`，見
# `cmd/pool-game/screen_state.go`），不是靠「按幾下、睡幾秒」。
# 舊版是盲按：漏掉一次 Return 時前後兩張圖仍然不一樣（畫面確實動了，
# 只是動到別的地方），`cmp` 判定通過，整條流程於是偏掉十幾步，
# 最後以「A 打不開平面圖」的形式炸出來——真正的原因早就發生在建角途中。
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
STATE=/tmp/pool-screen
rm -f "$STATE"
(cd /tmp && exec /tmp/pool-game -zip "/src/Pool of Radiance (1988).zip" \
   -lang zh -eten-font /fonts/stdfont.15 -screen-state "$STATE") >/tmp/game.log 2>&1 &
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

screen() { cat "$STATE" 2>/dev/null | tr -d "\n"; }
die() {
  echo "$1" >&2
  echo "目前畫面：$(screen)" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
}
# await <畫面> [輪數]：等遊戲自己回報走到了那個畫面。
await() {
  want=$1
  rounds=${2:-100}
  n=0
  while test "$(screen)" != "$want"; do
    sleep 0.1
    n=$((n + 1))
    test "$n" -lt "$rounds" || die "等不到畫面 $want"
  done
}
pulse() {
  xdotool keydown "$1"
  sleep 0.12
  xdotool keyup "$1"
  sleep 0.12
}
# step <鍵> <目標畫面>：按到走到目標為止。每按一次先等一段時間再決定要不要
# 重按——不等就重按會把 A、J、I、K 這種開關鍵按回去，看起來像「按了沒反應」。
step() {
  key=$1
  want=$2
  attempt=0
  while test "$(screen)" != "$want"; do
    attempt=$((attempt + 1))
    test "$attempt" -le 40 || die "按 $key 走不到 $want"
    pulse "$key"
    waited=0
    while test "$(screen)" != "$want" && test "$waited" -lt 15; do
      sleep 0.1
      waited=$((waited + 1))
    done
  done
}
shot() {
  ffmpeg -y -hide_banner -loglevel error -f x11grab -video_size "${WIDTH}x${HEIGHT}" \
    -i ":99+${X},${Y}" -frames:v 1 "$1"
}
# differs <前> <後> <說明>：同一個畫面識別字底下的內容變化（加入隊伍、
# 翻到某一條線索）只能靠比圖，這一個留給那幾處。
differs() {
  if cmp -s "$1" "$2"; then
    die "$3"
  fi
}

await title 200
sleep 0.5
shot docs/screenshots/pool-remake-chinese-title.png
step Return menu
sleep 0.4
shot docs/screenshots/pool-remake-chinese-menu.png

# C 進建角：種族 → 性別 → 職業 → 陣營 → 屬性表，各拍一張能證明選單項目
# 也是中文的。每一站都等遊戲回報自己到了，不用猜要按幾下。
step c creation-race
# **這一隊建的是牧師，不是戰士。** 挑法術那一頁（spec 134）要有記得起來的
# 法術才走得到，而種族清單的第一個是矮人——矮人當不了施法職業。往下五格
# 到人類，人類的職業清單第一個就是牧師，所以職業那一頁不用再動游標。
# 清單用 `End` 往下：原版的清單就是 Home／End，沒有 Up／Down（spec 133）。
for _ in 1 2 3 4 5; do pulse End; done
sleep 0.4
shot docs/screenshots/pool-remake-chinese-race.png
step Return creation-class
sleep 0.4
shot docs/screenshots/pool-remake-chinese-class.png
step Return creation-alignment
sleep 0.4
shot docs/screenshots/pool-remake-chinese-alignment.png
step Return creation-roll
sleep 0.5
shot docs/screenshots/pool-remake-chinese-sheet.png

step Return creation-name
sleep 0.3
xdotool type --delay 120 HERO
sleep 0.3
step Return creation-portrait
sleep 0.4
shot docs/screenshots/pool-remake-chinese-portrait.png

# combat icon editor 是巢狀選單（spec 003 第 7..10 步）：PARTS → HEAD →
# NEXT → KEEP → EXIT 回到頂層，**頂層再按一次 EXIT 才進確認頁**。
# 識別字帶著層號（`creation-icon-<層>`），所以走錯層會當場停住。
step k creation-icon-0
step p creation-icon-1
step h creation-icon-2
pulse n
step k creation-icon-1
step e creation-icon-0
sleep 0.4
shot docs/screenshots/pool-remake-chinese-icon.png
step e creation-icon-confirm
step y menu

# A 把人物加進隊伍。畫面識別字仍然是 menu（原版也是留在同一頁），
# 所以這一步只能比圖。
sleep 0.4
pulse a
sleep 0.6
shot docs/screenshots/pool-remake-chinese-party.png
differs docs/screenshots/pool-remake-chinese-menu.png \
        docs/screenshots/pool-remake-chinese-party.png "A 沒有把人物加進隊伍"

step b adventure-intro
sleep 0.6
shot docs/screenshots/pool-remake-chinese-tour.png
# 導覽是 34 步腳本移動加七頁文字；只有文字那幾頁在等 Return，其餘自己走。
# 走完才是自由移動，指令列那時才會出現。
step Return adventure-move 60
sleep 0.6
shot docs/screenshots/pool-remake-chinese-movement.png

step a adventure-map
sleep 0.5
shot docs/screenshots/pool-remake-chinese-map.png
step a adventure-move

# **順序有意義**：`J`／`I`／`K` 三個面板都擋在 `a.tactical == nil` 後面，
# 而 F5 關掉戰術預覽時只清 `tacticalPreview`、**不清 `a.tactical`**——
# 開過一次盤面之後那三個面板就再也叫不出來。所以先拍面板，最後才拍盤面。
#
# J 開探險者手冊，再打 46 + ENTER 跳到遊戲文字實際引用的那一條——
# 「抄進手冊，成為線索報導 46」在畫面上說得出口，這裡就要翻得到。
step j journal-clue
sleep 0.5
shot docs/screenshots/pool-remake-chinese-journal.png
pulse 4
pulse 6
pulse Return
sleep 0.6
shot docs/screenshots/pool-remake-chinese-journal-46.png
differs docs/screenshots/pool-remake-chinese-journal.png \
        docs/screenshots/pool-remake-chinese-journal-46.png "打 46 沒有跳到線索 46"
# 附錄是說明書書末那七節規則表（金錢換算、法術表、武器表…），TAB 翻到它。
step Tab journal-appendix
sleep 0.4
shot docs/screenshots/pool-remake-chinese-journal-appendix.png

# 三個面板都用自己的字母開關（`J`／`I`／`K`）。**不要用 ESC**——
# 冒險畫面的 ESC 是「回隊伍管理選單」，一按就掉出整條路徑。
# 這一隊剛建好、身上沒有東西，所以裝備頁看到的是空清單；拍它是為了確認
# 版面與字型，物品邏輯由 cmd/pool-game 的測試顧。
step j adventure-move
step i equipment
sleep 0.5
shot docs/screenshots/pool-remake-chinese-equipment.png
# K 開法術一覽，TAB 翻到巫術第 1 級——那一頁 13 種，是最長的一組。
# **組別要等畫面自己報**：這裡本來盲按三次 TAB，而掉一次鍵的症狀是拍到
# 神術第 3 級，標題白紙黑字寫著別的級別卻沒有任何一步失敗。
step i adventure-move
step k spells-cleric-1
step Tab spells-magic-user-1
sleep 0.5
shot docs/screenshots/pool-remake-chinese-spells.png
# 拍完巫術第 1 級再繞回神術第 1 級（六組，TAB 一直按就會轉回去），
# 用 `M` 記第一條祝福術。記了還不算數——法術要休息夠久才會 ready
#（spec 070），所以下一步去紮營。
step Tab spells-cleric-1
sleep 0.3
pulse m
sleep 0.4
shot docs/screenshots/pool-remake-chinese-memorise.png
# F3 是遊戲內攻略。剛走完導覽只有幾格是走過的，所以先拍霧的那一張，
# 再按兩次 V 攤開（第一次只出警告）拍完整的那一張。
# C 是探索畫面的施法（spec 119）：挑人 → 挑法術 → 挑目標。
step k adventure-move
# 紮營。原版按 `E` 不彈選單：視野換成營火、指令列換成
# `CAMP: SAVE VIEW MAGIC REST ALTER EXIT`（spec 135）。
step e camp
sleep 0.4
shot docs/screenshots/pool-remake-chinese-camp.png
# `R` 進排時間那一層，指令列再換成 `REST DAYS HOURS MINS INC DEC EXIT`。
step r camp-rest
sleep 0.4
shot docs/screenshots/pool-remake-chinese-camp-rest.png
# 休息兩小時。祝福術是第 1 級，記完要一小時（overlay-20 entry 15 每小時
# 把記錄 `+2Ch` 減一，spec 114）；多排一小時是留餘裕，不是規則。
# `H` 選到小時欄、`I` 加一、`R` 開始休息——休息完自己回到自由移動。
pulse h
pulse i
pulse i
sleep 0.3
pulse r
await adventure-move 60
sleep 0.4
step c field-cast
sleep 0.4
shot docs/screenshots/pool-remake-chinese-field-cast.png
# 挑完人就是整頁的法術清單（spec 134）。**走得到這裡等於前面三件事都成立**：
# 建的是施法職業、`M` 把法術記進去了、紮營休息讓它變成可施展。少任何一件，
# 按下去只會得到「沒有記憶法術」，這一步就會停在 field-cast 等到逾時。
step Return field-cast-spell
sleep 0.5
shot docs/screenshots/pool-remake-chinese-spell-page.png
step Escape field-cast
step Escape adventure-move
# F4 是素材總覽：肖像、戰鬥造形、牆面圖塊與外框符號各一排。
step F4 sprites
sleep 0.5
shot docs/screenshots/pool-remake-chinese-sprites.png
step Tab sprites-monsters
sleep 0.5
shot docs/screenshots/pool-remake-chinese-monsters.png
step Tab sprites-effects
sleep 0.5
shot docs/screenshots/pool-remake-chinese-effects.png
step Tab sprites-terrain
sleep 0.5
shot docs/screenshots/pool-remake-chinese-terrain.png
step Escape adventure-move
# V 是探索畫面的人物資料頁（spec 119 → spec 130）。
step v view-pick
step Return view-sheet
sleep 0.4
shot docs/screenshots/pool-remake-chinese-view-sheet.png
step Escape view-pick
step Escape adventure-move
step F3 guide
sleep 0.5
shot docs/screenshots/pool-remake-chinese-guide.png
# **兩次 `V` 都要等狀態，不能盲按。** 盲按時漏掉其中一次的症狀是
# 「停在 guide 等不到 guide-full」，看起來像 `V` 沒接上，實際上是按鍵掉了。
step v guide-warned
step v guide-full
sleep 0.5
shot docs/screenshots/pool-remake-chinese-guide-full.png
step Escape adventure-move
# 市政廳外牆一次報四則公告字號。remake 把手冊收進遊戲，那一下 ENTER 直接
# 翻過去——玩家不必再按 J 自己查（spec 132）。
# 導覽在 (0,4) 結束、朝西（spec 076）：右轉兩次朝東，再往前走就到 (3,4)。
pulse Right
pulse Right
step Up adventure-cell-text
pulse Return
sleep 0.4
pulse Return
sleep 0.4
pulse Return
sleep 0.6
shot docs/screenshots/pool-remake-chinese-journal-cue.png
step Return journal-proclamation
sleep 0.6
shot docs/screenshots/pool-remake-chinese-journal-proclamation.png
step Escape adventure-cell-text
# 剩下三則翻完，回到自由移動。
n=0
while test "$(screen)" != "adventure-move" && test "$n" -lt 30; do
  case "$(screen)" in
    journal-*) pulse Escape ;;
    *) pulse Return ;;
  esac
  n=$((n + 1))
done
test "$(screen)" = "adventure-move" || die "市政廳那一段走不回自由移動"

# 最後才開戰術盤面：確認那一頁在漢字字型下四行資訊與功能鍵列都不相疊。
step F5 tactical
sleep 0.6
shot docs/screenshots/pool-remake-chinese-tactical.png
# 再走到**真的打起來的那一場**，拍一張有敵方造形的。遭遇是隨機的，所以這裡
# 輪流四個方向走，中途碰到格子事件就按 Return 讓它跑完；走不到就跳過這一張
# ——那一張是加分項，不該讓整支腳本失敗。
pulse F5
sleep 0.4
n=0
while test "$(screen)" != "combat-staged" && test "$n" -lt 400; do
  case "$(screen)" in
    adventure-cell-menu|adventure-cell-text|adventure-intro|adventure-tour)
      pulse Return ;;
    adventure-move)
      # **方向鍵是「左右轉向、上前進」**，不是四方向移動。輪流按四個方向的話
      # 四分之三的按鍵都在原地轉，走不出去。這裡走五步轉一次。
      case $((n % 6)) in
        5) pulse Right ;;
        *) pulse Up ;;
      esac ;;
    *) pulse Escape ;;
  esac
  n=$((n + 1))
done
if test "$(screen)" = "combat-staged"; then
  step Return tactical
  sleep 0.6
  shot docs/screenshots/pool-remake-chinese-combat.png
  echo "走了 $n 步撞上一場架"
else
  echo "走了 $n 步沒遇到架，跳過實際戰鬥那一張" >&2
fi
sha256sum docs/screenshots/pool-remake-chinese-*.png
'

# 清冊跟著重生：手維護的雜湊表只要漏更新一次，之後每一份報告都在引用一個
# 已經不存在的畫面（rulebook 63）。這裡直接由剛拍好的檔案寫出去。
python3 - "$ROOT" "$FONT_DIR" <<'PY'
import hashlib
import json
import os
import struct
import subprocess
import sys

root, font_dir = sys.argv[1], sys.argv[2]
screens = [
    ("pool-remake-chinese-title.png", "title screen"),
    ("pool-remake-chinese-menu.png", "party creation menu"),
    ("pool-remake-chinese-race.png", "race picker"),
    ("pool-remake-chinese-class.png", "class picker"),
    ("pool-remake-chinese-alignment.png", "alignment picker"),
    ("pool-remake-chinese-sheet.png", "character sheet, keep-or-reroll prompt"),
    ("pool-remake-chinese-portrait.png", "portrait editor"),
    ("pool-remake-chinese-icon.png", "combat icon editor, top level"),
    ("pool-remake-chinese-party.png", "party creation menu with one member added"),
    ("pool-remake-chinese-tour.png", "first page of the Rolf guided tour"),
    ("pool-remake-chinese-movement.png", "free movement after the tour reaches its ECL exit"),
    ("pool-remake-chinese-map.png", "area map"),
    ("pool-remake-chinese-journal.png", "adventurer's journal"),
    ("pool-remake-chinese-journal-46.png", "journal clue 46"),
    ("pool-remake-chinese-journal-appendix.png", "journal appendix: the manual's rule tables"),
    ("pool-remake-chinese-equipment.png", "equipment screen, empty pack"),
    ("pool-remake-chinese-spells.png", "spell list, magic-user level 1"),
    ("pool-remake-chinese-memorise.png", "spell list, cleric level 1, after M memorises Bless"),
    ("pool-remake-chinese-camp.png", "camp: the view becomes a fire, the command bar becomes CAMP:"),
    ("pool-remake-chinese-camp-rest.png", "camp, REST: the rest-time row"),
    ("pool-remake-chinese-field-cast.png", "casting outside combat (C on the command bar)"),
    ("pool-remake-chinese-spell-page.png", "the full-page memorised-spell list the caster picks from (spec 134)"),
    ("pool-remake-chinese-view-sheet.png", "a party member's sheet from the command bar (V)"),
    ("pool-remake-chinese-sprites.png", "sprite overview (F4): portraits, combat icons, wall pieces, frame symbols"),
    ("pool-remake-chinese-monsters.png", "board icons (F4, TAB): the 32 bodies monsters and characters share"),
    ("pool-remake-chinese-effects.png", "combat effects (F4, TAB twice): missiles and blasts from COMSPR.DAX"),
    ("pool-remake-chinese-terrain.png", "combat terrain tiles (F4, TAB three times): DUNGCOM/WILDCOM/RANDCOM"),
    ("pool-remake-chinese-guide.png", "in-game guide (F3), fogged to explored cells"),
    ("pool-remake-chinese-guide-full.png", "in-game guide (F3) with V, every point shown"),
    ("pool-remake-chinese-journal-cue.png", "the City Hall wall cites four proclamations; ENTER opens the journal there"),
    ("pool-remake-chinese-journal-proclamation.png", "the journal opened straight at the cited proclamation"),
    ("pool-remake-chinese-combat.png", "a real fight: enemy icons on the board"),
    ("pool-remake-chinese-tactical.png", "tactical board (F5)"),
]


def png_size(data):
    width, height = struct.unpack(">II", data[16:24])
    return width, height


def git(*args):
    return subprocess.run(["git", "-C", root, *args],
                          capture_output=True, text=True, check=True).stdout.strip()


shots = []
for name, screen in screens:
    path = os.path.join("docs/screenshots", name)
    full = os.path.join(root, path)
    if not os.path.exists(full):
        # 有幾張是加分項（例如要撞上一場架才拍得到的那一張）。缺了就跳過，
        # 不要讓整份清單產不出來。
        continue
    data = open(full, "rb").read()
    width, height = png_size(data)
    shots.append({"path": path, "sha256": hashlib.sha256(data).hexdigest(),
                  "width": width, "height": height, "screen": screen})

present = sorted(f for f in ("stdfont.15", "ascfont.15", "spcfont.15")
                 if os.path.exists(os.path.join(font_dir, f)))
absent = [f for f in ("stdfont.15", "ascfont.15", "spcfont.15") if f not in present]

manifest = {
    "schema": "pool-remake-screenshot-manifest/1",
    "captured_at": git("log", "-1", "--format=%cs"),
    "source_commit": git("rev-parse", "HEAD"),
    "source_tree_dirty": bool(git("status", "--porcelain")),
    "method": "Docker/Xvfb real Ebitengine window; tools/capture-chinese-menu.sh",
    "state_relation": ("normal-remake-player-path with -lang zh, stepped by the game's own "
                       "-screen-state identifiers: title, ENTER, C, the whole creation flow, "
                       "A, B, the guided tour, free movement, then the J/I/K panels and F5. "
                       "Not an original-DOS parity claim."),
    "font": {
        "family": "ETen 16x15 Big5 bitmap",
        "files_present": present,
        "files_absent": absent,
        "in_repository": False,
    },
    "screenshots": shots,
}
target = os.path.join(root, "docs/audit/remake-chinese-screenshot-manifest.json")
with open(target, "w", encoding="utf-8") as handle:
    json.dump(manifest, handle, ensure_ascii=False, indent=2)
    handle.write("\n")
print("manifest →", target)
PY
