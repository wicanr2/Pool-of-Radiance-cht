#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE="${GOLDEN_BOX_REMAKE_ENGINE_DIR:-$ROOT/../golden-box-remake-engine}"
IMAGE="coab-go-test:20260729"

test -d "$ENGINE/.git"
mkdir -p "$ROOT/workplace/go-build-cache" "$ROOT/workplace/go-mod-cache"

exec docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  -u "$(id -u):$(id -g)" \
  -v "$ROOT:/src" -v "$ENGINE:/engine:ro" \
  -v "$ROOT/workplace/go-build-cache:/gocache" \
  -v "$ROOT/workplace/go-mod-cache:/gomod" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod \
  -e 'GOFLAGS=-buildvcs=false -modfile=/src/workplace/pool-local.mod' \
  -w /src "$IMAGE" bash -c \
  'cp go.mod workplace/pool-local.mod; printf "\nreplace github.com/wicanr2/golden-box-remake-engine => /engine\n" >> workplace/pool-local.mod; /usr/local/go/bin/go "$@"' bash "$@"
