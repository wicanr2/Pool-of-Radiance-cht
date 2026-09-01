"""Export raw-overlay instructions containing one little-endian word.

IDA loads the overlay as a 16-bit raw binary.  POOL_IDA_WORD is a word such
as 0x6A7C; POOL_IDA_OUT is the only writable output.  The report preserves
overlay-local addresses, bytes, operands, containing function boundaries,
input hash, and tool version.  It does not rename or annotate the database.
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
word_text = os.environ.get("POOL_IDA_WORD", "")
if not output_path or not word_text:
    raise RuntimeError("POOL_IDA_OUT and POOL_IDA_WORD are required")
word = int(word_text, 0)
if not 0 <= word <= 0xFFFF:
    raise RuntimeError("POOL_IDA_WORD is outside uint16")

minimum = ida_ida.inf_get_min_ea()
maximum = ida_ida.inf_get_max_ea()
if not idc.set_segm_attr(minimum, idc.SEGATTR_BITNESS, 0):
    raise RuntimeError("could not set raw overlay segment to 16-bit")
ida_bytes.del_items(minimum, ida_bytes.DELIT_EXPAND, maximum - minimum)
for address in range(minimum, maximum):
    idc.create_insn(address)
ida_auto.auto_wait()

needle = bytes((word & 0xFF, word >> 8))


def instruction(address):
    size = idc.get_item_size(address)
    function = ida_funcs.get_func(address)
    return {
        "address": address,
        "bytes": ida_bytes.get_bytes(address, size).hex(),
        "text": ida_lines.tag_remove(idc.generate_disasm_line(address, 0) or ""),
        "operands": [idc.print_operand(address, index) for index in range(2)],
        "operand_values": [idc.get_operand_value(address, index) for index in range(2)],
        "function_start": function.start_ea if function else None,
        "function_end": function.end_ea if function else None,
    }


matches = []
for address in idautils.Heads(minimum, maximum):
    size = idc.get_item_size(address)
    raw = ida_bytes.get_bytes(address, size) or b""
    if needle in raw:
        matches.append(instruction(address))

with open(idc.get_input_file_path(), "rb") as source:
    digest = hashlib.sha256(source.read()).hexdigest()
payload = {
    "schema": "pool-overlay-word-references-ida/1",
    "input": os.path.basename(idc.get_input_file_path()),
    "sha256": digest,
    "ida_version": ida_kernwin.get_kernel_version(),
    "address_space": "overlay-local file offset, base 0",
    "segment_bitness": idc.get_segm_attr(minimum, idc.SEGATTR_BITNESS),
    "word": word,
    "matches": matches,
}
with open(output_path, "w", encoding="utf-8") as output:
    json.dump(payload, output, indent=2)
idc.qexit(0)
