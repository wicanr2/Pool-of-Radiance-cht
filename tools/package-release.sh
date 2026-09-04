#!/usr/bin/env bash
# 產生 Linux AppImage、Windows ZIP 與 macOS 雙架構 ZIP。
# 用法：tools/package-release.sh <版本>
#
# 發行包不含原版遊戲資料與倚天字型：兩者都沒有公開散布權（見 NOTICE.md），
# 玩家要自己準備。`packaging/README-發行包.md` 說明放哪裡。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE="${GOLDEN_BOX_REMAKE_ENGINE_DIR:-$ROOT/../golden-box-remake-engine}"
VERSION="${1:-}"
if [[ -z "$VERSION" || ! "$VERSION" =~ ^[0-9A-Za-z._-]+$ ]]; then
  echo "用法：tools/package-release.sh <版本（僅英數、點、底線、連字號）>" >&2
  exit 2
fi

GO_IMAGE="coab-go-ebiten:1.24"
MAC_IMAGE="u2cht-osxcross:20260826-r1"
APPIMAGE_IMAGE="u5cht/appimage:latest"
OUT="dist-all/$VERSION"
UID_NOW="$(id -u)"
GID_NOW="$(id -g)"

for image in "$GO_IMAGE" "$MAC_IMAGE" "$APPIMAGE_IMAGE"; do
  docker image inspect "$image" >/dev/null
done
test -d "$ENGINE/.git"
mkdir -p "$ROOT/workplace/go-build-cache" "$ROOT/workplace/go-mod-cache"

# 圖示是原創的，不用原版標題畫面——那是 SSI 的美術。
python3 "$ROOT/tools/build-release-icon.py" "$ROOT/packaging/linux/pool-of-radiance-remake.png"

go_mounts=(
  -v "$ROOT:/src" -v "$ENGINE:/engine:ro"
  -v "$ROOT/workplace/go-build-cache:/gocache"
  -v "$ROOT/workplace/go-mod-cache:/gomod"
  -e GOCACHE=/gocache -e GOMODCACHE=/gomod
  -w /src
)
# 共用 engine 以 replace 指進唯讀掛載，與 tools/go.sh 同一套；
# modfile 放在 /tmp，不動 repo 裡的 go.mod。
prepare_modfile='cp go.mod /tmp/pool.mod; cp go.sum /tmp/pool.sum
  printf "\nreplace github.com/wicanr2/golden-box-remake-engine => /engine\n" >> /tmp/pool.mod'

build() { # build <image> <GOOS> <GOARCH> <CGO> <輸出檔名> [額外的 -e...]
  local image="$1" goos="$2" goarch="$3" cgo="$4" output="$5"; shift 5
  docker run --rm --network none --memory 3g --cpus 2 --pids-limit 512 \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$UID_NOW:$GID_NOW" "${go_mounts[@]}" \
    -e GOOS="$goos" -e GOARCH="$goarch" -e CGO_ENABLED="$cgo" "$@" \
    "$image" bash -c "set -eu; $prepare_modfile
      /usr/local/go/bin/go build -buildvcs=false -modfile=/tmp/pool.mod \
        -trimpath -ldflags='-s -w' -o '$OUT/build/$output' ./cmd/pool-game"
}

run_helper() { # run_helper <script>
  docker run --rm --network none --memory 1g --cpus 1 --pids-limit 128 \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$UID_NOW:$GID_NOW" -v "$ROOT:/src" -w /src "$GO_IMAGE" bash -c "$1"
}

run_helper "set -eu; rm -rf '$OUT'; mkdir -p '$OUT/build' '$OUT/patch'"

build "$GO_IMAGE"  linux   amd64 1 pool-game-linux-amd64
build "$GO_IMAGE"  windows amd64 0 pool-game.exe
build "$MAC_IMAGE" darwin  amd64 1 pool-game-darwin-amd64 -e CC=o64-clang  -e CXX=o64-clang++
build "$MAC_IMAGE" darwin  arm64 1 pool-game-darwin-arm64 -e CC=oa64-clang -e CXX=oa64-clang++

