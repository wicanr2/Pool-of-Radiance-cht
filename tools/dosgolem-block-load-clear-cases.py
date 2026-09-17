#!/usr/bin/env python3
"""載入 ECL 區塊時的清除：跨 archive 與讀檔兩種情形的原版收據（#41，spec 106）。

`tools/dosgolem-4a01-block-load.py` 量的是同一個 archive 內換區（市政廳 ecl3/8 → 城區 ecl3/0）。
這一支補另外兩種：

  A. 跨 archive、class 0：貧民窟（ecl2/20，`4A00 = FFh`）→ 城區（ecl3/0），監看 `4A00..4A1F`
     （`2EA2:0200..023F` → 線性 `2EC20h..2EC5Fh`）。斷言：前三步沒有換區、沒有寫入；進城那一步
     `4A00 FF→00` 來自 overlay-07 `02F9h` 的下一條（`xxxx:02FE`），**之後**城區入口的 ECL `SAVE`
     才寫範圍內的值——清除在入口之前。
  B. 跨 archive、class 1：同一條路，監看 `6E79..6E82`（`[4937h] = 2F22:0000` → 線性 `2F912h..2F925h`）。
     斷言：`6E7D 0B→00` 來自 `xxxx:0323`（`031Eh` 的下一條），之後入口才寫回。
  C. 讀檔：在職員格（`4A01 = 1`，市政廳 ecl3/8）紮營存 J 槽；重開程式、建隊選單 L 讀 J。
     斷言：讀回之後 `4A01 = 1`；讀檔過程 `DS:4959h`（線性 `CE59h`）沒有 CPU 寫入；讀檔之後第一次
     換區（走出市政廳）`4A01 01→00` 仍來自 `xxxx:02FE`，且那一段逐位元組等於 overlay-07。

輸出 docs/audit/dosgolem-block-load-clear-cases.json。狀態檔、存檔都在 workplace/dosgolem-4a01。"""
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
OVERLAY7 = open(os.path.join(ROOT, 'workplace', 'ovr', 'overlay-07.bin'), 'rb').read()

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

def watches(stdout):
    out = []
    for line in stdout.splitlines():
        m = re.match(r'\[watch\] #(\d+) ([0-9A-F]+): ([0-9A-F]{2}) → ([0-9A-F]{2})\s+← ([0-9A-F]{4}):([0-9A-F]{4})', line)
        if m:
            out.append({'step': int(m.group(1)), 'linear': m.group(2), 'old': m.group(3), 'new': m.group(4),
                        'cs': m.group(5), 'ip': m.group(6)})
    return out

def hexpeek(shot, key):
    return bytes.fromhex(shot['peek'][key].split('|')[1])

def frames(shots, keys):
    rows = []
    for shot, key in zip(shots[1:], keys):
        pos = hexpeek(shot, '0850:6A0B')
        rows.append({'key': key, 'step': shot['step'], 'x': pos[0], 'y': pos[1], 'terrain_C04F': pos[4]})
    return rows

def key_at(rows, step):
    """寫入發生在哪一個按鍵之後（第一個 step 不小於它的畫面）。"""
    for index, row in enumerate(rows):
        if row['step'] >= step:
            return index
    return None

problems = []
if not os.path.exists(os.path.join(WORK, 'slums.state')):
    run(PREFIX, 'walk-out', '0850:6A0B:5', save='slums.state')

# A／B：貧民窟 (14,4) → (15,4) → 城區 (0,4) → (1,4)。
CROSS = ['Right', 'Right', 'Up', 'Up', 'Up']
PEEK = '0850:6A0B:5,0850:4933:8,0850:4959:1,2EA2:0200:40,2F22:06F2:14,2EA2:01CA:4,2F22:05A4:4,2F22:05C2:2'
out0, shots0 = run(','.join(CROSS), 'cases-cross-class0', PEEK, load='slums.state', watch='2EC20-2EC5F')
rows0 = frames(shots0, CROSS)
w0 = watches(out0)
first = shots0[0]
if hexpeek(first, '0850:4933') != bytes.fromhex('0000a22e0000222f'):
    problems.append('class 0／1 pointers moved: %s' % first['peek']['0850:4933'])
if hexpeek(first, '2EA2:0200')[:2] != b'\xff\x00':
    problems.append('positive control: 4A00 should be FF in the slums state')
city = next((i for i, r in enumerate(rows0) if r['x'] == 0 and r['y'] == 4 and r['terrain_C04F'] == 0x19), None)
if city is None:
    problems.append('route A never reached city (0,4): %s' % rows0)
clear0 = [w for w in w0 if w['linear'] == '2EC20' and (w['old'], w['new']) == ('FF', '00')]
if len(clear0) != 1 or clear0[0]['ip'] != '02FE' or key_at(rows0, clear0[0]['step']) != city:
    problems.append('A: want one 4A00 FF→00 from :02FE on the step into the city: %s' % w0)
elif any(key_at(rows0, w['step']) < city for w in w0):
    problems.append('A: writes before the block change: %s' % w0)
elif not [w for w in w0 if w['step'] > clear0[0]['step'] and w['old'] == '00' and w['new'] != '00']:
    problems.append('A: no entry write after the clear (ordering not shown): %s' % w0)

