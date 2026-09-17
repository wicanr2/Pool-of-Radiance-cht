"""原版 DOS 畫面的文字讀取（dosgolem 的 .idx：320×200 色號）。

原版文字對齊 40×25 的 8×8 字格。每一格取「出現最多的顏色」當背景，其餘位元組成 64-bit 簽章，
與 cmd/dos-screen-text 的做法相同（先出現的顏色在平手時優先）。簽章表由已知字串的畫面收集，
存在 workplace（原版字形衍生資料，不進版控）。"""
import json, os

COLUMNS, ROWS = 40, 25

def signature(idx, column, row):
    values = [idx[(row * 8 + y) * 320 + column * 8 + x] & 0x0F for y in range(8) for x in range(8)]
    counts = {}
    order = []
    for v in values:
        if v not in counts:
            counts[v] = 0
            order.append(v)
        counts[v] += 1
    background = max(order, key=lambda v: (counts[v], -order.index(v)))
    bits = 0
    for i, v in enumerate(values):
        if v != background:
            bits |= 1 << i
    return bits

def load_font(path):
    if not os.path.exists(path):
        return {}
    return {int(k, 16): v for k, v in json.load(open(path)).items()}

def save_font(path, font):
    json.dump({'%016x' % k: v for k, v in sorted(font.items())}, open(path, 'w'), ensure_ascii=False, indent=0)

def read_rows(idx, font):
    rows = []
    for row in range(ROWS):
        line = ''
        for column in range(COLUMNS):
            bits = signature(idx, column, row)
            line += ' ' if bits == 0 else font.get(bits, '?')
        rows.append(line.rstrip())
    return rows

def learn(idx, row, column, text, font):
    """把畫面上 (row, column) 起的一串已知文字學進簽章表；回傳衝突（同簽章不同字）。"""
    conflicts = []
    for i, ch in enumerate(text):
        if ch == ' ':
            continue
        bits = signature(idx, column + i, row)
        if bits == 0:
            conflicts.append((ch, 'blank'))
            continue
        if bits in font and font[bits] != ch:
            conflicts.append((ch, font[bits]))
            continue
        font[bits] = ch
    return conflicts

def align(idx, row, text, first=0):
    """在第 row 列找出 text 的起始欄：空白位置必須是空格、非空白位置必須有字。回傳所有吻合的欄。"""
    hits = []
    for column in range(first, COLUMNS - len(text) + 1):
        if all((ch == ' ') == (signature(idx, column + i, row) == 0) for i, ch in enumerate(text)):
            hits.append(column)
    return hits

EGA16 = [(0, 0, 0), (0, 0, 170), (0, 170, 0), (0, 170, 170), (170, 0, 0), (170, 0, 170), (170, 85, 0), (170, 170, 170),
         (85, 85, 85), (85, 85, 255), (85, 255, 85), (85, 255, 255), (255, 85, 85), (255, 85, 255), (255, 255, 85), (255, 255, 255)]

def write_png(path, idx):
    import struct, zlib
    raw = b''.join(b'\x00' + bytes(c for x in range(320) for c in EGA16[idx[y * 320 + x] & 15]) for y in range(200))
    def chunk(tag, data):
        return struct.pack('>I', len(data)) + tag + data + struct.pack('>I', zlib.crc32(tag + data) & 0xffffffff)
    open(path, 'wb').write(b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', struct.pack('>IIBBBBB', 320, 200, 8, 2, 0, 0, 0)) +
                           chunk(b'IDAT', zlib.compress(raw)) + chunk(b'IEND', b''))
