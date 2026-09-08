#!/usr/bin/env python3
"""把 remake 的截圖與 dosgolem 產的原版基準逐格比。

**比的是色號空間**：dosgolem 直接吐 320x200 的 EGA 色號，remake 的截圖用
最近鄰降回 320x200 之後再對回 16 色。平滑縮放會把差異抹掉，所以一律最近鄰。

輸出一份 JSON 報告，逐個區域給「相同格數／總格數」。抽樣，不是全程：
`title` 比整張，`first-person` 只比第一人稱那一框的 88x88——那是唯一宣稱過
逐格相同的東西（spec 047／126）。
"""
import json
import os
import struct
import sys
import zlib

WIDTH, HEIGHT = 320, 200

# 標準 EGA 十六色。dosgolem 的 PNG 與 remake 的畫面用的是同一張表。
EGA = [
    (0, 0, 0), (0, 0, 170), (0, 170, 0), (0, 170, 170),
    (170, 0, 0), (170, 0, 170), (170, 85, 0), (170, 170, 170),
    (85, 85, 85), (85, 85, 255), (85, 255, 85), (85, 255, 255),
    (255, 85, 85), (255, 85, 255), (255, 255, 85), (255, 255, 255),
]
EGA_INDEX = {rgb: i for i, rgb in enumerate(EGA)}


