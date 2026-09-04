#!/usr/bin/env bash
# 從原版走到第一人稱畫面並逐步抓圖，供「同狀態 DOS 畫面對拍」使用。
#
# 與 capture-dos-oracle.sh 的差別：那一支只到主選單，這一支一路走完建角、
# 加入隊伍與 B)EGIN ADVENTURING，把每一步都存成圖，對拍時才分得出「走到哪
# 一步就不一樣了」。
#
# **`A)DD` 在空名單上不會有反應**：出貨的 `chrdat*` 不在 `CHARLIST.TXT` 裡
#（那個檔是空的），所以一定要先 C)REATE 一個角色。這一點卡了一輪才發現——
# 症狀是「按鍵沒反應」，看起來像輸入管線壞掉。
#
# **DOSBox 是輪詢鍵盤的**：`xdotool key` 的按下與放開太快會整個漏掉，
# 一定要 keydown、停一下、keyup。這是第二個看起來像「輸入管線壞掉」的坑。
#
# 鍵序由 POOL_KEYS 給（預設就是走到第一人稱畫面那一條）；`type:XXX` 是打字。
# 全部在一次性 Docker 內；輸出落在 workplace/oracle/screens（gitignore），
# 要進版控的自己挑出來複製。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXTRACTED="$ROOT/workplace/oracle/dos"
OUT="$ROOT/workplace/oracle/screens"
UID_NOW="$(id -u)"
GID_NOW="$(id -g)"
test -d "$EXTRACTED"
mkdir -p "$OUT"

docker run --rm --network none --memory 1g --cpus 1 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$UID_NOW:$GID_NOW" \
  --tmpfs /tmp/.X11-unix:rw,mode=1777 --tmpfs /run-game:rw,size=64m \
  -e HOME=/tmp/home -e "POOL_KEYS=${POOL_KEYS:-}" \
  -v "$EXTRACTED:/source:ro" -v "$OUT:/out" \
  dosbox-run:latest sh -c '
set -eu
mkdir -p "$HOME" /run-game/POOLRAD
cp -R /source/. /run-game/POOLRAD/
Xvfb :99 -screen 0 800x600x24 >/tmp/xvfb.log 2>&1 &
xvfb=$!
trap "kill $xvfb 2>/dev/null || true" EXIT
until test -S /tmp/.X11-unix/X99; do sleep 0.1; done
DISPLAY=:99 dosbox -c "mount c /run-game" -c "c:" -c "cd POOLRAD" -c "start" \
  >/tmp/dosbox.log 2>&1 &
dosbox_pid=$!
sleep 3

shot() {
  DISPLAY=:99 import -window root /tmp/shot.png
  convert /tmp/shot.png -crop 640x400+0+0 +repage "/out/$1.png"
}
settle() {
  previous=/tmp/prev.png
  current=/tmp/cur.png
  DISPLAY=:99 import -window root "$previous"
  retries=0
  while :; do
    sleep 1
    DISPLAY=:99 import -window root "$current"
    changed=$(compare -metric AE "$previous" "$current" null: 2>&1 || true)
    test "$changed" = 0 && break
    cp "$current" "$previous"
    retries=$((retries + 1))
    test "$retries" -lt 20 || break
  done
}

# 標題：等畫面出得夠花再往下按。
retries=0
while :; do
  DISPLAY=:99 import -window root /tmp/probe.png
  colors=$(identify -format "%k" /tmp/probe.png)
  test "$colors" -ge 10 && break
  sleep 1
  retries=$((retries + 1))
  test "$retries" -lt 30 || exit 1
done
settle
shot 00-title

window=$(DISPLAY=:99 xdotool search --name DOSBox | head -1)
DISPLAY=:99 xdotool windowfocus "$window"
for _ in $(seq 1 18); do
  DISPLAY=:99 xdotool key space
  sleep 0.5
done
DISPLAY=:99 xdotool key Return
sleep 2
settle
shot 01-party-menu

# 這裡開始是探索：每按一鍵就抓一張，看原版走到哪。鍵序由 $POOL_KEYS 給，
# 預設是 A)DD CHARACTER。
index=2
for key in ${POOL_KEYS:-c Return Return Return Return Return Return y type:HERO Return k e y a a e b Return Return Return}; do
  # DOSBox 是輪詢鍵盤的：xdotool key 的按下與放開太快，整個按鍵會被漏掉。
  # 分成 keydown、停一下、keyup 才進得去。
  case "$key" in
    type:*)
      # 打字：每個字元都要撐夠久，所以逐字送。
      text=${key#type:}
      while test -n "$text"; do
        letter=$(printf %.1s "$text")
        text=${text#?}
        DISPLAY=:99 xdotool keydown --window "$window" "$letter"
        sleep 0.2
        DISPLAY=:99 xdotool keyup --window "$window" "$letter"
        sleep 0.2
      done
      ;;
    *)
      DISPLAY=:99 xdotool keydown --window "$window" "$key"
      sleep 0.3
      DISPLAY=:99 xdotool keyup --window "$window" "$key"
      ;;
  esac
  # 兩張都拍：`raw` 是按下去之後**馬上**拍的，`settle` 是等畫面不動了才拍。
  # 有些畫面（例如 BEGIN 之後的第一人稱視野）只閃一下就被事件蓋掉，
  # 只拍 settle 會抓到下一幕，而檔名還寫著上一幕——那種錯看起來像「畫面
  # 不一樣」，其實是拍錯時間點。
  label="$(printf %s "$key" | tr -c "A-Za-z0-9" _)"
  sleep 0.15
  shot "$(printf "%02d-%s-raw" "$index" "$label")"
  sleep 1
  settle
  shot "$(printf "%02d-%s" "$index" "$label")"
  index=$((index + 1))
done
kill "$dosbox_pid" 2>/dev/null || true
wait "$dosbox_pid" 2>/dev/null || true
'
ls -la "$OUT"
