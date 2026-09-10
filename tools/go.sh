#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE="${GOLDEN_BOX_REMAKE_ENGINE_DIR:-$ROOT/../golden-box-remake-engine}"
IMAGE="coab-go-test:20260729"

test -d "$ENGINE/.git"
mkdir -p "$ROOT/workplace/go-build-cache" "$ROOT/workplace/go-mod-cache"

# `go fmt` 的操作單位是 package，所以 `go fmt ./cmd/pool-game` 會把那個目錄裡
# 沒動過的檔案一起重排。2026-09-10 一次就動了 31 個檔案 494 行，其中還把中文
# 全形括號前插了空格（`//（` 變成 `// （`），破壞排版——而那些檔案跟當下的
# 工作完全無關，混進 diff 之後也很難看出哪幾行是真的改動。
#
# 「記得不要那樣做」擋不住這件事，它已經發生過好幾次。所以這裡把 fmt 換成
# 「只格式化這次改動過的檔案」，讓預設行為就是安全的。
if test "${1:-}" = fmt; then
  mapfile -t fresh < <(cd "$ROOT" && git ls-files --others --exclude-standard -- '*.go' | sort)
  mapfile -t touched < <(
    cd "$ROOT" &&
    { git diff --name-only --diff-filter=ACMR -- '*.go'
      git diff --cached --name-only --diff-filter=ACMR -- '*.go'
    } | sort -u
  )
  if test "$#" -gt 1; then
    printf '忽略參數 %s——這支不接 package。\n' "${*:2}" >&2
    printf '真的要整包重排請用 tools/go.sh fmt-all <package>，並先想清楚為什麼。\n' >&2
  fi
  if test "${#touched[@]}" -gt 0; then
    printf '這幾個既有檔案改過，但**不會**自動格式化：\n'
    printf '  %s\n' "${touched[@]}"
    printf '這個 repo 的檔案本來就不合 gofmt，對整個檔案跑 -w 會把你沒碰過的\n'
    printf '行一起重排——範圍縮到單一檔案也一樣。要看自己那幾行有沒有問題，\n'
    printf '用 gofmt -d <檔案> 讀 diff，只手改屬於自己的部分。\n'
  fi
  if test "${#fresh[@]}" -eq 0; then
    test "${#touched[@]}" -gt 0 || echo '沒有改動過的 .go 檔。'
    exit 0
  fi
  printf '格式化這 %d 個新檔（新檔沒有既有格式要保，直接排乾淨）：\n' "${#fresh[@]}"
  printf '  %s\n' "${fresh[@]}"
  set -- gofmt -w "${fresh[@]}"
elif test "${1:-}" = fmt-all; then
  if test "$#" -lt 2; then
    echo 'fmt-all 要指定 package，例如 tools/go.sh fmt-all ./cmd/pool-game' >&2
    exit 2
  fi
  printf '整包重排 %s——這會動到那個目錄裡每一個檔案，包含你沒碰過的。\n' "${*:2}" >&2
  printf '確認過就繼續；只是想整理自己改的那幾個，用 tools/go.sh fmt。\n' >&2
  set -- fmt "${@:2}"
fi

exec docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  -u "$(id -u):$(id -g)" \
  --tmpfs /tmp/.X11-unix:rw,mode=1777 \
  -v "$ROOT:/src" -v "$ENGINE:/engine:ro" \
  -v "$ROOT/workplace/go-build-cache:/gocache" \
  -v "$ROOT/workplace/go-mod-cache:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod \
  -e 'GOFLAGS=-buildvcs=false -modfile=/src/workplace/pool-local.mod' \
  -w /src "$IMAGE" bash -c \
  'cp go.mod workplace/pool-local.mod; cp go.sum workplace/pool-local.sum
   printf "\nreplace github.com/wicanr2/golden-box-remake-engine => /engine\n" >> workplace/pool-local.mod
   if test "${1:-}" = gofmt; then
     exec /usr/local/go/bin/gofmt "${@:2}"
   fi
   if test "${1:-}" = test; then
     Xvfb :99 -screen 0 1024x768x24 >/tmp/xvfb.log 2>&1 & xvfb=$!
     trap "kill $xvfb 2>/dev/null || true" EXIT
     until test -S /tmp/.X11-unix/X99; do sleep .1; done
     export DISPLAY=:99
     # go 的預設 timeout 是 10 分鐘，而 cmd/pool-game 的探索測試光一條
     # （TestTheCastleBehindStojanowGateHasContent）就要七八分鐘，整個套件
     # 長期在 380..600 秒之間浮動——偶爾就整批被砍，症狀是
     # `panic: test timed out`，看起來像當掉而不是慢。這裡給一個誠實的上限；
     # 呼叫端自己傳的 -timeout 排在後面，會蓋掉這一個。
     set -- test -timeout 25m "${@:2}"
   fi
   /usr/local/go/bin/go "$@"' bash "$@"
