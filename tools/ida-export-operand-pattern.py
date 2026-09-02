"""Export raw-overlay instructions whose disassembly matches a regex.

POOL_IDA_PATTERN is a Python regex matched against the plain-text disassembly
line (case-insensitive); POOL_IDA_OUT is the only writable output. Matching on
the rendered operand finds record-field references whose displacement is a
single byte, which a byte-pattern search cannot separate from ordinary data.

The report preserves overlay-local addresses, bytes, operands, containing
function boundaries, input hash and IDA version, and renames nothing.
"""

import hashlib
import json
import os
import re

import ida_auto
import ida_bytes
import ida_funcs
import ida_ida
import ida_kernwin
import ida_lines
import idautils
import idc


output_path = os.environ.get("POOL_IDA_OUT", "")
pattern_text = os.environ.get("POOL_IDA_PATTERN", "")
if not output_path or not pattern_text:
    raise RuntimeError("POOL_IDA_OUT and POOL_IDA_PATTERN are required")
pattern = re.compile(pattern_text, re.IGNORECASE)

minimum = ida_ida.inf_get_min_ea()
maximum = ida_ida.inf_get_max_ea()
if not idc.set_segm_attr(minimum, idc.SEGATTR_BITNESS, 0):
    raise RuntimeError("could not set raw overlay segment to 16-bit")
ida_bytes.del_items(minimum, ida_bytes.DELIT_EXPAND, maximum - minimum)
for address in range(minimum, maximum):
    idc.create_insn(address)
ida_auto.auto_wait()

matches = []
for address in idautils.Heads(minimum, maximum):
    text = ida_lines.tag_remove(idc.generate_disasm_line(address, 0) or "")
    if not pattern.search(text):
        continue
    size = idc.get_item_size(address)
    function = ida_funcs.get_func(address)
    matches.append({
        "address": address,
        "bytes": (ida_bytes.get_bytes(address, size) or b"").hex(),
        "text": text,
        "mnemonic": idc.print_insn_mnem(address),
        "operands": [idc.print_operand(address, index) for index in range(2)],
        "function_start": function.start_ea if function else None,
        "function_end": function.end_ea if function else None,
    })

with open(idc.get_input_file_path(), "rb") as source:
    digest = hashlib.sha256(source.read()).hexdigest()
payload = {
    "schema": "pool-overlay-operand-pattern-ida/1",
    "input": os.path.basename(idc.get_input_file_path()),
    "sha256": digest,
    "ida_version": ida_kernwin.get_kernel_version(),
    "address_space": "overlay-local file offset, base 0",
    "segment_bitness": idc.get_segm_attr(minimum, idc.SEGATTR_BITNESS),
    "pattern": pattern_text,
    "matches": matches,
}
with open(output_path, "w", encoding="utf-8") as output:
    json.dump(payload, output, indent=2)
idc.qexit(0)
