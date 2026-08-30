"""Export fixed Pool overlay entry functions without destructive renaming.

Container usage sets POOL_IDA_SEEDS to comma-separated local offsets and
POOL_IDA_OUT to an explicit writable JSON path, then runs IDA with a raw
binary loader. The output preserves local addresses, bytes, operands, input
hash, IDA version, and 16-bit segment mode.
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


input_path = idc.get_input_file_path()
output_path = os.environ.get("POOL_IDA_OUT", "")
seed_text = os.environ.get("POOL_IDA_SEEDS", "")
if not output_path or not seed_text:
    raise RuntimeError("POOL_IDA_OUT and POOL_IDA_SEEDS are required")
seeds = [int(item, 0) for item in seed_text.split(",")]

minimum = ida_ida.inf_get_min_ea()
maximum = ida_ida.inf_get_max_ea()
if not idc.set_segm_attr(minimum, idc.SEGATTR_BITNESS, 0):
    raise RuntimeError("could not set raw overlay segment to 16-bit")
ida_bytes.del_items(minimum, ida_bytes.DELIT_EXPAND, maximum - minimum)
for address in seeds:
    idc.create_insn(address)
    ida_funcs.add_func(address)
ida_auto.auto_wait()


def instruction(address):
    size = idc.get_item_size(address)
    return {
        "address": address,
        "bytes": ida_bytes.get_bytes(address, size).hex(),
        "text": ida_lines.tag_remove(idc.generate_disasm_line(address, 0) or ""),
        "operands": [idc.print_operand(address, index) for index in range(2)],
        "operand_values": [idc.get_operand_value(address, index) for index in range(2)],
    }


functions = []
for seed in seeds:
    function = ida_funcs.get_func(seed)
    if function is None:
        functions.append({"seed": seed, "error": "no function"})
        continue
    functions.append(
        {
            "seed": seed,
            "start": function.start_ea,
            "end": function.end_ea,
            "instructions": [instruction(address) for address in idautils.FuncItems(function.start_ea)],
        }
    )

with open(input_path, "rb") as source:
    digest = hashlib.sha256(source.read()).hexdigest()
payload = {
    "schema": "pool-overlay-functions-ida/1",
    "input": os.path.basename(input_path),
    "sha256": digest,
    "ida_version": ida_kernwin.get_kernel_version(),
    "address_space": "overlay-local file offset, base 0",
    "segment_bitness": idc.get_segm_attr(minimum, idc.SEGATTR_BITNESS),
    "entry_seeds": seeds,
    "functions": functions,
}
with open(output_path, "w", encoding="utf-8") as output:
    json.dump(payload, output, indent=2)
idc.qexit(0)
