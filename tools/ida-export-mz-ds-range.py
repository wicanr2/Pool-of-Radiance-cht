"""Export one initialized MZ data-segment range with loader provenance."""

import hashlib
import json
import os

import ida_auto
import ida_bytes
import ida_ida
import ida_kernwin
import ida_loader
import ida_segment
import idautils
import idc


output_path = os.environ.get("POOL_IDA_OUT", "")
offset = int(os.environ.get("POOL_IDA_DS_OFFSET", ""), 0)
length = int(os.environ.get("POOL_IDA_LENGTH", ""), 0)
if not output_path or offset < 0 or length <= 0:
    raise RuntimeError("POOL_IDA_OUT, DS_OFFSET, and positive LENGTH are required")

ida_auto.auto_wait()
input_path = idc.get_input_file_path()
entry = ida_ida.inf_get_start_ea()
selector = idc.get_sreg(entry, "ds")
segment = None
for start in idautils.Segments():
    candidate = ida_segment.getseg(start)
    if candidate.sel == selector:
        segment = candidate
        break
if segment is None:
    raise RuntimeError("entry DS selector has no IDA segment")
address = segment.start_ea + offset
if address + length > segment.end_ea:
    raise RuntimeError("requested DS range is outside the data segment")
raw = ida_bytes.get_bytes(address, length)
if raw is None or len(raw) != length:
    raise RuntimeError("IDA did not return the complete initialized DS range")

with open(input_path, "rb") as source:
    digest = hashlib.sha256(source.read()).hexdigest()
payload = {
    "schema": "pool-mz-ds-range-ida/1",
    "input": os.path.basename(input_path),
    "sha256": digest,
    "ida_version": ida_kernwin.get_kernel_version(),
    "address_space": "entry DS-relative offset mapped by the IDA MZ loader",
    "ds_selector": selector,
    "segment_start": segment.start_ea,
    "segment_name": ida_segment.get_segm_name(segment),
    "ds_offset": offset,
    "length": length,
    "linear_address": address,
    "file_offset": ida_loader.get_fileregion_offset(address),
    "bytes": raw.hex(),
}
with open(output_path, "w", encoding="utf-8") as output:
    json.dump(payload, output, indent=2)
idc.qexit(0)
