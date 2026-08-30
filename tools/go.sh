#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE="${GOLDEN_BOX_REMAKE_ENGINE_DIR:-$ROOT/../golden-box-remake-engine}"
IMAGE="coab-go-test:20260729"

test -d "$ENGINE/.git"
mkdir -p "$ROOT/workplace/go-build-cache" "$ROOT/workplace/go-mod-cache"

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
   if test "${1:-}" = test; then
     Xvfb :99 -screen 0 1024x768x24 >/tmp/xvfb.log 2>&1 & xvfb=$!
     trap "kill $xvfb 2>/dev/null || true" EXIT
     until test -S /tmp/.X11-unix/X99; do sleep .1; done
     export DISPLAY=:99
   fi
   /usr/local/go/bin/go "$@"' bash "$@"
