"""Export raw-overlay instructions that reference any of several 16-bit words.

POOL_IDA_WORDS is a comma-separated list (for example "0x110,0x111"); every
instruction whose bytes contain one of those little-endian words is reported
with the words it matched. POOL_IDA_OUT is the only writable output. The
report preserves overlay-local addresses, bytes, operands, containing
function boundaries, input hash and tool version, and renames nothing.

The multi-word form exists so one analysis pass answers a whole field group:
running IDA once per word costs a full auto-analysis each time.
"""

import hashlib
import json
import os

import ida_auto
import ida_bytes
import ida_funcs
import ida_ida
import ida_kernwin
import ida_lines
import idautils
import idc


output_path = os.environ.get("POOL_IDA_OUT", "")
words_text = os.environ.get("POOL_IDA_WORDS", "")
if not output_path or not words_text:
    raise RuntimeError("POOL_IDA_OUT and POOL_IDA_WORDS are required")
words = [int(part, 0) for part in words_text.split(",") if part.strip()]
for word in words:
    if not 0 <= word <= 0xFFFF:
        raise RuntimeError("POOL_IDA_WORDS holds a value outside uint16")

minimum = ida_ida.inf_get_min_ea()
maximum = ida_ida.inf_get_max_ea()
if not idc.set_segm_attr(minimum, idc.SEGATTR_BITNESS, 0):
    raise RuntimeError("could not set raw overlay segment to 16-bit")
ida_bytes.del_items(minimum, ida_bytes.DELIT_EXPAND, maximum - minimum)
for address in range(minimum, maximum):
    idc.create_insn(address)
ida_auto.auto_wait()

needles = {word: bytes((word & 0xFF, word >> 8)) for word in words}

matches = []
for address in idautils.Heads(minimum, maximum):
    size = idc.get_item_size(address)
    raw = ida_bytes.get_bytes(address, size) or b""
    matched = [word for word, needle in needles.items() if needle in raw]
    if not matched:
        continue
    function = ida_funcs.get_func(address)
    matches.append({
        "address": address,
        "words": matched,
        "bytes": raw.hex(),
        "text": ida_lines.tag_remove(idc.generate_disasm_line(address, 0) or ""),
        "mnemonic": idc.print_insn_mnem(address),
        "operands": [idc.print_operand(address, index) for index in range(2)],
        "function_start": function.start_ea if function else None,
        "function_end": function.end_ea if function else None,
    })

with open(idc.get_input_file_path(), "rb") as source:
    digest = hashlib.sha256(source.read()).hexdigest()
payload = {
    "schema": "pool-overlay-record-field-references-ida/1",
    "input": os.path.basename(idc.get_input_file_path()),
    "sha256": digest,
    "ida_version": ida_kernwin.get_kernel_version(),
    "address_space": "overlay-local file offset, base 0",
    "segment_bitness": idc.get_segm_attr(minimum, idc.SEGATTR_BITNESS),
    "words": words,
    "matches": matches,
}
with open(output_path, "w", encoding="utf-8") as output:
    json.dump(payload, output, indent=2)
idc.qexit(0)
