"""Export one exact 16-bit raw-overlay instruction span without renaming it."""

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

instructions = []
cursor = minimum + start
while cursor < minimum + end:
    ida_bytes.del_items(cursor, ida_bytes.DELIT_SIMPLE, 1)
    if not idc.create_insn(cursor):
        raise RuntimeError("could not decode overlay offset 0x%X" % (cursor - minimum))
    size = idc.get_item_size(cursor)
    instructions.append({
        "address": cursor - minimum,
        "bytes": (ida_bytes.get_bytes(cursor, size) or b"").hex(),
        "text": ida_lines.tag_remove(idc.generate_disasm_line(cursor, 0) or ""),
        "operands": [idc.print_operand(cursor, index) for index in range(2)],
        "operand_values": [idc.get_operand_value(cursor, index) for index in range(2)],
    })
    cursor += size

with open(input_path, "rb") as source:
    digest = hashlib.sha256(source.read()).hexdigest()
payload = {
    "schema": "pool-overlay-span-ida/1",
    "input": os.path.basename(input_path),
    "sha256": digest,
    "ida_version": ida_kernwin.get_kernel_version(),
    "address_space": "overlay-local file offset, base 0, 16-bit metapc",
    "start": start,
    "end": end,
    "instructions": instructions,
}
with open(output_path, "w", encoding="utf-8") as output:
    json.dump(payload, output, indent=2)
idc.qexit(0)