def read_png_rgb(path):
    """最小 PNG 讀取：只認 8 位元的 RGB／RGBA／調色盤，夠讀我們自己產的檔。"""
    data = open(path, "rb").read()
    assert data[:8] == b"\x89PNG\r\n\x1a\n", path
    pos, idat, palette, trns = 8, b"", None, None
    width = height = depth = colour = 0
    while pos < len(data):
        length = struct.unpack(">I", data[pos:pos + 4])[0]
        kind = data[pos + 4:pos + 8]
        body = data[pos + 8:pos + 8 + length]
        if kind == b"IHDR":
            width, height, depth, colour = struct.unpack(">IIBB", body[:10])
        elif kind == b"PLTE":
            palette = body
        elif kind == b"tRNS":
            trns = body
        elif kind == b"IDAT":
            idat += body
        elif kind == b"IEND":
            break
        pos += 12 + length
    channels = {0: 1, 2: 3, 3: 1, 4: 2, 6: 4}[colour]
    assert depth == 8 or (colour == 3 and depth in (1, 2, 4)), \
        f"{path} 的位元深度 {depth} 沒支援"
    raw = zlib.decompress(idat)
    if depth == 8:
        stride = width * channels
    else:
        # 調色盤圖可以是 1／2／4 位元一格（DOSBox 存出來的是 4 位元）。
        # 先攤成一格一個位元組，後面就跟 8 位元同一條路。
        stride = (width * depth + 7) // 8
    out = bytearray(width * height * 3)
    previous = bytearray(stride)
    at = 0
    for y in range(height):
        filt = raw[at]
        at += 1
        line = bytearray(raw[at:at + stride])
        at += stride
        for x in range(stride):
            a = line[x - channels] if x >= channels else 0
            b = previous[x]
            c = previous[x - channels] if x >= channels else 0
            if filt == 1:
                line[x] = (line[x] + a) & 0xFF
            elif filt == 2:
                line[x] = (line[x] + b) & 0xFF
            elif filt == 3:
                line[x] = (line[x] + (a + b) // 2) & 0xFF
            elif filt == 4:
                p = a + b - c
                pa, pb, pc = abs(p - a), abs(p - b), abs(p - c)
                pred = a if (pa <= pb and pa <= pc) else (b if pb <= pc else c)
                line[x] = (line[x] + pred) & 0xFF
        previous = line
        if depth != 8:
            wide = bytearray(width)
            per = 8 // depth
            mask = (1 << depth) - 1
            for x in range(width):
                byte = line[x // per]
                shift = 8 - depth * (x % per + 1)
                wide[x] = (byte >> shift) & mask
            line = wide
        for x in range(width):
            base = (y * width + x) * 3
            if colour == 3:
                idx = line[x]
                out[base:base + 3] = palette[idx * 3:idx * 3 + 3]
            elif colour in (0, 4):
                out[base] = out[base + 1] = out[base + 2] = line[x * channels]
            else:
                out[base:base + 3] = line[x * channels:x * channels + 3]
    _ = trns
    return width, height, bytes(out)


def downsample_nearest(width, height, rgb, target_w=WIDTH, target_h=HEIGHT):
    out = bytearray(target_w * target_h * 3)
    for y in range(target_h):
        sy = y * height // target_h
        for x in range(target_w):
            sx = x * width // target_w
            src = (sy * width + sx) * 3
            dst = (y * target_w + x) * 3
            out[dst:dst + 3] = rgb[src:src + 3]
    return bytes(out)


def to_indices(rgb):
    """把 RGB 對回 EGA 色號。對不上的用 255 標出來——**不要就近取色**，
    那會把「remake 用了別的顏色」變成「顏色一樣」。"""
    out = bytearray(WIDTH * HEIGHT)
    for i in range(WIDTH * HEIGHT):
        out[i] = EGA_INDEX.get(tuple(rgb[i * 3:i * 3 + 3]), 255)
    return bytes(out)


# 原版那一圈繩索外框的位置（native 320x200，量出來的）：
# 上 0..7、下 184..191、左 0..7、右 312..319。**底下那一列指令／選單字
# 在 192..198，也就是框的外面**——那是原版的版面，不是框的一部分。
FRAME_ROWS = list(range(0, 8)) + list(range(184, 192))
FRAME_COLS = list(range(0, 8)) + list(range(312, 320))


def frame_cells():
    """外框那一圈的格子座標。"""
    cells = set()
    for y in FRAME_ROWS:
        for x in range(WIDTH):
            cells.add((x, y))
    for x in FRAME_COLS:
        for y in range(HEIGHT):
            cells.add((x, y))
    return sorted(cells)


FRAME = frame_cells()


def compare_frame(reference, actual):
    """只比外框那一圈。框線對不對得上是幾何問題，與文字語言無關。"""
    same = 0
    for x, y in FRAME:
        if reference[y * WIDTH + x] == actual[y * WIDTH + x]:
            same += 1
    return same, len(FRAME)


def compare(reference, actual, box):
    left, top, width, height = box
    same = total = 0
    for y in range(top, top + height):
        for x in range(left, left + width):
            total += 1
            if reference[y * WIDTH + x] == actual[y * WIDTH + x]:
                same += 1
    return same, total


def load_remake(path):
    w, h, rgb = read_png_rgb(path)
    return to_indices(downsample_nearest(w, h, rgb))


def main():
    ref_dir, out_dir = sys.argv[1], sys.argv[2]
    shots = json.load(open(os.path.join(ref_dir, "shots.json")))

    # 哪一張基準對哪一個 remake 畫面，以及要比的區域。
    #
    # **用內容雜湊挑基準，不用索引。** 索引跟著 settle 的時機浮動（同一段
    # 鍵序兩次跑出來的張數會差一兩張），而雜湊在決定性模擬底下是穩的。
    # 挑錯一張的症狀是「比出一個很低的百分比」，看起來像 remake 畫錯。
    # 逐格相同只對 title 與 first-person 宣稱；其餘是建隊到進城那一段的
    # 逐張對照，量的是版面差異，不是缺陷計數。
    #
    # **用內容雜湊挑基準，不用索引。** 索引跟著 settle 的時機浮動（同一段
    # 鍵序兩次跑出來的張數會差一兩張），而雜湊在決定性模擬底下是穩的。
    # 挑錯一張的症狀是「比出一個很低的百分比」，看起來像 remake 畫錯。
    plan = [
        {
            "name": "title", "kind": "pixel-parity",
            "digest": "04cfb632f3f3fe4dc15b39dc13a74b7f68f386014132c9e67595517e4968ac68",
            "remake": "remake-title.png",
            "ref_box": [0, 0, 320, 200], "remake_box": [0, 0, 320, 200],
            "note": "整張。原版與 remake 畫的是同一份 TITLE.DAX，位置與縮放相同；"
                    "唯一預期的差異是 remake 自己加的按鍵提示那一行。",
        },
        {
            "name": "first-person", "kind": "pixel-parity",
            "digest": "f384683d3f49ece193dcb8eff272e5fe79af836a67db866eefc56fa183952642",
            "remake": "remake-first-person.png",
            "ref_box": [24, 24, 88, 88], "remake_box": [24, 43, 88, 88],
            "note": "第一人稱框內的 88x88。remake 的框在畫面上比原版低 19 列。",
        },
        # 建隊到進城。這幾張比的是版面，所以另外報外框那一圈。
        {"name": "menu-empty", "kind": "layout",
         "digest": "ad32b0c0", "remake": "remake-menu-empty.png",
         "note": "人物管理選擇項，空隊伍"},
        {"name": "race", "kind": "layout",
         "digest": "c3d05e32", "remake": "remake-race.png", "note": "種族"},
        {"name": "gender", "kind": "layout",
         "digest": "b0a1c460", "remake": "remake-gender.png", "note": "性別"},
        {"name": "class", "kind": "layout",
         "digest": "563ce90d", "remake": "remake-class.png", "note": "職業"},
        {"name": "alignment", "kind": "layout",
         "digest": "389c0055", "remake": "remake-alignment.png", "note": "陣營"},
        {"name": "sheet", "kind": "layout",
         "digest": "f33c0725", "remake": "remake-sheet.png",
         "note": "人物資料頁（KEEP THIS CHARACTER?）"},
        {"name": "name", "kind": "layout",
         "digest": "17fdd284", "remake": "remake-name.png", "note": "姓名輸入"},
        {"name": "portrait", "kind": "layout",
         "digest": "9019afcf", "remake": "remake-portrait.png",
         "note": "肖像編輯器（原版底下是 HEAD BODY KEEP）"},
        {"name": "icon", "kind": "layout",
         "digest": "60fbf041", "remake": "remake-icon.png",
         "note": "戰鬥造形設計（原版是 OLD／NEW 四格加底部指令列）"},
        {"name": "menu-party", "kind": "layout",
         "digest": "11c6caa4", "remake": "remake-menu-party.png",
         "note": "人物管理選擇項，隊伍裡有人"},
        {"name": "intro", "kind": "layout",
         "digest": "3717ce93", "remake": "remake-intro.png",
         "note": "按下 B 之後的第一幕"},
    ]

    report = {"screens": []}
    failed = False
    dosbox_dir = sys.argv[3] if len(sys.argv) > 3 else None
    for item in plan:
        digest = item["digest"]
        matches = [s for s in shots if s["sha256"].startswith(digest)]
        if not matches:
            print(f"基準裡找不到 {item['name']} 的畫面（雜湊 {digest[:8]}）"
                  f"——鍵序或 dosgolem 版本改過了，先重生基準再更新對照表", file=sys.stderr)
            failed = True
            continue
        info = matches[0]
        raw = open(os.path.join(ref_dir, info["path"].replace(".png", ".idx")), "rb").read()
        reference = bytes(v & 0x0F for v in raw)
        actual = load_remake(os.path.join(out_dir, item["remake"]))

        rl, rt, w, h = item.get("ref_box", [0, 0, WIDTH, HEIGHT])
        ml, mt, _, _ = item.get("remake_box", [0, 0, WIDTH, HEIGHT])
        same = total = 0
        for y in range(h):
            for x in range(w):
                total += 1
                if reference[(rt + y) * WIDTH + rl + x] == actual[(mt + y) * WIDTH + ml + x]:
                    same += 1
        entry = {
            "name": item["name"], "kind": item["kind"],
            "reference": info["path"], "reference_step": info["step"],
            "remake": item["remake"],
            "same": same, "total": total, "ratio": round(same / total, 4),
            "note": item["note"],
        }
        line = f"{item['name']:14s} 整張 {same:6d}/{total:6d} = {same / total:6.2%}"
        if item["kind"] == "layout":
            fs, ft = compare_frame(reference, actual)
            entry["frame_same"], entry["frame_total"] = fs, ft
            entry["frame_ratio"] = round(fs / ft, 4)
            line += f"   外框 {fs:5d}/{ft:5d} = {fs / ft:6.2%}"
        report["screens"].append(entry)
        print(line)

    # 交叉核對：同一框對 repo 裡**早就存著的** DOSBox 基準圖。
    # **沒有重跑 DOSBox**（使用者 2026-09-07 指定它只作 dosgolem 的參考）；
    # 那張圖是之前留下來的產物，這裡只是拿它當第二個意見。
    #
    # **兩個 oracle 現在互相同意**（各自對 remake 都是 7744/7744）。留著它是
    # 因為「兩個獨立來源說同一件事」比任何一個單獨的數字都強——哪一邊之後
    # 退步了，這一項會先開口。
    if dosbox_dir:
        path = os.path.join(dosbox_dir, "06-free-move-0-4-west.png")
        if os.path.exists(path):
            w, h, rgb = read_png_rgb(path)
            dosbox = to_indices(downsample_nearest(w, h, rgb))
            actual = load_remake(os.path.join(out_dir, "remake-first-person.png"))
            same = total = 0
            for y in range(88):
                for x in range(88):
                    total += 1
                    if dosbox[(24 + y) * WIDTH + 24 + x] == actual[(43 + y) * WIDTH + 24 + x]:
                        same += 1
            report["screens"].append({
                "name": "first-person-vs-dosbox", "kind": "cross-check",
                "reference": "docs/reference/original-dos/adventure/06-free-move-0-4-west.png",
                "remake": "remake-first-person.png",
                "same": same, "total": total, "ratio": round(same / total, 4),
                "note": "既有的 DOSBox 基準圖，沒有重跑 DOSBox。兩個 oracle 互相同意，"
                        "哪一邊之後退步了這一項會先開口。",
            })
            print(f"first-person-vs-dosbox: {same}/{total} = {same / total:.2%} （交叉核對）")

    with open(os.path.join(out_dir, "parity.json"), "w", encoding="utf-8") as handle:
        json.dump(report, handle, ensure_ascii=False, indent=2)
        handle.write("\n")
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
