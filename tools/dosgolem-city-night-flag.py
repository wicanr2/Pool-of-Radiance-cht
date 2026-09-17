#!/usr/bin/env python3
"""城區入口 0 的「晚上」判斷對原版（#22／#26，spec 102〈晚上鎖門〉）。

`ecl3/0 9920h` 是 `COMPARE @49C9, 14 ; IF >= ; SAVE 8 → 6E7D`（前一條先寫 11），
然後 `992Dh` 把 `6E7D` 抄進 `49FD`、`9934h` 寫 `49FE = 10`。比較的方向靜態讀不出來：
engine 讀成「49C9 ≥ 14」，反過來讀是「14 ≥ 49C9」。開局時鐘是 0，兩種讀法在城區走一步
之後寫進 `6E7D` 的值不同（11 對 8），所以在原版走一步讀出來就能判定。

走法：沿用 docs/audit/dosgolem-deployment-peek-goblins.json 建隊與導覽的鍵序，導覽結束後
朝西三步（第一步就出城進貧民窟），存狀態；再從狀態轉身走回城區兩步，每一步讀
- class 0 記錄 `[DS:4933h]`（far pointer）＋ `(6E00h + addr × 2) mod 10000h`：49C6..49CC、49FD、49FE
- class 1 記錄 `[DS:4937h]` ＋ `(2A00h + addr × 2) mod 10000h`：6E7B..6E7F
- `DS:6A0Bh` 的 X、Y、朝向、牆、地形（spec 015）

正對照：`49FE` 由貧民窟與城區各自的入口 0 寫（城區是 10），進城那一步必須從別的值變成 10；
沒變就是位址換算錯，這份收據作廢。

輸出 docs/audit/dosgolem-city-night-flag.json。"""
import json, os, subprocess, sys
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DOSGOLEM = os.path.join(ROOT, 'workplace', 'dosgolem')
SOURCE = os.path.join(ROOT, 'workplace', 'oracle', 'dos')
WORK = os.path.join(ROOT, 'workplace', 'city-night-flag')
SCRATCH = os.path.join(WORK, 'scratch')
os.makedirs(SCRATCH, exist_ok=True)
GOBLINS = json.load(open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-deployment-peek-goblins.json')))
TAIL = ',e,b,rep:14:Return,Up,Up,Up'
PREFIX = GOBLINS['keys'][:GOBLINS['keys'].index(TAIL) + len(TAIL)]
BACK = 'Right,Right,Up,Up,Up,Up'

def run(keys, tag, peek, load=None, save=None):
    args = ['docker', 'run', '--rm', '--network', 'none', '--memory', '4g', '--cpus', '2', '--pids-limit', '256',
            '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3', '-u', '%d:%d' % (os.getuid(), os.getgid()),
            '-v', DOSGOLEM + ':/dosgolem', '-v', SOURCE + ':/orig:ro', '-v', WORK + ':/work', '-v', SCRATCH + ':/scratch',
            '-v', os.path.join(DOSGOLEM, 'workplace', 'gocache') + ':/gocache', '-v', os.path.join(DOSGOLEM, 'workplace', 'gomodcache') + ':/gomodcache',
            '-e', 'GOCACHE=/gocache', '-e', 'GOMODCACHE=/gomodcache', '-e', 'HOME=/tmp', '-e', 'GOFLAGS=-mod=mod',
            '-w', '/dosgolem', 'golang:1.24-bookworm', 'go', 'run', './cmd/shots', '-exe', '/orig/start.exe', '-root', '/orig',
            '-scratch', '/scratch', '-out', '/work/' + tag, '-budget', '200000000', '-idle', '40000000', '-peek', peek, '-keys', keys]
    if load: args += ['-load-state', '/work/' + load]
    if save: args += ['-save-state', '/work/' + save]
    p = subprocess.run(args, capture_output=True, text=True, timeout=3600)
    open(os.path.join(WORK, tag + '.stdout.txt'), 'w').write(p.stdout + p.stderr)
    if p.returncode != 0 or '看不懂' in p.stdout:
        print(p.stdout[-3000:], p.stderr[-2000:]); raise SystemExit('shots failed: ' + tag)
    return json.load(open(os.path.join(WORK, tag, 'shots.json')))

def words(hexs):
    b = bytes.fromhex(hexs)
    return [int.from_bytes(b[i:i + 2], 'little') for i in range(0, len(b), 2)]

first = run(PREFIX, 'walk-out', 'ds:4933:8,ds:6A0B:5', save='slums.state')
seg0, p0 = first[-1]['peek']['ds:4933'].split('|')
ptr = bytes.fromhex(p0)
off0, s0 = int.from_bytes(ptr[0:2], 'little'), int.from_bytes(ptr[2:4], 'little')
off1, s1 = int.from_bytes(ptr[4:6], 'little'), int.from_bytes(ptr[6:8], 'little')
c0 = lambda addr: (off0 + 0x6E00 + addr * 2) & 0xFFFF
c1 = lambda addr: (off1 + 0x2A00 + addr * 2) & 0xFFFF
clock = '%04X:%04X:14' % (s0, c0(0x49C6))
flag49 = '%04X:%04X:4' % (s0, c0(0x49FD))
temp = '%04X:%04X:10' % (s1, c1(0x6E7B))
second = run(BACK, 'walk-back', ','.join(['ds:6A0B:5', clock, flag49, temp]), load='slums.state')

frames = []
for shot in second:
    pos = bytes.fromhex(shot['peek']['ds:6A0B'].split('|')[1])
    clk = words(shot['peek'][clock[:-3]].split('|')[1])
    fl = words(shot['peek'][flag49[:-2]].split('|')[1])
    tv = words(shot['peek'][temp[:-3]].split('|')[1])
    frames.append({'label': shot['label'], 'x': pos[0], 'y': pos[1], 'facing_0_7': pos[2], 'wall_C04E': pos[3], 'terrain_C04F': pos[4],
                   'clock_49C6_49CC': clk, 'hour_49C9': clk[3], '49FD': fl[0], '49FE': fl[1], '6E7B_6E7F': tv, '6E7D': tv[2],
                   'frame_sha256': shot['sha256']})
entered = [i for i in range(1, len(frames)) if frames[i - 1]['49FE'] != 10 and frames[i]['49FE'] == 10]
if not entered:
    raise SystemExit('正對照失敗：49FE 沒有在進城那一步變成 10，位址換算不對')
city = frames[entered[0]]
engine_reading = 8 if city['hour_49C9'] >= 14 else 11
reversed_reading = 8 if 14 >= city['hour_49C9'] else 11
verdict = ('engine 讀法（49C9 ≥ 14）' if city['6E7D'] == engine_reading and city['6E7D'] != reversed_reading
           else '反向讀法（14 ≥ 49C9）' if city['6E7D'] == reversed_reading and city['6E7D'] != engine_reading
           else '分不出來')
out = {'schema': 'pool-dosgolem-city-night-flag/1', 'generator': 'dosgolem',
       'generator_revision': os.popen('git -C %s rev-parse --short HEAD' % DOSGOLEM).read().strip(),
       'original_exe_sha256': GOBLINS['original_exe_sha256'],
       'walk_out_keys': PREFIX, 'walk_back_keys': BACK,
       'pointers': {'class0_DS_4933': '%04X:%04X' % (s0, off0), 'class1_DS_4937': '%04X:%04X' % (s1, off1)},
       'peek': {'clock': clock, '49FD_49FE': flag49, '6E7B_6E7F': temp},
       'positive_control': '49FE 在 %s 那一步從 %d 變成 10（城區入口 0 `9934h`）' % (city['label'], frames[entered[0] - 1]['49FE']),
       'city_step': city, 'engine_reading_expects_6E7D': engine_reading, 'reversed_reading_expects_6E7D': reversed_reading,
       'verdict': verdict, 'frames': frames}
json.dump(out, open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-city-night-flag.json'), 'w'), ensure_ascii=False, indent=1)
print(verdict, city)
