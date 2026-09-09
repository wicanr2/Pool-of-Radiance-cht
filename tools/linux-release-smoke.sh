#!/usr/bin/env bash
# 在 Docker／Xvfb 裡啟動打好的 AppImage 並截圖。
# 用法：tools/linux-release-smoke.sh <版本> [patch|full-local]
#
# 「建得出來」不等於「啟得動」：AppImage 少一個執行期依賴時，建置階段一切正常，
# 到玩家手上才是啟動失敗。這支就是拿來把那個差別量出來的。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
FONT_DIR="${ETEN_FONT_DIR:-/home/anr2/cht/etan_font}"
FLAVOUR="${2:-patch}"
[[ -n "$VERSION" ]] || { echo "用法：tools/linux-release-smoke.sh <版本> [patch|full-local]" >&2; exit 2; }
case "$FLAVOUR" in patch|full-local) ;; *) echo "口味只有 patch 或 full-local" >&2; exit 2;; esac
APPIMAGE="dist-all/$VERSION/$FLAVOUR/pool-of-radiance-remake-$VERSION-x86_64.AppImage"
test -f "$ROOT/$APPIMAGE"
test -f "$ROOT/Pool of Radiance (1988).zip"
test -f "$FONT_DIR/stdfont.15"

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" --tmpfs /tmp/.X11-unix:rw,mode=1777 \
  -e HOME=/tmp/home -e APPIMAGE_PATH="/src/$APPIMAGE" -e FLAVOUR="$FLAVOUR" \
  -v "$ROOT:/src" -v "$FONT_DIR:/fonts:ro" -w /src \
  wasteland-go:1.24-x11-record-r1 bash -c '
set -eu
mkdir -p "$HOME" /tmp/run docs/screenshots
Xvfb :99 -screen 0 1400x900x24 >/tmp/xvfb.log 2>&1 &
xvfb_pid=$!
game_pid=
finish() {
  test -z "$game_pid" || kill "$game_pid" 2>/dev/null || true
  kill "$xvfb_pid" 2>/dev/null || true
}
trap finish EXIT
export DISPLAY=:99
until test -S /tmp/.X11-unix/X99; do sleep 0.1; done

# 容器裡沒有 FUSE，用 AppImage 自帶的解壓模式啟動；驗的是包內容與相依，
# 不是 FUSE 掛載本身。
cp "$APPIMAGE_PATH" /tmp/run/game.AppImage
cp "/src/Pool of Radiance (1988).zip" /tmp/run/
cd /tmp/run
./game.AppImage --appimage-extract >/tmp/extract.log 2>&1
(exec ./squashfs-root/AppRun -lang zh -eten-font /fonts/stdfont.15) >/tmp/game.log 2>&1 &
game_pid=$!
sleep 6
if ! kill -0 "$game_pid" 2>/dev/null; then
  echo "AppImage exited before the title screen" >&2
  cat /tmp/game.log >&2
  exit 1
fi
# 配樂：full-local 那一包必須帶滿六個 OGG，而且遊戲不能因為它們而抱怨。
# 容器裡沒有音訊裝置，所以「有沒有真的出聲」不在這裡驗——那由
# internal/music 的對拍測試顧（解回 PCM 與渲染來源比相關係數）。
oggs=$(find ./squashfs-root -name "*.ogg" | wc -l)
if test "$FLAVOUR" = full-local; then
  if test "$oggs" -ne 6; then
    echo "full-local 只有 $oggs 個 OGG，應該是 6 個" >&2
    exit 1
  fi
elif test "$oggs" -ne 0; then
  echo "可散布的 patch 包裡有 $oggs 個 OGG：那是第三方著作權，不該在裡面" >&2
  exit 1
fi
# 容器裡沒有音訊裝置，所以 full-local 會走「這一次不放音樂」那條——
# **那一行是預期的，而且正好證明事前判斷有效**：玩家的機器沒有音效卡時
# 遊戲要照常開得起來，不是打不開。其餘的 music: 行（讀不到檔、解不開）才是問題。
if grep "^music:" /tmp/game.log | grep -qv "找不到音訊裝置"; then
  echo "配樂載入有問題：" >&2
  grep "^music:" /tmp/game.log | grep -v "找不到音訊裝置" >&2
  exit 1
fi
if test "$FLAVOUR" = full-local; then
  grep -q "找不到音訊裝置" /tmp/game.log \
    && echo "沒有音訊裝置，已照事前判斷跳過音樂（這是預期的）" \
    || echo "有音訊裝置，音樂已啟用"
fi
echo "OGG $oggs 個（口味 $FLAVOUR）"
# Xvfb 沒有視窗管理員，windowactivate 會失敗；抓視窗的幾何再從畫面上截。
window=$(xdotool search --sync --onlyvisible --name "Pool of Radiance Remake" | head -1)
eval "$(xdotool getwindowgeometry --shell "$window")"
sleep 1
ffmpeg -y -hide_banner -loglevel error -f x11grab \
  -video_size "${WIDTH}x${HEIGHT}" -i ":99+${X},${Y}" -frames:v 1 \
  /src/docs/screenshots/pool-release-linux-appimage.png
echo "AppImage started; window $window ${WIDTH}x${HEIGHT}"
'
sha256sum "$ROOT/docs/screenshots/pool-release-linux-appimage.png"