out1, shots1 = run(','.join(CROSS), 'cases-cross-class1', '0850:6A0B:5', load='slums.state', watch='2F912-2F925')
rows1 = frames(shots1, CROSS)
w1 = watches(out1)
clear1 = [w for w in w1 if w['linear'] == '2F91A' and (w['old'], w['new']) == ('0B', '00')]
if len(clear1) != 1 or clear1[0]['ip'] != '0323' or key_at(rows1, clear1[0]['step']) != city:
    problems.append('B: want one 6E7D 0B→00 from :0323 on the step into the city: %s' % w1)
elif not [w for w in w1 if w['linear'] == '2F91A' and w['step'] > clear1[0]['step'] and w['new'] == '0B']:
    problems.append('B: the city entry did not write 6E7D back after the clear: %s' % w1)
for w in clear0 + clear1:
    seg = int(w['cs'], 16)
    if seg != 0x1997:
        problems.append('A／B: writer segment %s, want 1997 (overlay-07 at this state)' % w['cs'])

# C：職員格存檔、重開讀檔、走出市政廳。
CLERK = ['Right', 'Right', 'Up', 'Up', 'Up', 'Up', 'Up', 'Return', 'Up', 'Return', 'Up', 'Return', 'Left', 'Up',
         'Return', 'Return', 'Return', 'Return', 'Return', 'Return']
_, shotsc = run(','.join(CLERK), 'cases-clerk', '0850:6A0B:5,2EA2:0202:2', load='slums.state', save='clerk.state')
if hexpeek(shotsc[-1], '2EA2:0202') != b'\x01\x00':
    problems.append('C: 4A01 is not 1 at the clerk')
run('e,s,J', 'cases-save', '2EA2:0202:2', load='clerk.state')
if not os.path.exists(os.path.join(SCRATCH, 'savgamJ.dat')):
    problems.append('C: savgamJ.dat was not written')
LOAD = 'rep:9:Space,Return,Return,l,J'
outl, shotsl = run(LOAD, 'cases-load', '0850:6A0B:5,0850:4959:1,2EA2:0202:2', save='loaded.state', watch='CE59-CE59')
if watches(outl):
    problems.append('C: DS:4959h was written during the load: %s' % watches(outl))
loaded4A01 = hexpeek(shotsl[-1], '2EA2:0202')
if loaded4A01 != b'\x01\x00' or hexpeek(shotsl[-1], '0850:6A0B')[:2] != b'\x05\x05':
    problems.append('C: after loading, want (5,5) with 4A01=1: %s' % shotsl[-1]['peek'])
OUT = ['Left', 'Left', 'Up', 'Right', 'Up', 'Left', 'Up', 'Return']
outa, shotsa = run(','.join(OUT), 'cases-after-load', '0850:6A0B:5,2EA2:0202:2', load='loaded.state', watch='2EC22-2EC23')
rowsa = frames(shotsa, OUT)
wa = watches(outa)
street = next((i for i, r in enumerate(rowsa) if (r['x'], r['y']) == (3, 4)), None)
if len(wa) != 1 or (wa[0]['old'], wa[0]['new'], wa[0]['ip']) != ('01', '00', '02FE') or key_at(rowsa, wa[0]['step']) != street:
    problems.append('C: want one 4A01 01→00 from :02FE stepping out to (3,4): %s %s' % (wa, rowsa))
else:
    # overlay 會被換出，所以段內容要在寫入的那一步讀，不能事後另開一次讀。
    key = '%s:02E0' % wa[0]['cs']
    _, seg = run(','.join(OUT), 'cases-after-load-seg', key + ':48', load='loaded.state')
    if hexpeek(seg[1 + street], key) != OVERLAY7[0x2E0:0x2E0 + 48]:
        problems.append('C: segment %s is not overlay-07 on the step out' % wa[0]['cs'])

side = {k: first['peek'][k] for k in ('2EA2:01CA', '2F22:05A4', '2F22:05C2')}
out = {'schema': 'pool-dosgolem-block-load-clear-cases/1', 'generator': 'dosgolem',
       'generator_revision': os.popen('git -C %s rev-parse --short HEAD' % DOSGOLEM).read().strip(),
       'original_exe_sha256': GOBLINS['original_exe_sha256'],
       'overlay07_sha256': 'a59f9d16a1d186bbd3806865ffe55de4be5b58484287237fb73872295da778ae',
       'walk_out_keys': PREFIX,
       'pointers': {'class0': '2EA2:0000 (DS:4933h)', 'class1': '2F22:0000 (DS:4937h)'},
       'entry3_side_writes_at_slums_state': {
           'note': '49E5=0／49E6=1（2EA2:01CA）、6DD2=0／6DD3=0（2F22:05A4）、6DE1=FF（2F22:05C2）；值已經等於 entry 3 要寫的，監看看不到變化',
           'peek': side},
       'cross_archive_class0': {'route': CROSS, 'watch_linear': '2EC20-2EC5F', 'writes': w0, 'frames': rows0},
       'cross_archive_class1': {'route': CROSS, 'watch_linear': '2F912-2F925', 'writes': w1, 'frames': rows1},
       'load': {'save_keys': 'e,s,J (from clerk.state)', 'load_keys': LOAD, 'watch_4959_linear': 'CE59', 'writes_4959': watches(outl),
                'after_load_peek': shotsl[-1]['peek'], 'walk_out': OUT, 'walk_out_writes': wa, 'walk_out_frames': rowsa},
       'verdict': 'ok' if not problems else 'failed', 'problems': problems}
json.dump(out, open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-block-load-clear-cases.json'), 'w'), ensure_ascii=False, indent=1)
print(out['verdict'], problems)
if problems:
    sys.exit(1)
