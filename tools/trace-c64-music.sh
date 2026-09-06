#!/usr/bin/env bash
# 在 VICE 裡跑 C64 版的 Pool of Radiance，錄下 SID 輸出，量「什麼時候有音樂」。
#
# 用法：tools/trace-c64-music.sh [秒數]
#
# 為什麼要實跑：靜態讀位址在這一份上不可靠——DUNGEON／COMBAT／INIT／POST.COM／
# CAMP 的 PRG 表頭全寫 $1000（它們互相覆蓋），而 BOOT 用自訂位址載入，
# 所以「檔案裡的位址」不等於「執行時的位址」。實測也證實：`break $BA00` 從來
# 沒觸發，但 SID 是從 $BA18／$BA8C.. 被寫的（IRQ 直接進 $BA10）。
#
# **輸出要配正對照看。** 第一次量的時候 watch 完全沒有輸出，看起來像「原版不放
# 音樂」；換成一定會被寫的 $D020 邊框暫存器再量一次也沒有輸出——壞的是擷取
# （-console 模式下 monitor 的輸出不進 stdout），不是遊戲。所以這支一律開
# `logname` + `log on`，而且保留 -exitscreenshot：**沒有畫面對照的 RMS 讀不出意思**。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SECONDS_WANTED="${1:-400}"
DISKS="$ROOT/c64-disks"
OUT="$ROOT/workplace/vice"
IMAGE="${POOL_VICE_IMAGE:-pool-vice:2}"
DISK="$DISKS/Pool_of_Radiance_v1.1_1988_SSI_Disk_1_of_4_Side_A.d64"

test -f "$DISK" || { echo "找不到 C64 磁碟：$DISK" >&2; exit 2; }
docker image inspect "$IMAGE" >/dev/null
mkdir -p "$OUT"

# PAL C64 是 985248 Hz。
cycles=$(( SECONDS_WANTED * 985248 ))

cat > "$OUT/cmds.txt" <<'EOF'
logname "/work/trace.log"
log on
break $FFE4
command 1 "> $00c6 01; > $0277 4e; del 1; g"
EOF

rm -f "$OUT/c64.wav" "$OUT/trace.log" "$OUT/c64-end.png"
docker run --rm --network none --memory 2g --cpus 2 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" -v "$DISKS:/disks:ro" -v "$OUT:/work" -e HOME=/tmp/h \
  "$IMAGE" sh -c "
mkdir -p /tmp/h
export SDL_VIDEODRIVER=dummy
timeout $(( SECONDS_WANTED + 120 )) x64sc -console -sounddev wav -soundarg /work/c64.wav \
  -8 '/disks/$(basename "$DISK")' -autostart '/disks/$(basename "$DISK")' \
  -moncommands /work/cmds.txt -limitcycles $cycles \
  -exitscreenshot /work/c64-end.png >/dev/null 2>&1
"

python3 - "$OUT/c64.wav" <<'PY'
import array, math, struct, sys

raw = open(sys.argv[1], 'rb').read()
offset, channels, rate, data = 12, 1, 1, b''
while offset + 8 <= len(raw):
    chunk_id = raw[offset:offset + 4]
    size = struct.unpack('<I', raw[offset + 4:offset + 8])[0]
    if chunk_id == b'fmt ':
        _, channels, rate = struct.unpack('<HHI', raw[offset + 8:offset + 16])
    if chunk_id == b'data':
        data = raw[offset + 8:offset + 8 + size]
        break
    offset += 8 + size + (size & 1)

samples = array.array('h')
samples.frombytes(data[:len(data) // 2 * 2])
print('錄到 %.1f 秒（%d 聲道 %d Hz）' % (len(samples) / channels / rate, channels, rate))
window = 20 * rate * channels
for start in range(0, len(samples), window):
    piece = samples[start:start + window]
    if len(piece) < window // 2:
        break
    rms = math.sqrt(sum(float(v) * v for v in piece) / len(piece))
    print('  %4.0f–%4.0f 秒  RMS %8.1f  (%6.1f dBFS)' % (
        start / channels / rate, (start + window) / channels / rate, rms,
        20 * math.log10(rms / 32768) if rms else -99))
print()
print('畫面對照：%s' % sys.argv[1].replace('c64.wav', 'c64-end.png'))
PY
