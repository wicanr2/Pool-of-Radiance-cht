#!/usr/bin/env bash
# 把 Amiga 版的配樂渲染成 remake 用的 OGG。
#
# 用法：tools/build-amiga-music.sh
#
# 輸入是 UnExoticA 的 disk rip `wb.Pool_of_Radiance`（Wally Beben 的自訂
# 播放器模組，75446 bytes），放在 workplace/amiga-music/Pool_of_Radiance/。
# 沒有的話先從 https://www.exotica.org.uk/wiki/Pool_of_Radiance 取得
# `Game/Beben_Wally/Pool_of_Radiance.lha` 解開。
#
# **那份音訊是第三方著作權**（Wally Beben，1990 U.S. Gold／SSI）：
# workplace/ 與 dist-all/ 都在 .gitignore 裡，可散布的 patch 發行包不帶它，
# 只有本機的 full-local 包會放進去。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="$ROOT/workplace/amiga-music"
MODULE="$WORK/Pool_of_Radiance/wb.Pool_of_Radiance"
IMAGE="${POOL_UADE_IMAGE:-pool-uade:1}"

test -f "$MODULE" || { echo "找不到模組：$MODULE" >&2; exit 2; }
docker image inspect "$IMAGE" >/dev/null

# 模組的大小是 ExoticA 著錄的 75446——對不上就不是同一份 rip。
size="$(stat -c %s "$MODULE")"
if [ "$size" != "75446" ]; then
  echo "模組是 $size bytes，ExoticA 著錄的是 75446：不是同一份 rip" >&2
  exit 1
fi

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -v "$WORK:/m" -w /m "$IMAGE" sh -c '
set -eu
export HOME=/tmp/uade-home; mkdir -p "$HOME" wav ogg
# UADE 讀得出模組裡有幾首；把它印出來當紀錄。
uade123 -g Pool_of_Radiance/wb.Pool_of_Radiance 2>&1 | grep -E "subsongs|playername"
for n in 1 2 3 4 5 6; do
  # -1 只放一首、-w 240 每首最多渲染 240 秒、-y 6 靜音 6 秒就收。
  # 會循環的那兩首會用滿 240 秒，那是截斷不是自然結尾。
  uade123 -1 -s "$n" -w 240 -y 6 -f "wav/por-amiga-sub$n.wav" -e wav \
    Pool_of_Radiance/wb.Pool_of_Radiance >/dev/null 2>&1
  ffmpeg -hide_banner -loglevel error -y -i "wav/por-amiga-sub$n.wav" \
    -c:a libvorbis -q:a 5 "ogg/por-amiga-$(printf %02d "$n").ogg"
done
'
echo "OGG 在 $WORK/ogg："
ls -la "$WORK/ogg"
