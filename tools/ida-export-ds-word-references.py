"""Export code references to one resident DOS data-segment word.

IDA loads the MZ executable with its normal loader.  POOL_IDA_DS_WORD is a
little-endian 16-bit displacement such as 0x4954; POOL_IDA_OUT is the only
writable output.  The report preserves IDA addresses, file offsets, bytes,
operands, containing function boundaries, input hash, and tool version.  It
does not rename anything or write annotations into the database.
"""

import hashlib
import json
import os

import ida_auto
import ida_bytes
import ida_funcs
import ida_kernwin
import ida_lines
import ida_loader
import idautils
import idc


output_path = os.environ.get("POOL_IDA_OUT", "")
word_text = os.environ.get("POOL_IDA_DS_WORD", "")
if not output_path or not word_text:
    raise RuntimeError("POOL_IDA_OUT and POOL_IDA_DS_WORD are required")
word = int(word_text, 0)
if not 0 <= word <= 0xFFFF:
    raise RuntimeError("POOL_IDA_DS_WORD is outside uint16")

ida_auto.auto_wait()
input_path = idc.get_input_file_path()
needle = bytes((word & 0xFF, word >> 8))
minimum = idc.get_inf_attr(idc.INF_MIN_EA)
maximum = idc.get_inf_attr(idc.INF_MAX_EA)


def instruction(address):
    size = idc.get_item_size(address)
    function = ida_funcs.get_func(address)
    return {
        "address": address,
        "file_offset": ida_loader.get_fileregion_offset(address),
        "bytes": ida_bytes.get_bytes(address, size).hex(),
        "text": ida_lines.tag_remove(idc.generate_disasm_line(address, 0) or ""),
        "operands": [idc.print_operand(address, index) for index in range(2)],
        "function_start": function.start_ea if function else None,
        "function_end": function.end_ea if function else None,
    }


matches = []
for address in idautils.Heads(minimum, maximum):
    size = idc.get_item_size(address)
    raw = ida_bytes.get_bytes(address, size) or b""
    if needle in raw:
        matches.append(instruction(address))

with open(input_path, "rb") as source:
    digest = hashlib.sha256(source.read()).hexdigest()
payload = {
    "schema": "pool-resident-ds-references-ida/1",
    "input": os.path.basename(input_path),
    "sha256": digest,
    "ida_version": ida_kernwin.get_kernel_version(),
    "address_space": "IDA MZ loader linear address; file_offset recorded per instruction",
    "ds_word": word,
    "matches": matches,
}
with open(output_path, "w", encoding="utf-8") as output:
    json.dump(payload, output, indent=2)
idc.qexit(0)
