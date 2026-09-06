#!/usr/bin/env bash
# 用 dosgolem 跑原版 DOS 執行檔，把對拍要用的基準畫面產到
# `workplace/dosgolem-ref/`（gitignore）。
#
# **不用 DOSBox。** DOSBox 只能從外面看畫面：送鍵靠 xdotool、等畫面靠 sleep、
# 判斷靠像素，而「猜對」與「猜錯」在截圖上長得一樣。dosgolem 是可程式化的
# DOS 執行器，走位靠「程式讀走了幾個鍵」與「畫面連續多少道指令沒動」，
# 而且直接吐 320x200 的色號陣列——對拍本來就該在色號空間做，不是在 PNG 上。
#
# 需要：
#   - dosgolem 的工作區（DOSGOLEM_DIR，預設 ../../dosgolem-pool），
#     且在 `pool-of-radiance-oracle` 分支上（`cmd/shots`、spec 008／009）。
#   - 原版 DOS 檔案解壓在 workplace/oracle/dos（本 repo 不含原版素材）。
#
# 輸出 `.idx`（一格一個位元組的色號，320x200）與 `.png`（EGA 調色盤，給人看）。
# **對拍的依據是 `.idx`**：PNG 過了調色盤，而調色盤是另一個變數。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOSGOLEM="${DOSGOLEM_DIR:-$(cd "$ROOT/../../dosgolem-pool" 2>/dev/null && pwd || true)}"
SOURCE="$ROOT/workplace/oracle/dos"
OUT="$ROOT/workplace/dosgolem-ref"

test -n "$DOSGOLEM" || { echo "找不到 dosgolem 工作區；設 DOSGOLEM_DIR" >&2; exit 2; }
test -f "$DOSGOLEM/cmd/shots/main.go" || {
  echo "$DOSGOLEM 沒有 cmd/shots——需要 pool-of-radiance-oracle 分支" >&2; exit 2; }
test -f "$SOURCE/start.exe" || {
  echo "原版檔案要先解壓到 $SOURCE（本 repo 不含原版素材）" >&2; exit 2; }

# 鍵序：開場動畫 → 防拷提示頁（按 Return 送空字串就過）→ 人物管理選擇項 →
# 建角一路到人物資料頁 → 存進名單 → 加進隊伍 → 開始冒險 → 導覽。
#
# 名字**一個字母一個 script 項**：整串一次推進佇列時，原版的輸入欄只收得到
# 最後一個字（量到的結果是 `O.cha` 不是 `HERO.CHA`）。原因還沒解，
# 所以照會動的方式送，不要假設它跟一次推一串等價。
KEYS="${POOL_DOSGOLEM_KEYS:-rep:9:Space,Return,Return,c,Return,Return,Return,Return,Return,Return,y,H,E,R,O,Return,k,e,y,a,a,e,b,rep:14:Return}"

rm -rf "$OUT"
mkdir -p "$OUT" "$ROOT/workplace/dosgolem-scratch"
rm -rf "$ROOT/workplace/dosgolem-scratch"
mkdir -p "$ROOT/workplace/dosgolem-scratch"

# 原版目錄唯讀掛載；程式的存檔落在另外一個可寫的暫存層（dosgolem spec 009）。
docker run --rm --network none --memory 4g --cpus "${PARITY_CPUS:-2}" --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  -v "$DOSGOLEM:/dosgolem" \
  -v "$SOURCE:/orig:ro" \
  -v "$OUT:/out" \
  -v "$ROOT/workplace/dosgolem-scratch:/scratch" \
  -v "$DOSGOLEM/workplace/gocache:/gocache" \
  -v "$DOSGOLEM/workplace/gomodcache:/gomodcache" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomodcache -e HOME=/tmp -e GOFLAGS=-mod=mod \
  -w /dosgolem golang:1.24-bookworm \
  go run ./cmd/shots -exe /orig/start.exe -root /orig -scratch /scratch \
    -out /out -budget 60000000 -idle 3000000 -keys "$KEYS"

echo "基準畫面 → $OUT"
