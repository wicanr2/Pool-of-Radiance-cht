"""Export the resident Pool archive push/restore helpers and their callers.

IDA loads START.EXE with its MZ loader.  The helper seeds are exact load-module
file offsets established from the original bytes; this script preserves both
IDA linear addresses and file offsets and does not rename the database.
"""

import hashlib
import json
import os

import ida_auto
import ida_bytes
import ida_kernwin
import ida_lines
import ida_loader
import ida_xref
import idautils
import idc


output_path = os.environ.get("POOL_IDA_OUT", "")
if not output_path:
    raise RuntimeError("POOL_IDA_OUT is required")

ida_auto.auto_wait()
input_path = idc.get_input_file_path()
image_base = idc.get_inf_attr(idc.INF_MIN_EA)
header_size = 0x3B0
seeds = [0x2A2F, 0x2A44]


def ea_from_file_offset(file_offset):
    return image_base + file_offset - header_size


def instruction(ea):
    size = idc.get_item_size(ea)
    return {
        "address": ea,
        "file_offset": ida_loader.get_fileregion_offset(ea),
        "bytes": (ida_bytes.get_bytes(ea, size) or b"").hex(),
        "text": ida_lines.tag_remove(idc.generate_disasm_line(ea, 0) or ""),
        "operands": [idc.print_operand(ea, index) for index in range(2)],
    }


helpers = []
for file_offset in seeds:
    ea = ea_from_file_offset(file_offset)
    items = []
    cursor = ea
    for _ in range(16):
        idc.create_insn(cursor)
        item = instruction(cursor)
        items.append(item)
        mnemonic = idc.print_insn_mnem(cursor).lower()
        if mnemonic in ("retf", "ret", "retfq"):
            break
        size = idc.get_item_size(cursor)
        if size <= 0:
            raise RuntimeError("IDA could not decode helper at 0x%X" % cursor)
        cursor += size
    else:
        raise RuntimeError("archive helper at file offset 0x%X has no return" % file_offset)
    callers = []
    for caller in idautils.XrefsTo(ea, ida_xref.XREF_FAR):
        callers.append(instruction(caller.frm))
    helpers.append({
        "seed_file_offset": file_offset,
        "start_address": ea,
        "end_address": cursor + idc.get_item_size(cursor),
        "instructions": items,
        "callers": callers,
    })

with open(input_path, "rb") as source:
    digest = hashlib.sha256(source.read()).hexdigest()
payload = {
    "schema": "pool-resident-archive-switch-ida/1",
    "input": os.path.basename(input_path),
    "sha256": digest,
    "ida_version": ida_kernwin.get_kernel_version(),
    "address_space": "IDA MZ-loader linear address; file offset retained per instruction",
    "mz_header_size": header_size,
    "helpers": helpers,
}
with open(output_path, "w", encoding="utf-8") as output:
    json.dump(payload, output, indent=2)
idc.qexit(0)
