"""Export the MZ loader's segment map without modifying names or analysis."""

import hashlib
import json
import os

import ida_auto
import ida_ida
import ida_kernwin
import ida_loader
import ida_segment
import idautils
import idc


output_path = os.environ.get("POOL_IDA_OUT", "")
if not output_path:
    raise RuntimeError("POOL_IDA_OUT is required")

ida_auto.auto_wait()
input_path = idc.get_input_file_path()
segments = []
for start in idautils.Segments():
    segment = ida_segment.getseg(start)
    segments.append(
        {
            "selector": segment.sel,
            "start": segment.start_ea,
            "end": segment.end_ea,
            "name": ida_segment.get_segm_name(segment),
            "class": ida_segment.get_segm_class(segment),
            "file_offset_start": ida_loader.get_fileregion_offset(segment.start_ea),
            "entry_ds_selector": idc.get_sreg(ida_ida.inf_get_start_ea(), "ds"),
        }
    )

with open(input_path, "rb") as source:
    digest = hashlib.sha256(source.read()).hexdigest()
payload = {
    "schema": "pool-mz-segments-ida/1",
    "input": os.path.basename(input_path),
    "sha256": digest,
    "ida_version": ida_kernwin.get_kernel_version(),
    "address_space": "IDA MZ loader linear address",
    "segments": segments,
}
with open(output_path, "w", encoding="utf-8") as output:
    json.dump(payload, output, indent=2)
idc.qexit(0)
