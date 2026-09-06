#!/usr/bin/env bash
# 以正式 Linux 發行包（AppImage）錄一段繁中實機畫面，合成敘事型推廣片。
#
# 用法：tools/build-promo.sh <已建置的版本>
#
# 素材界線（rulebook 93）：
#   * 畫面一律是**真的實機錄影**，不是 mockup 也不是重畫的假畫面。
#   * 配樂是**原版素材**：Amiga 版（1990，U.S. Gold／SSI）的遊戲內音樂，
#     作曲 Wally Beben，由 UnExoticA 的 disk rip `wb.Pool_of_Radiance`
#     （75446 bytes）以 UADE 2.13 渲染。DOS 版沒有音樂檔（只有 PC 喇叭），
#     所以配樂只能來自別的平台版本。
#   * 那份音訊是**第三方著作權**，不進 repo、不隨發行包散布，
#     由 tools/fetch-amiga-music.sh 另外備妥在 workplace/amiga-music/。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
IMAGE="game-video:latest"
RELEASE="$ROOT/dist-all/$VERSION/patch"
APPIMAGE="$RELEASE/pool-of-radiance-remake-$VERSION-x86_64.AppImage"
OUT="$ROOT/dist-all/$VERSION/promo"
FONT_DIR="${ETEN_FONT_DIR:-/home/anr2/cht/etan_font}"
CAPTION_FONT="${PROMO_CAPTION_FONT:-$ROOT/../curse_of_the_azure_bonds/assets/fonts/NotoSansTC-Regular.ttf}"
MUSIC="${PROMO_MUSIC:-$ROOT/workplace/amiga-music/wav/por-amiga-sub1.wav}"

if [[ -z "$VERSION" || ! -x "$APPIMAGE" ]]; then
  echo "用法：tools/build-promo.sh <已建置版本>（先跑 tools/package-release.sh）" >&2
  exit 2
fi
test -f "$ROOT/Pool of Radiance (1988).zip"
test -f "$FONT_DIR/stdfont.15"
test -f "$CAPTION_FONT"
test -f "$MUSIC"
docker image inspect "$IMAGE" >/dev/null

rm -rf "$OUT"; mkdir -p "$OUT"

docker run --rm --network none --memory 4g --cpus 2 --pids-limit 384 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" --tmpfs /tmp/.X11-unix:rw,mode=1777 \
  -v "$RELEASE:/release:ro" -v "$OUT:/promo" \
  -v "$ROOT/Pool of Radiance (1988).zip:/zip/pool.zip:ro" \
  -v "$FONT_DIR:/fonts:ro" \
  -v "$CAPTION_FONT:/caption.ttf:ro" \
  -v "$MUSIC:/music.wav:ro" \
  -e VERSION="$VERSION" "$IMAGE" bash -c '
set -eu
export HOME=/tmp/pool-promo-home APPIMAGE_EXTRACT_AND_RUN=1 DISPLAY=:99
mkdir -p "$HOME"
Xvfb :99 -screen 0 960x600x24 -nolisten tcp >/tmp/xvfb.log 2>&1 &
xvfb=$!
game=""; grab=""
finish() {
  test -z "$grab" || kill "$grab" 2>/dev/null || true
  test -z "$game" || kill "$game" 2>/dev/null || true
  kill "$xvfb" 2>/dev/null || true
}
trap finish EXIT
n=0; until test -S /tmp/.X11-unix/X99; do n=$((n+1)); test "$n" -lt 100 || exit 1; sleep 0.1; done

"/release/pool-of-radiance-remake-$VERSION-x86_64.AppImage" \
  -zip /zip/pool.zip -lang zh -eten-font /fonts/stdfont.15 >/tmp/game.log 2>&1 &
game=$!
window=""; n=0
until test -n "$window"; do
  window=$(xdotool search --name "Pool of Radiance Remake" 2>/dev/null | head -1 || true)
  n=$((n+1)); test "$n" -lt 300 || { tail -20 /tmp/game.log >&2; exit 1; }
  sleep 0.2
done
xdotool windowfocus "$window"
eval "$(xdotool getwindowgeometry --shell "$window")"
# 游標挪到角落。錄影本身用 -draw_mouse 0 不畫它——Xvfb 的螢幕就是視窗大小，
# 沒有「視窗外面」可以放，所以只能不畫。逐拍比對的抓圖也一樣。
xdotool mousemove 959 599
sleep 1

# 一次連續錄影，中途逐拍記下時間戳；字幕在後製用 drawtext 依時間戳套上去。
# 分段各錄一次會讓每一段都要重走一遍建角流程，慢而且接縫會跳。
: > /promo/beats.txt
mkdir -p /promo/beats
ffmpeg -hide_banner -loglevel error -y -f x11grab -framerate 30 -draw_mouse 0 \
  -video_size "${WIDTH}x${HEIGHT}" -i ":99+${X},${Y}" \
  -c:v libx264 -preset veryfast -crf 16 -pix_fmt yuv420p /promo/raw.mp4 &