run_helper "set -eu
  V='$VERSION'
  B='$OUT/build'
  BASE='$OUT/patch'
  mkdir -p \"\$BASE/linux/AppDir/usr/bin\" \"\$BASE/linux/AppDir/usr/lib\" \\
    \"\$BASE/linux/AppDir/usr/share/doc\" \"\$BASE/windows\" \\
    \"\$BASE/macos-amd64/Pool of Radiance Remake.app/Contents/MacOS\" \\
    \"\$BASE/macos-arm64/Pool of Radiance Remake.app/Contents/MacOS\"

  # 條款要跟著每一個發行包走：拿到 ZIP 的人看不到儲存庫。
  for target in \\
    \"\$BASE/linux/AppDir\" \\
    \"\$BASE/windows\" \\
    \"\$BASE/macos-amd64/Pool of Radiance Remake.app/Contents/MacOS\" \\
    \"\$BASE/macos-arm64/Pool of Radiance Remake.app/Contents/MacOS\"; do
    cp LICENSE NOTICE.md \"\$target/\"
    cp packaging/README-發行包.md \"\$target/README.md\"
  done
  cp LICENSE NOTICE.md \"\$BASE/linux/AppDir/usr/share/doc/\"
  # 發行根目錄自己也要一份：從 dist-all/ 直接取檔的人看不到儲存庫。
  cp LICENSE NOTICE.md '$OUT/'

  cp \"\$B/pool-game-linux-amd64\" \"\$BASE/linux/AppDir/usr/bin/pool-game\"
  cp packaging/linux/AppRun \"\$BASE/linux/AppDir/AppRun\"
  cp packaging/linux/pool-of-radiance-remake.desktop \"\$BASE/linux/AppDir/\"
  cp packaging/linux/pool-of-radiance-remake.png \"\$BASE/linux/AppDir/\"
  chmod 0755 \"\$BASE/linux/AppDir/AppRun\" \"\$BASE/linux/AppDir/usr/bin/pool-game\"

  cp \"\$B/pool-game.exe\" \"\$BASE/windows/pool-game.exe\"
  cp packaging/windows/啟動遊戲.bat \"\$BASE/windows/\"

  for arch in amd64 arm64; do
    APP=\"\$BASE/macos-\$arch/Pool of Radiance Remake.app/Contents\"
    cp packaging/macos/launcher \"\$APP/MacOS/pool-game\"
    cp \"\$B/pool-game-darwin-\$arch\" \"\$APP/MacOS/pool-game-bin\"
    sed \"s/@VERSION@/\$V/g\" packaging/macos/Info.plist > \"\$APP/Info.plist\"
    chmod 0755 \"\$APP/MacOS/pool-game\" \"\$APP/MacOS/pool-game-bin\"
  done

  # AppDir 不夾帶任何 .so。這個執行檔只連 X11 那四顆（libX11／libxcb／libXau／
  # libXdmcp）加 glibc，全部屬於系統圖形堆疊，本來就該由主機提供。
  # 從建置 image 複製過去反而會壞：那幾顆是對 glibc 2.38 連結的，搬到 glibc
  # 較舊的機器上會噴 `version GLIBC_2.38 not found`——而且是在建置階段完全
  # 正常、到玩家手上才失敗。tools/linux-release-smoke.sh 就是量這件事的。"

docker run --rm --network none --memory 1g --cpus 1 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$UID_NOW:$GID_NOW" -v "$ROOT:/src" -w /src "$APPIMAGE_IMAGE" bash -c \
  "ARCH=x86_64 appimagetool '$OUT/patch/linux/AppDir' '$OUT/patch/pool-of-radiance-remake-$VERSION-x86_64.AppImage'"

docker run --rm --network none --memory 512m --cpus 1 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$UID_NOW:$GID_NOW" -v "$ROOT:/src" -w /src python:3.12-slim python -c '
import pathlib, sys, zipfile
root, version = pathlib.Path(sys.argv[1]), sys.argv[2]
for source, name in [
    (root / "windows", f"pool-of-radiance-remake-{version}-windows-x86_64.zip"),
    (root / "macos-amd64", f"pool-of-radiance-remake-{version}-macos-x86_64.zip"),
    (root / "macos-arm64", f"pool-of-radiance-remake-{version}-macos-arm64.zip"),
]:
    with zipfile.ZipFile(root / name, "w", zipfile.ZIP_DEFLATED) as archive:
        for path in sorted(source.rglob("*")):
            if path.is_file():
                info = zipfile.ZipInfo(str(path.relative_to(source)))
                info.external_attr = (path.stat().st_mode & 0xFFFF) << 16
                info.compress_type = zipfile.ZIP_DEFLATED
                archive.writestr(info, path.read_bytes())
' "$OUT/patch" "$VERSION"

docker run --rm --network none --memory 512m --cpus 1 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$UID_NOW:$GID_NOW" -v "$ROOT:/src" -w /src python:3.12-slim python -c '
import hashlib, json, pathlib, sys
root, version = pathlib.Path(sys.argv[1]), sys.argv[2]
artifacts = sorted(list(root.glob("*.AppImage")) + list(root.glob("*.zip")))
if not artifacts:
    raise SystemExit("no release artifacts were produced")
rows = []
for path in artifacts:
    rows.append({
        "name": path.name,
        "bytes": path.stat().st_size,
        "sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
    })
manifest = {"schema": "pool-release/1", "version": version, "artifacts": rows}
(root.parent / "manifest.json").write_text(
    json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
for row in rows:
    print(row["sha256"], row["name"], row["bytes"], "bytes")
' "$OUT/patch" "$VERSION"

echo "發行包在 $ROOT/$OUT/patch"
