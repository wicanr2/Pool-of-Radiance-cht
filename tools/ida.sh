#!/usr/bin/env bash
# IDA Pro 9.4 headless 包裝（本專案唯一入口）。
#
# 固定用 ida-pro-9.4-idapython:locked-v1：其他基底 image 跑 IDAPython 會
# 「零輸出、零訊息」地失敗，而且 exit code 不可信。不要覆寫容器裡的 $HOME——
# idapyswitch 把設定寫在 image 的 $HOME/.idapro，蓋掉它會讓 IDAPython 靜默失效。
#
# 用法：
#   tools/ida.sh binary16 <raw.bin>          以 16-bit 8086、base 0 建 .i64
#   tools/ida.sh py <i64|bin> <script.py>    跑 tools/ 裡的 IDAPython 腳本
#
# 硬規則：headless 的 print 不進 stdout，exit code 也不可信。腳本一律把結果
# 寫檔，收工前驗檔案存在且非空。
set -euo pipefail

IMAGE="${POOL_IDA_IMAGE:-ida-pro-9.4-idapython:locked-v1}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

die() { echo "ida.sh: $*" >&2; exit 2; }

run_in() {
  local work="$1"; shift
  [ -d "$work" ] || die "工作目錄不存在：$work"
  docker run --rm --network none --memory 4g --cpus 2 --pids-limit 512 \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" \
    -v "$work:/work" -v "$ROOT/tools:/work-tools:ro" \
    "${env_args[@]}" \
    -w /work "$IMAGE" idat "$@"
}

# 腳本用的環境變數一律以 POOL_IDA_ 開頭，這裡整批轉給容器；
# 不要傳 HOME——image 的 $HOME/.idapro 是 idapyswitch 寫的，蓋掉會讓 IDAPython 靜默失效。
env_args=()
while IFS= read -r name; do
  case "$name" in
    POOL_IDA_IMAGE) ;;
    POOL_IDA_*) env_args+=(-e "$name=${!name}") ;;
  esac
done < <(compgen -v | grep '^POOL_IDA_' || true)

cmd="${1:-}"; shift || true
case "$cmd" in
  binary16)
    target="${1:?用法: tools/ida.sh binary16 <raw.bin>}"; shift
    [ -f "$target" ] || die "找不到檔案：$target"
    run_in "$(cd "$(dirname "$target")" && pwd)" -A -B -p8086 -b0 "$@" "$(basename "$target")"
    ;;
  py)
    target="${1:?用法: tools/ida.sh py <i64|bin> <script.py> [args]}"; shift
    script="${1:?缺少 script}"; shift
    [ -f "$target" ] || die "找不到檔案：$target"
    [ -f "$ROOT/tools/$(basename "$script")" ] || die "script 必須放在 tools/：$script"
    run_in "$(cd "$(dirname "$target")" && pwd)" -A "-S/work-tools/$(basename "$script") $*" "$(basename "$target")"
    ;;
  *)
    sed -n '3,15p' "${BASH_SOURCE[0]}"
    exit 2
    ;;
esac
