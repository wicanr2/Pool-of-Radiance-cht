"""Export a 16-bit raw-overlay span, recording undecodable bytes instead of failing.

ida-export-overlay-span.py 逐位元組 create_insn，碰到解不開的位元組就丟例外——
而 headless 的例外看起來就是「沒有訊息、沒有輸出檔」。overlay 裡夾著跳躍表、
字串常數與 Turbo Pascal 的區段標頭，所以一段夠長的程式碼幾乎一定會撞到。

這一支照樣逐位元組解，解不開就把那個 byte 記成 data 再往下走一格，並在輸出
裡留下每一段 data 的位置與內容。判讀時要自己決定那一段是資料還是解錯位——
兩者在這裡看起來一樣。
"""

import hashlib
import json
import os

import ida_bytes
import ida_ida
import ida_kernwin
import ida_lines
import idc


input_path = idc.get_input_file_path()
output_path = os.environ.get("POOL_IDA_OUT", "")
start = int(os.environ.get("POOL_IDA_START", ""), 0)
end = int(os.environ.get("POOL_IDA_END", ""), 0)
if not output_path or start < 0 or end <= start:
    raise RuntimeError("POOL_IDA_OUT and a valid START/END span are required")

minimum = ida_ida.inf_get_min_ea()
maximum = ida_ida.inf_get_max_ea()
if end > maximum - minimum:
    raise RuntimeError("requested overlay span is outside the input")
idc.set_segm_attr(minimum, idc.SEGATTR_BITNESS, 0)

items = []
cursor = minimum + start
while cursor < minimum + end:
    ida_bytes.del_items(cursor, ida_bytes.DELIT_SIMPLE, 1)
    if idc.create_insn(cursor):
        size = idc.get_item_size(cursor)
        items.append({
            "kind": "insn",
            "address": cursor - minimum,
            "bytes": (ida_bytes.get_bytes(cursor, size) or b"").hex(),
            "text": ida_lines.tag_remove(idc.generate_disasm_line(cursor, 0) or ""),
            "operands": [idc.print_operand(cursor, index) for index in range(2)],
            "operand_values": [idc.get_operand_value(cursor, index) for index in range(2)],
        })
        cursor += size
        continue
    if items and items[-1]["kind"] == "data" and \
            items[-1]["address"] + len(items[-1]["bytes"]) // 2 == cursor - minimum:
        items[-1]["bytes"] += (ida_bytes.get_bytes(cursor, 1) or b"").hex()
    else:
        items.append({
            "kind": "data",
            "address": cursor - minimum,
            "bytes": (ida_bytes.get_bytes(cursor, 1) or b"").hex(),
        })
    cursor += 1

with open(input_path, "rb") as source:
    digest = hashlib.sha256(source.read()).hexdigest()
payload = {
    "schema": "pool-overlay-span-lenient-ida/1",
    "input": os.path.basename(input_path),
    "sha256": digest,
    "ida_version": ida_kernwin.get_kernel_version(),
    "address_space": "overlay-local file offset, base 0, 16-bit metapc",
    "start": start,
    "end": end,
    "instruction_count": sum(1 for item in items if item["kind"] == "insn"),
    "data_runs": sum(1 for item in items if item["kind"] == "data"),
    "items": items,
}
with open(output_path, "w", encoding="utf-8") as sink:
    json.dump(payload, sink, ensure_ascii=False, indent=1)