grab=$!
START=$(date +%s.%N)
beat() { # beat <字幕>
  awk -v s="$START" -v t="$(date +%s.%N)" -v c="$1" \
    "BEGIN { printf \"%.2f\\t%s\\n\", t - s, c }" >> /promo/beats.txt
}
# expect 抓一張現況，跟上一拍比。一樣就代表這一拍該切的畫面沒切成——
# 那是掉鍵，往後每一句字幕都會配到錯的畫面。**寧可整支失敗也不要出片。**
shot_n=0
expect() { # expect <這一拍的名字>
  shot_n=$((shot_n + 1))
  ffmpeg -y -hide_banner -loglevel error -f x11grab -draw_mouse 0 \
    -video_size "${WIDTH}x${HEIGHT}" -i ":99+${X},${Y}" -frames:v 1 \
    "/promo/beats/$(printf %02d "$shot_n")-$1.png"
  last="/promo/beats/$(printf %02d "$((shot_n - 1))")-"*.png
  if test "$shot_n" -gt 1 && cmp -s $last "/promo/beats/$(printf %02d "$shot_n")-$1.png"; then
    echo "推廣片第 $shot_n 拍（$1）畫面沒有換：掉鍵了，不出片" >&2
    echo "逐拍的畫面留在 dist-all/*/promo/beats/ 可以直接看" >&2
    tail -20 /tmp/game.log >&2 || true
    exit 1
  fi
}
# 節奏與 tools/capture-chinese-menu.sh 同一組。**不要調快**——按快了會掉鍵，
# 而掉鍵的症狀是字幕與畫面整段漂移：影片看起來是好的，只是講的不是畫面上
# 那件事。這比拍壞更糟，所以每一拍都用 expect 對過。
pulse() { xdotool keydown "$1"; sleep 0.18; xdotool keyup "$1"; sleep 0.28; }
hold() { sleep "$1"; }

beat "SSI 金盒子《光芒之池》繁體中文重製"
expect 標題
hold 4
pulse Return
expect 人物管理
beat "人物管理：十一個指令的可見規則照原版接"
hold 4
pulse c
expect 種族
beat "建角的用詞取自軟體世界當年的官方中文說明書"
hold 3.5
pulse Return; pulse Return
expect 職業
beat "九個陣營、四種職業、六項屬性"
hold 3.5
pulse Return; pulse Return
expect 人物資料頁
beat "人物資料頁"
hold 4
pulse Return
sleep 0.5
xdotool type --delay 120 HERO
pulse Return
sleep 0.6
expect 肖像編輯器
beat "肖像用原版 HEAD／BODY 素材"
hold 3.5
pulse k
sleep 0.8
for key in p h n k e; do pulse "$key"; done
sleep 0.5
expect 戰鬥圖示編輯器
beat "戰鬥圖示用原版 READY／ACTION 素材"
hold 3.5
pulse e; sleep 0.5; pulse y; sleep 0.8; pulse a; sleep 0.8
expect 隊伍
beat "加入隊伍，開始冒險"
hold 3
pulse b
sleep 1.5
expect 導覽第一頁
beat "原版開場導覽：遊戲內 1731 句敘事全部翻完"
hold 5
for _ in $(seq 1 6); do pulse Return; done
beat "導覽走的是原版 34 步腳本移動與七頁文字"
hold 3
# 導覽剩下的部分用**可靠的節奏**按完。先前用 `xdotool key` 快轉，
# 遊戲收不到那些按鍵，而畫面在等待頁上不動——「連續幾張相同」就被騙成
# 「導覽結束了」，從這裡開始每一句字幕都配到錯的畫面。
#
# 慢慢按要二十幾秒，那在推廣片裡是一整段空白，所以**照實錄下來、在後製剪掉**：
# 這一段夾在 @CUT 與 @RESUME 之間。錄的是真的畫面，只是不播那一段。
beat "@CUT"
# 導覽剩下的部分按到**確定走完**為止。判準不能是「畫面有沒有變」——
# 導覽每一頁都在變，那個判準永遠成立；也不能是「按 A 之後變了」——
# 導覽會把 A 當成翻頁。兩個都是假陽性，而假陽性的症狀是字幕從這裡開始
# 整段配錯：影片看起來好好的，只是講的不是畫面上那件事。
#
# 用**不變量**：平面圖是開關，按兩次要回到原本那一格；導覽按兩次會前進
# 兩頁，回不去。所以「按兩次 A 回到同一張、而且中間那張不一樣」才算數。
grab() { ffmpeg -y -hide_banner -loglevel error -f x11grab -draw_mouse 0 \
  -video_size "${WIDTH}x${HEIGHT}" -i ":99+${X},${Y}" -frames:v 1 "$1"; }
opened=0
for _ in $(seq 1 40); do
  for _ in $(seq 1 10); do pulse Return; done
  grab /tmp/a0.png
  pulse a
  grab /tmp/a1.png
  pulse a
  grab /tmp/a2.png
  if cmp -s /tmp/a0.png /tmp/a2.png && ! cmp -s /tmp/a0.png /tmp/a1.png; then
    opened=1
    break
  fi
