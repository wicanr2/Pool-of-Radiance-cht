#!/usr/bin/env python3
"""載入 ECL 區塊時清 4A00..4A1F 的原版收據（#41）。

overlay-07 entry 3（`01C8h`）在 `02DFh..0302h` 以「基底 ＋ 索引 × 2」把 class 0 的 `4A00..4A1F`
清成 0（`DS:4959h` 非 0 時跳過）。這一支用 dosgolem 在原版裡監看 `4A01`（class 0 記錄 `2EA2:0202`
→ 線性 `2EC22h`）的寫入：

  1. 沿用 docs/audit/dosgolem-deployment-peek-goblins.json 建隊與導覽的鍵序，存成貧民窟 (14,4) 的狀態；
  2. 一條鍵序：進城 (0,4) → 市政廳門口 (3,4) → 進市政廳 → 職員格 (5,5) → 走回 (4,4) → 朝西踏出到 (3,4)，
     全程 `shots -watch 2EC22-2EC23`（dosgolem `cmd/shots` 的 -watch）。

斷言：恰好兩筆寫入——`00 → 01` 在踏進職員格那一步（`ecl3/8 9BACh SAVE 1 @4A01`，正對照），
`01 → 00` 在踏出市政廳那一步；寫入者的段逐位元組等於 overlay-07。

輸出 docs/audit/dosgolem-4a01-block-load-clear.json。"""
import json, os, re, subprocess, sys
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DOSGOLEM = os.path.join(ROOT, 'workplace', 'dosgolem')
SOURCE = os.path.join(ROOT, 'workplace', 'oracle', 'dos')
WORK = os.path.join(ROOT, 'workplace', 'dosgolem-4a01')
SCRATCH = os.path.join(WORK, 'scratch')
os.makedirs(SCRATCH, exist_ok=True)
GOBLINS = json.load(open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-deployment-peek-goblins.json')))
TAIL = ',e,b,rep:14:Return,Up,Up,Up'
PREFIX = GOBLINS['keys'][:GOBLINS['keys'].index(TAIL) + len(TAIL)]
# 每一步都有標籤，收據才對得回位置。
ROUTE = [
    ('Right', ''), ('Right', ''), ('Up', 'slums (15,4)'), ('Up', 'city (0,4)'),
    ('Up', ''), ('Up', ''), ('Up', 'city hall door (3,4)'), ('Return', ''),
    ('Up', 'hall (4,4)'), ('Return', ''), ('Up', 'hall (4,5)'), ('Return', ''),
    ('Left', ''), ('Up', 'clerk (5,5)'),
    ('Return', ''), ('Return', ''), ('Return', ''), ('Return', ''), ('Return', ''), ('Return', ''),
    ('Left', ''), ('Left', ''), ('Up', 'hall (4,5)'), ('Right', ''), ('Up', 'hall (4,4)'), ('Left', ''),
    ('Up', 'out to city (3,4)'), ('Return', ''),
]
WATCH = '2EC22-2EC23'
LINEAR_OVERLAY7_BYTES = [(0x2E0, 48), (0xCA0, 32)]

def run(keys, tag, peek, load=None, save=None, watch=None):
    args = ['docker', 'run', '--rm', '--network', 'none', '--memory', '4g', '--cpus', '2', '--pids-limit', '256',
            '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3', '-u', '%d:%d' % (os.getuid(), os.getgid()),
            '-v', DOSGOLEM + ':/dosgolem', '-v', SOURCE + ':/orig:ro', '-v', WORK + ':/work', '-v', SCRATCH + ':/scratch',
            '-v', os.path.join(DOSGOLEM, 'workplace', 'gocache') + ':/gocache', '-v', os.path.join(DOSGOLEM, 'workplace', 'gomodcache') + ':/gomodcache',
            '-e', 'GOCACHE=/gocache', '-e', 'GOMODCACHE=/gomodcache', '-e', 'HOME=/tmp', '-e', 'GOFLAGS=-mod=mod',
            '-w', '/dosgolem', 'golang:1.24-bookworm', 'go', 'run', './cmd/shots', '-exe', '/orig/start.exe', '-root', '/orig',
            '-scratch', '/scratch', '-out', '/work/' + tag, '-budget', '200000000', '-idle', '40000000', '-peek', peek, '-keys', keys]
    if load: args += ['-load-state', '/work/' + load]
    if save: args += ['-save-state', '/work/' + save]
    if watch: args += ['-watch', watch]
    p = subprocess.run(args, capture_output=True, text=True, timeout=3600)
    open(os.path.join(WORK, tag + '.stdout.txt'), 'w').write(p.stdout + p.stderr)
    if p.returncode != 0 or '看不懂' in p.stdout:
        print(p.stdout[-3000:], p.stderr[-2000:]); raise SystemExit('shots failed: ' + tag)
    return p.stdout, json.load(open(os.path.join(WORK, tag, 'shots.json')))

if not os.path.exists(os.path.join(WORK, 'slums.state')):
    run(PREFIX, 'walk-out', 'ds:6A0B:5', save='slums.state')
peek = 'ds:6A0B:5,2EA2:0202:2,1997:02E0:48,1997:0CA0:32'
stdout, shots = run(','.join(k for k, _ in ROUTE), 'route', peek, load='slums.state', watch=WATCH)

frames = []
for shot, (key, label) in zip(shots[1:], ROUTE):
    pos = bytes.fromhex(shot['peek']['ds:6A0B'].split('|')[1])
    value = int.from_bytes(bytes.fromhex(shot['peek']['2EA2:0202'].split('|')[1]), 'little')
    frames.append({'key': key, 'label': label, 'step': shot['step'], 'x': pos[0], 'y': pos[1], 'facing_0_7': pos[2],
                   'terrain_C04F': pos[4], '4A01': value, 'frame_sha256': shot['sha256']})
writes = []
for line in stdout.splitlines():
    m = re.match(r'\[watch\] #(\d+) ([0-9A-F]+): ([0-9A-F]{2}) → ([0-9A-F]{2})\s+← ([0-9A-F]{4}):([0-9A-F]{4})', line)
    if m:
        writes.append({'step': int(m.group(1)), 'linear': m.group(2), 'old': m.group(3), 'new': m.group(4), 'cs': m.group(5), 'ip': m.group(6)})

def frame_of(step):
    for f in frames:
        if f['step'] >= step:
            return f
    return None

overlay7 = open(os.path.join(ROOT, 'workplace', 'ovr', 'overlay-07.bin'), 'rb').read()
last = shots[-1]['peek']
segment_is_overlay7 = all(bytes.fromhex(last['1997:%04X' % off].split('|')[1]) == overlay7[off:off + n]
                          for off, n in LINEAR_OVERLAY7_BYTES)
low = [w for w in writes if w['linear'] == '2EC22']
problems = []
if len(low) != 2:
    problems.append('want exactly two writes to 2EC22, got %d' % len(low))
else:
    set_frame, clear_frame = frame_of(low[0]['step']), frame_of(low[1]['step'])
    if (low[0]['old'], low[0]['new']) != ('00', '01') or set_frame['label'] != 'clerk (5,5)':
        problems.append('first write should be 00→01 at the clerk: %s %s' % (low[0], set_frame))
    if (low[1]['old'], low[1]['new']) != ('01', '00') or clear_frame['label'] != 'out to city (3,4)':
        problems.append('second write should be 01→00 stepping out of City Hall: %s %s' % (low[1], clear_frame))
    if low[1]['cs'] != '1997' or low[1]['ip'] != '02FE':
        problems.append('clear should come from 1997:02FE (overlay-07 02F9h): %s' % low[1])
if not segment_is_overlay7:
    problems.append('runtime segment 1997 does not match overlay-07.bin')

out = {'schema': 'pool-dosgolem-4a01-block-load-clear/1', 'generator': 'dosgolem',
       'generator_revision': os.popen('git -C %s rev-parse --short HEAD' % DOSGOLEM).read().strip(),
       'original_exe_sha256': GOBLINS['original_exe_sha256'],
       'overlay07_sha256': 'a59f9d16a1d186bbd3806865ffe55de4be5b58484287237fb73872295da778ae',
       'walk_out_keys': PREFIX, 'route': ROUTE, 'watch_linear': WATCH,
       'class0_pointer': '2EA2:0000 (DS:4933h)', 'address_4A01': '2EA2:0202',
       'segment_1997_is_overlay07': segment_is_overlay7, 'writes': writes, 'frames': frames,
       'verdict': 'ok' if not problems else 'failed', 'problems': problems}
json.dump(out, open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-4a01-block-load-clear.json'), 'w'), ensure_ascii=False, indent=1)
print(out['verdict'], problems, writes)
if problems:
    sys.exit(1)
