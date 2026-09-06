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
sleep 0.6
shot docs/screenshots/pool-remake-chinese-portrait.png
# combat icon editor 是巢狀選單（spec 003 第 7..10 步）：PARTS → HEAD →
# NEXT → KEEP → EXIT 回到頂層，**頂層再按一次 EXIT 才進確認頁**。
# 先前這裡沿用扁平版的 `Return`，結果整條流程停在圖示編輯器裡，
# 後面每一張都拍到同一個畫面——而 `cmp` 只擋得住「兩張一樣」，
# 擋不住「兩張都是錯的畫面」。
pulse k
sleep 0.8
for key in p h n k e; do
  pulse "$key"
done
sleep 0.5
shot docs/screenshots/pool-remake-chinese-icon.png
pulse e
sleep 0.6
pulse y
sleep 0.8
pulse a
sleep 0.8
shot docs/screenshots/pool-remake-chinese-party.png
pulse b
sleep 1.5
shot docs/screenshots/pool-remake-chinese-tour.png
if cmp -s docs/screenshots/pool-remake-chinese-party.png docs/screenshots/pool-remake-chinese-tour.png; then
  echo "B did not begin the adventure" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
# 把 34 步導覽按完走到自由移動，抓第一人稱視野與平面圖各一張——
# 那兩張是繁中介面配原版素材最說明問題的畫面。
# 導覽是 34 步腳本移動加七頁文字，按不完就還在事件裡——**事件沒結束時
# `J`／`I`／`K` 會被 adventureCommandInput 先吃掉**，後面那幾張就會全部
# 拍到同一個第一人稱畫面。按到有餘裕再往下走。
for _ in $(seq 1 80); do
  pulse Return
done
sleep 2
shot docs/screenshots/pool-remake-chinese-movement.png
if cmp -s docs/screenshots/pool-remake-chinese-tour.png docs/screenshots/pool-remake-chinese-movement.png; then
  echo "the guided tour never reached free movement" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
pulse a
sleep 0.8
shot docs/screenshots/pool-remake-chinese-map.png
if cmp -s docs/screenshots/pool-remake-chinese-movement.png docs/screenshots/pool-remake-chinese-map.png; then
  echo "A did not open the area map" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
pulse a
sleep 0.5
# **順序有意義**：`J`／`I`／`K` 三個面板都擋在 `a.tactical == nil` 後面，
# 而 F5 關掉戰術預覽時只清 `tacticalPreview`、**不清 `a.tactical`**——
# 開過一次盤面之後那三個面板就再也叫不出來。所以先拍面板，最後才拍盤面。
#
# J 開探險者手冊，再打 46 + ENTER 跳到遊戲文字實際引用的那一條——
# 「抄進手冊，成為線索報導 46」在畫面上說得出口，這裡就要翻得到。
pulse j
sleep 0.8
shot docs/screenshots/pool-remake-chinese-journal.png
if cmp -s docs/screenshots/pool-remake-chinese-movement.png docs/screenshots/pool-remake-chinese-journal.png; then
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
# 三個面板都用自己的字母開關（`J`／`I`／`K`）。**不要用 ESC**——
# 冒險畫面的 ESC 是「回隊伍管理選單」，一按就掉出整條路徑。
# 這一隊剛建好、身上沒有東西，所以裝備頁看到的是空清單；拍它是為了確認
# 版面與字型，物品邏輯由 cmd/pool-game 的測試顧。
pulse j
sleep 0.5
pulse i
sleep 0.8
shot docs/screenshots/pool-remake-chinese-equipment.png
if cmp -s docs/screenshots/pool-remake-chinese-journal-46.png docs/screenshots/pool-remake-chinese-equipment.png; then
  echo "I did not open the equipment screen" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
# K 開法術一覽，TAB 翻到巫術第 1 級——那一頁 13 種，是最長的一組。
pulse i
sleep 0.5
pulse k
sleep 0.8
for step in 1 2 3; do
  pulse Tab
  sleep 0.3
done
sleep 0.5
shot docs/screenshots/pool-remake-chinese-spells.png
if cmp -s docs/screenshots/pool-remake-chinese-equipment.png docs/screenshots/pool-remake-chinese-spells.png; then
  echo "K did not open the spell list" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
# 最後才開戰術盤面：確認那一頁在漢字字型下四行資訊與功能鍵列都不相疊。
pulse k
sleep 0.5
pulse F5
sleep 0.8
shot docs/screenshots/pool-remake-chinese-tactical.png
if cmp -s docs/screenshots/pool-remake-chinese-spells.png docs/screenshots/pool-remake-chinese-tactical.png; then
  echo "F5 did not open the tactical board" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
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
    ("pool-remake-chinese-menu.png", "party creation menu"),
    ("pool-remake-chinese-race.png", "race picker"),
    ("pool-remake-chinese-class.png", "class picker"),
    ("pool-remake-chinese-sheet.png", "character sheet, keep-or-reroll prompt"),
    ("pool-remake-chinese-portrait.png", "portrait editor"),
    ("pool-remake-chinese-icon.png", "combat icon editor, top level"),
    ("pool-remake-chinese-party.png", "party creation menu with one member added"),
    ("pool-remake-chinese-tour.png", "first page of the Rolf guided tour"),
    ("pool-remake-chinese-movement.png", "free movement after the tour reaches its ECL exit"),
    ("pool-remake-chinese-map.png", "area map"),
    ("pool-remake-chinese-journal.png", "adventurer's journal"),
    ("pool-remake-chinese-journal-46.png", "journal clue 46"),
    ("pool-remake-chinese-equipment.png", "equipment screen, empty pack"),
    ("pool-remake-chinese-spells.png", "spell list, magic-user level 1"),
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
    data = open(os.path.join(root, path), "rb").read()
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
    "state_relation": ("normal-remake-player-path with -lang zh: title, ENTER, C, the whole "
                       "creation flow, A, B, the guided tour, free movement, then the "
                       "J/I/K panels and F5. Not an original-DOS parity claim."),
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
