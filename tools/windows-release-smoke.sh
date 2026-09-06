#!/usr/bin/env bash
# 在 Docker／Wine／Xvfb 裡啟動打好的 Windows 版並截圖。
# 用法：tools/windows-release-smoke.sh <版本> [patch|full-local]
#
# 「交叉編得出 .exe」不等於「在 Windows 上啟得動」：少一個執行期相依、
# 資源路徑寫死成 Linux 形狀、或是視窗根本開不起來，在建置階段全部看不出來。
# 這支把那個差別量出來。
#
# **這不能取代真機驗收。** Wine 不是 Windows：驅動、字型後備、DPI 縮放與
# 檔案總管的行為都不同，`docs/verification/real-machine-startup-checklist.md`
# 那份清單仍然要在真的 Windows 上跑一次。這支證的是「不是連跑都跑不起來」。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
FLAVOUR="${2:-full-local}"
IMAGE="${WINE_IMAGE:-coab-wine-smoke:ubuntu-noble-20260826}"
FONT_DIR="${ETEN_FONT_DIR:-/home/anr2/cht/etan_font}"
[[ -n "$VERSION" ]] || { echo "用法：tools/windows-release-smoke.sh <版本> [patch|full-local]" >&2; exit 2; }
case "$FLAVOUR" in patch|full-local) ;; *) echo "口味只有 patch 或 full-local" >&2; exit 2;; esac

RELEASE="$ROOT/dist-all/$VERSION/$FLAVOUR/windows"
SHOT="$ROOT/docs/screenshots/pool-release-windows-wine.png"
test -f "$RELEASE/pool-game.exe"
test -f "$ROOT/Pool of Radiance (1988).zip"
test -f "$FONT_DIR/stdfont.15"
docker image inspect "$IMAGE" >/dev/null

mkdir -p "$(dirname "$SHOT")"
rm -f "$SHOT"
docker run --rm --network none --memory 3g --cpus 2 --pids-limit 512 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" --tmpfs /tmp/.X11-unix:rw,mode=1777 \
  -e HOME=/tmp/wine-home -e WINEPREFIX=/tmp/wine-prefix -e WINEDEBUG=-all \
  -e FLAVOUR="$FLAVOUR" \
  -v "$RELEASE:/release:ro" -v "$FONT_DIR:/fonts:ro" \
  -v "$ROOT/Pool of Radiance (1988).zip:/assets/game.zip:ro" \
  -v "$(dirname "$SHOT"):/shots" -w /tmp "$IMAGE" sh -c '
set -eu
# 唯讀掛載的發行包不能當工作目錄——遊戲要在旁邊寫存檔。
# 工作目錄也不能用 `docker run -w` 現開的：那是 root 建的，我們寫不進去。
run=$HOME/run
mkdir -p "$HOME" "$WINEPREFIX" "$run"
cp -r /release/. "$run/"
cp /assets/game.zip "$run/Pool of Radiance (1988).zip"
Xvfb :99 -screen 0 1200x800x24 >/tmp/xvfb.log 2>&1 &
xvfb=$!
game=
finish() {
  test -z "$game" || kill "$game" 2>/dev/null || true
  kill "$xvfb" 2>/dev/null || true
}
trap finish EXIT
n=0
until test -S /tmp/.X11-unix/X99; do
  n=$((n+1)); test "$n" -lt 50 || { cat /tmp/xvfb.log; exit 1; }
  sleep 0.1
done
export DISPLAY=:99
cd "$run"
# **先讓 Wine 把 prefix 建好。** 第一次啟動要跑 wineboot，那段時間遊戲還沒
# 開始畫；不先做的話等再久截到的都是全黑，看起來像「畫不出來」。
/usr/lib/wine/wine64 wineboot -u >/tmp/wineboot.log 2>&1 || true
(exec /usr/lib/wine/wine64 pool-game.exe -lang zh -eten-font Z:\\fonts\\stdfont.15) \
  >/tmp/game.log 2>&1 &
game=$!
sleep 20
if ! kill -0 "$game" 2>/dev/null; then
  echo "pool-game.exe 在畫出標題畫面之前就結束了" >&2
  cat /tmp/game.log >&2
  exit 1
fi
# 發行包的內容檢查與 Linux 那一支同一組判準。
oggs=$(find "$run" -name "*.ogg" | wc -l)
if test "$FLAVOUR" = full-local; then
  test "$oggs" -eq 6 || { echo "full-local 只有 $oggs 個 OGG，應該是 6 個" >&2; exit 1; }
elif test "$oggs" -ne 0; then
  echo "可散布的 patch 包裡有 $oggs 個 OGG：那是第三方著作權，不該在裡面" >&2
  exit 1
fi
echo "OGG $oggs 個（口味 $FLAVOUR）"
# Xvfb 沒有視窗管理員，抓視窗 id 直接截。
window=$(xdotool search --sync --onlyvisible --name "Pool of Radiance Remake" | head -1)
eval "$(xdotool getwindowgeometry --shell "$window")"
sleep 2
# **從 root 抓再裁**，不要 `import -window <id>`：沒有視窗管理員時直接抓
# 視窗會拿到一張全黑的圖（內容在 root 上，不在那個 drawable 裡）。
import -window root -crop "${WIDTH}x${HEIGHT}+${X}+${Y}" +repage \
  /shots/pool-release-windows-wine.png
mean=$(identify -format "%[fx:mean]" /shots/pool-release-windows-wine.png)
echo "Wine 下啟動成功；視窗 $window ${WIDTH}x${HEIGHT}，畫面平均亮度 $mean"
# 全黑代表「視窗開了但沒畫東西」——那和啟動失敗一樣要擋下來。
awk -v m="$mean" "BEGIN{exit (m > 0.01) ? 0 : 1}" || {
  echo "畫面是全黑的：視窗開了但沒有畫出東西" >&2
  cat /tmp/game.log >&2
  exit 1
}
'
test -s "$SHOT"
sha256sum "$SHOT"
