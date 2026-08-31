"""Export one DOS MZ runtime segment:offset function without renaming it.

POOL_IDA_SEGMENT and POOL_IDA_OFFSET are the original load-module address;
POOL_IDA_OUT is an explicit JSON path. IDA must load the executable with its
MZ loader so the selector mapping remains authoritative.
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
import ida_segment
import idautils
import idc


input_path = idc.get_input_file_path()
output_path = os.environ.get("POOL_IDA_OUT", "")
segment_value = int(os.environ.get("POOL_IDA_SEGMENT", ""), 0)
offset_value = int(os.environ.get("POOL_IDA_OFFSET", ""), 0)
if not output_path:
    raise RuntimeError("POOL_IDA_OUT is required")

# The DOS loader rebases the load module (normally to 0x10000 in IDA), so
# runtime selector values are not IDA selectors. Preserve both spaces and
# derive the linear EA from the loader's image base.
image_base = ida_ida.inf_get_min_ea()
address = image_base + segment_value * 16 + offset_value
segment = ida_segment.getseg(address)
if segment is None:
    raise RuntimeError(
        f"IDA has no segment containing runtime {segment_value:04X}:{offset_value:04X}"
    )
idc.create_insn(address)
ida_funcs.add_func(address)
ida_auto.auto_wait()
function = ida_funcs.get_func(address)
if function is None:
    raise RuntimeError(f"IDA has no function at {segment_value:04X}:{offset_value:04X}")


def instruction(ea):
    size = idc.get_item_size(ea)
    return {
        "linear_address": ea,
        "runtime_segment": segment_value,
        "runtime_offset": offset_value + (ea - address),
        "bytes": ida_bytes.get_bytes(ea, size).hex(),
        "text": ida_lines.tag_remove(idc.generate_disasm_line(ea, 0) or ""),
        "operands": [idc.print_operand(ea, index) for index in range(2)],
        "operand_values": [idc.get_operand_value(ea, index) for index in range(2)],
    }


with open(input_path, "rb") as source:
    digest = hashlib.sha256(source.read()).hexdigest()
payload = {
    "schema": "pool-mz-function-ida/1",
    "input": os.path.basename(input_path),
    "sha256": digest,
    "ida_version": ida_kernwin.get_kernel_version(),
    "address_space": "DOS load-module runtime segment:offset; IDA linear address retained",
    "ida_image_base": image_base,
    "seed": {"segment": segment_value, "offset": offset_value, "linear": address},
    "function": {
        "start_linear": function.start_ea,
        "end_linear": function.end_ea,
        "instructions": [instruction(ea) for ea in idautils.FuncItems(function.start_ea)],
    },
}
with open(output_path, "w", encoding="utf-8") as output:
    json.dump(payload, output, indent=2)
idc.qexit(0)
