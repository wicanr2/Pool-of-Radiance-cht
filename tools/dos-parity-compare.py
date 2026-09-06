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
    plan = [
        {
            "name": "title",
            "digest": "04cfb632f3f3fe4dc15b39dc13a74b7f68f386014132c9e67595517e4968ac68",
            "remake": "remake-title.png",
            "ref_box": [0, 0, 320, 200], "remake_box": [0, 0, 320, 200],
            "status": "compared",
            "note": "整張。原版與 remake 畫的是同一份 TITLE.DAX，位置與縮放相同；"
                    "唯一預期的差異是 remake 自己加的按鍵提示那一行。",
        },
        {
            "name": "first-person",
            "digest": "fc2383a02d907c874c4e8ab381291b5fd952c89d21002ff73e241b18731754f7",
            "remake": "remake-first-person.png",
            "ref_box": [24, 24, 88, 88], "remake_box": [24, 43, 88, 88],
            "status": "blocked",
            "blocked_by": "dosgolem 還沒實作 EGA 圖形控制器（3CE／3CF），"
                          "第一人稱那一框在基準側是黑底白線框——"
                          "dosgolem docs/findings/004。這個數字量的是 oracle 的缺口，"
                          "不是 remake 的正確性。",
            "note": "第一人稱框內的 88x88。remake 的框在畫面上比原版低 19 列。",
        },
    ]

    report = {"screens": []}
    failed = False
    for item in plan:
        matches = [s for s in shots if s["sha256"] == item["digest"]]
        if not matches:
            print(f"基準裡找不到 {item['name']} 的畫面（雜湊 {item['digest'][:8]}）"
                  f"——鍵序改過了，先更新對照表", file=sys.stderr)
            failed = True
            continue
        info = matches[0]
        raw = open(os.path.join(ref_dir, info["path"].replace(".png", ".idx")), "rb").read()
        reference = bytes(v & 0x0F for v in raw)
        actual = load_remake(os.path.join(out_dir, item["remake"]))

        rl, rt, w, h = item["ref_box"]
        ml, mt, _, _ = item["remake_box"]
        same = total = 0
        for y in range(h):
            for x in range(w):
                total += 1
                if reference[(rt + y) * WIDTH + rl + x] == actual[(mt + y) * WIDTH + ml + x]:
                    same += 1
        entry = {
            "name": item["name"], "status": item["status"],
            "reference": info["path"], "reference_step": info["step"],
            "remake": item["remake"],
            "same": same, "total": total, "ratio": round(same / total, 4),
            "note": item["note"],
        }
        if "blocked_by" in item:
            entry["blocked_by"] = item["blocked_by"]
        report["screens"].append(entry)
        mark = "" if item["status"] == "compared" else "（基準側有缺口）"
        print(f"{item['name']}: {same}/{total} = {same / total:.2%} {mark}")

    with open(os.path.join(out_dir, "parity.json"), "w", encoding="utf-8") as handle:
        json.dump(report, handle, ensure_ascii=False, indent=2)
        handle.write("\n")
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