done
if test "$opened" != 1; then
  cp /tmp/a0.png /promo/beats/99-導覽卡住.png 2>/dev/null || true
  echo "導覽按不完，A 一直沒有變成開關：不出片" >&2
  echo "卡住的那一張留在 dist-all/*/promo/beats/99-導覽卡住.png" >&2
  tail -20 /tmp/game.log >&2 || true
  exit 1
fi
# 迴圈結束時平面圖是關著的（按了兩次），現在就在自由移動裡。
beat "@RESUME"
expect 自由移動
beat "第一人稱視野與原版 DOS 逐格 100% 相同"
hold 6
pulse a
expect 平面圖
beat "平面圖"
hold 4
pulse a; sleep 0.4
pulse j; sleep 0.6; pulse 4; sleep 0.3; pulse 6; sleep 0.3; pulse Return
expect 探險者手冊
beat "說明書的探險者手冊整本進了遊戲"
hold 6
pulse j; sleep 0.4; pulse k; sleep 0.6
for _ in 1 2 3; do pulse Tab; sleep 0.2; done
expect 法術一覽
beat "六十七支法術全部接完"
hold 5
pulse k; sleep 0.4; pulse F5
expect 戰術盤面
beat "戰術戰鬥：原版八方向與回合流程"
hold 5
beat "END"
sleep 0.5
kill -INT "$grab" 2>/dev/null || true
wait "$grab" 2>/dev/null || true
grab=""
cat /promo/beats.txt >&2
'

# 後製：套字幕、加原版 Amiga 配樂、淡入淡出。
docker run --rm --network none --memory 4g --cpus 2 --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -i -u "$(id -u):$(id -g)" -v "$OUT:/promo" \
  -v "$CAPTION_FONT:/caption.ttf:ro" -v "$MUSIC:/music.wav:ro" \
  "$IMAGE" python3 - <<'PY'
import json
import subprocess

raw = []
with open("/promo/beats.txt", encoding="utf-8") as handle:
    for line in handle:
        seconds, caption = line.rstrip("\n").split("\t", 1)
        raw.append((float(seconds), caption))
raw_total = raw[-1][0]

# @CUT..@RESUME 是照實錄下來但不播的那一段（導覽的按鍵快轉）。
cut_start = next((t for t, c in raw if c == "@CUT"), None)
cut_end = next((t for t, c in raw if c == "@RESUME"), None)
if (cut_start is None) != (cut_end is None):
    raise SystemExit("@CUT 與 @RESUME 要成對")
dropped = (cut_end - cut_start) if cut_start is not None else 0.0

beats = []
for seconds, caption in raw:
    if caption in ("END", "@CUT", "@RESUME"):
        continue
    beats.append((seconds - dropped if seconds >= (cut_end or 0) else seconds, caption))
total = raw_total - dropped


def escape(text):
    # drawtext 的反斜線、冒號與單引號都要跳脫，否則整條 filter 會被切斷。
    # `%` 交給 expansion=none 處理——字幕裡有「100%」，開著展開會噴
    # `Stray %` 而且那一段字幕整段不畫。
    return text.replace("\\", r"\\\\").replace(":", r"\:").replace("'", r"\'")


draws = []
for index, (start, caption) in enumerate(beats):
    end = beats[index + 1][0] if index + 1 < len(beats) else total
    draws.append(
        "drawtext=fontfile=/caption.ttf:text='%s':expansion=none"
        ":fontcolor=0xF2E4C4:fontsize=30"
        ":x=(w-text_w)/2:y=666:enable='between(t,%.2f,%.2f)'" % (escape(caption), start, end))

trim = []
if cut_start is not None:
    trim = ["select='not(between(t,%.2f,%.2f))'" % (cut_start, cut_end),
            "setpts=N/FRAME_RATE/TB"]

video = ",".join(trim + [
    "scale=1024:640:flags=lanczos",
    "pad=1280:720:128:8:0x0B0E14",
] + draws + [
    # select 丟掉 @CUT 那一段之後 setpts 會重算出一個奇怪的幀率（實測 57），
    # 這裡釘回 30——錄的時候本來就是 30。
    "fps=30",
    "fade=t=in:st=0:d=1",
    "fade=t=out:st=%.2f:d=1.2" % max(total - 1.2, 0),
])
audio = "afade=t=in:st=0:d=2,afade=t=out:st=%.2f:d=3,volume=0.55" % max(total - 3, 0)

subprocess.run([
    "ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
    "-i", "/promo/raw.mp4", "-i", "/music.wav",
    "-filter_complex", "[0:v]%s[v];[1:a]%s[a]" % (video, audio),
    "-map", "[v]", "-map", "[a]", "-t", "%.2f" % total,
    "-r", "30", "-c:v", "libx264", "-preset", "slow", "-crf", "20", "-pix_fmt", "yuv420p",
    "-c:a", "aac", "-b:a", "192k", "-movflags", "+faststart",
    "/promo/pool-of-radiance-cht-promo.mp4",
], check=True)
print(json.dumps({"seconds": round(total, 2), "captions": len(beats)}, ensure_ascii=False))
PY

ls -la "$OUT"
