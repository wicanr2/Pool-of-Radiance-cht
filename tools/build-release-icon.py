"""產生發行包用的圖示。

不用原版的標題畫面：那是 SSI 的美術，`NOTICE.md` 已經把它排除在本專案的
授權範圍外，拿它當圖示等於把原版素材放進發行包。這裡畫一個原創的圖形——
發光的池水，同心圓由中心往外變暗，外圈一道亮環。

只用標準函式庫（zlib + struct）寫 PNG，不引進影像套件。
"""

import math
import struct
import sys
import zlib

SIZE = 256


def chunk(kind: bytes, payload: bytes) -> bytes:
    return (struct.pack(">I", len(payload)) + kind + payload
            + struct.pack(">I", zlib.crc32(kind + payload) & 0xFFFFFFFF))


def pixel(x: int, y: int) -> tuple[int, int, int, int]:
    centre = (SIZE - 1) / 2
    distance = math.hypot(x - centre, y - centre) / centre
    if distance > 0.98:
        return 0, 0, 0, 0
    if distance > 0.86:
        return 255, 202, 72, 255            # 外圈亮環
    if distance > 0.80:
        return 16, 20, 30, 255
    glow = max(0.0, 1.0 - distance / 0.80) ** 1.6
    return (
        int(20 + 150 * glow),
        int(40 + 205 * glow),
        int(70 + 185 * glow),
        255,
    )


def main(path: str) -> None:
    rows = bytearray()
    for y in range(SIZE):
        rows.append(0)                       # filter type 0
        for x in range(SIZE):
            rows.extend(pixel(x, y))
    header = struct.pack(">IIBBBBB", SIZE, SIZE, 8, 6, 0, 0, 0)
    png = (b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", header)
           + chunk(b"IDAT", zlib.compress(bytes(rows), 9)) + chunk(b"IEND", b""))
    with open(path, "wb") as out:
        out.write(png)


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "packaging/linux/pool-of-radiance-remake.png")
