#!/usr/bin/env bash
# 報出遊戲要顯示、但倚天字型畫不出來的字。
#
# 字型是第三方資產、不進 repo，所以這支腳本跟 capture-chinese-menu.sh 一樣，
# 以唯讀掛載把主機的字型帶進容器；路徑可由 ETEN_FONT_DIR 覆寫。
# 有缺字時以非零狀態結束，方便當成閘門用。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE="$(cd "$ROOT/../golden-box-remake-engine" && pwd)"
FONT_DIR="${ETEN_FONT_DIR:-/home/anr2/cht/etan_font}"
test -f "$FONT_DIR/stdfont.15"
mkdir -p "$ROOT/workplace/go-build-cache" "$ROOT/workplace/go-mod-cache"

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  -v "$ROOT:/src" -v "$ENGINE:/engine:ro" -v "$FONT_DIR:/fonts:ro" \
  -v "$ROOT/workplace/go-build-cache:/gocache" \
  -v "$ROOT/workplace/go-mod-cache:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod -w /src \
  coab-go-test:20260729 bash -c '
set -eu
cp go.mod /tmp/pool.mod; cp go.sum /tmp/pool.sum
printf "\nreplace github.com/wicanr2/golden-box-remake-engine => /engine\n" >> /tmp/pool.mod
exec /usr/local/go/bin/go run -buildvcs=false -modfile=/tmp/pool.mod ./cmd/pool-font-coverage \
  -eten-font /fonts/stdfont.15 \
  -eten-ascii-font /fonts/ascfont.15'
