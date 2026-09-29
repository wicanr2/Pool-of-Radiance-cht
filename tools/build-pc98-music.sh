#!/usr/bin/env bash
# 把 PC-98 版（Pony Canyon 1989）的 15 首 YM2203 配樂渲染成 remake 用的 OGG
# 與循環點清單（spec 169）。
#
# 用法：tools/build-pc98-music.sh
#
# 輸入是 PC-98 版開機碟（Disk A）上的 `MSCDRV.EXE`，放在
# workplace/pc98-music/MSCDRV.EXE。沒有的話從 POOL_PC98_MSCDRV 指的檔案複製
#（預設找 pc98golem 已經從 .d88 取出的那一份）。SHA-256 對不上就停。
#
# 輸出：workplace/pc98-music/wav/（中間檔）與 workplace/pc98-music/ogg/
#（pc98-01.ogg..pc98-15.ogg ＋ loops.json）。
#
# **那 15 首是 Pony Canyon 的著作權**：workplace/ 在 .gitignore 裡，
# 可散布的 patch 發行包不帶，只有本機的 full-local 包會放進去。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="$ROOT/workplace/pc98-music"
DRIVER="$WORK/MSCDRV.EXE"
SOURCE="${POOL_PC98_MSCDRV:-$ROOT/../../pc98golem/workplace/pool-pc98/diskA/MSCDRV.EXE}"
IMAGE="${POOL_FFMPEG_IMAGE:-pool-uade:1}"
# 手上那一份 PC-98 版（Disk A [WizV5]）的 MSCDRV.EXE。
WANT_SHA256=50118db5588097063dbd5923b53fecc02030a72c4d027df42269c72266105dc8

mkdir -p "$WORK"
if ! test -f "$DRIVER"; then
  test -f "$SOURCE" || { echo "找不到 MSCDRV.EXE：$DRIVER 或 $SOURCE" >&2; exit 2; }
  cp "$SOURCE" "$DRIVER"
fi
got="$(sha256sum "$DRIVER" | cut -d' ' -f1)"
if test "$got" != "$WANT_SHA256"; then
  echo "MSCDRV.EXE 的 SHA-256 是 $got，預期 $WANT_SHA256：不是同一份驅動" >&2
  exit 1
fi
docker image inspect "$IMAGE" >/dev/null

rm -f "$WORK"/wav/pc98-*.wav "$WORK"/wav/loops.json
"$ROOT/tools/go.sh" run ./cmd/pool-pc98-music \
  -driver workplace/pc98-music/MSCDRV.EXE -out workplace/pc98-music/wav

mkdir -p "$WORK/ogg"
rm -f "$WORK"/ogg/pc98-*.ogg "$WORK"/ogg/loops.json
timeout 1800 docker run --rm --name "poolP-pc98ogg-$$" --network none \
  --memory 2g --cpus 3 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -v "$WORK:/m" -w /m "$IMAGE" sh -c '
set -eu
for wav in wav/pc98-*.wav; do
  name="$(basename "$wav" .wav)"
  ffmpeg -hide_banner -loglevel error -y -i "$wav" -c:a libvorbis -q:a 4 "ogg/$name.ogg"
  # 循環點以樣本計，OGG 的樣本數要與 WAV 一樣，差一個就對不上。
  want="$(ffprobe -v error -select_streams a:0 -count_packets -show_entries stream=duration_ts -of csv=p=0 "$wav")"
  got="$(ffprobe -v error -select_streams a:0 -show_entries stream=duration_ts -of csv=p=0 "ogg/$name.ogg")"
  if test "$want" != "$got"; then
    echo "$name：WAV $want 個樣本，OGG $got 個" >&2
    exit 1
  fi
  volume="$(ffmpeg -hide_banner -nostats -i "ogg/$name.ogg" -af volumedetect -f null - 2>&1 |
    sed -n "s/.*\(mean_volume: .*\)/\1/p;s/.*\(max_volume: .*\)/\1/p" | tr "\n" " ")"
  echo "$name.ogg：$got 個樣本，$volume"
done
cp wav/loops.json ogg/loops.json
'
echo "OGG 與 loops.json 在 $WORK/ogg："
ls -la "$WORK/ogg"
