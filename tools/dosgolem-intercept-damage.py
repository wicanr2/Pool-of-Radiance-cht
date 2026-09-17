#!/usr/bin/env python3
"""原版扣血入口攔截的收據（#5，spec 084〈呼叫契約與所有呼叫端〉）。

原版所有扣血都經過 overlay-25 entry 28（`2266h`），呼叫端一律遠呼叫 resident stub `010Ah:00ACh`。
dosgolem `shots -intercept-damage` 在 stub 上讀 (目標 far pointer, 傷害 byte)。這一支從
`workplace/orc-drive/guards2.state`（貧民窟衛兵攔截，戰鬥剛開始）開快速戰鬥（每位隊員按 q），跑三趟：

  log       只記不改：陣營值從這一趟量出來；結局要是全滅（負對照）
  lock      party=0,lock：我方的傷害全部改成 0；結局要是敵方歸零、我方五人都在
  lock-kill party=0,lock,kill：同上，而且敵方被打到就歸零

判讀用 `0850:6772`（兩邊人數，第一個 byte 我方、第二個敵方）與最後一張畫面的底部列。
已經跑過的輸出（workplace/dosgolem-intercept/<趟>）直接沿用；要重跑就刪掉那個目錄。
輸出 docs/audit/dosgolem-intercept-damage.json。"""
import json, os, re, shutil, subprocess, sys
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DOSGOLEM = os.path.join(ROOT, 'workplace', 'dosgolem')
SOURCE = os.path.join(ROOT, 'workplace', 'oracle', 'dos')
WORK = os.path.join(ROOT, 'workplace', 'dosgolem-intercept')
STUB = 'stub=010A:00AC,hp=11B,side=10E,combat=4954:5'
RUNS = [
    ('log-q6', STUB, 'rep:8:q', 1500000000),
    ('lock-q', STUB + ',party=0,lock', 'rep:10:q', 4000000000),
    ('lockkill-q', STUB + ',party=0,lock,kill', 'rep:8:q', 1500000000),
]

def run(tag, hook, keys, budget):
    out = os.path.join(WORK, tag)
    if os.path.exists(os.path.join(out, 'shots.json')) and os.path.exists(os.path.join(WORK, tag + '.txt')):
        return
    shutil.rmtree(out, ignore_errors=True)
    scratch = os.path.join(WORK, 'scratch')
    shutil.rmtree(scratch, ignore_errors=True)
    shutil.copytree(os.path.join(ROOT, 'workplace', 'orc-drive', 'scratch'), scratch)
    shutil.copy(os.path.join(ROOT, 'workplace', 'orc-drive', 'guards2.state'), os.path.join(WORK, 'guards2.state'))
    args = ['docker', 'run', '--rm', '--network', 'none', '--memory', '4g', '--cpus', '2', '--pids-limit', '256',
            '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3', '-u', '%d:%d' % (os.getuid(), os.getgid()),
            '-v', DOSGOLEM + ':/dosgolem', '-v', SOURCE + ':/orig:ro', '-v', WORK + ':/work', '-v', scratch + ':/scratch',
            '-v', os.path.join(DOSGOLEM, 'workplace', 'gocache') + ':/gocache',
            '-v', os.path.join(DOSGOLEM, 'workplace', 'gomodcache') + ':/gomodcache',
            '-e', 'GOCACHE=/gocache', '-e', 'GOMODCACHE=/gomodcache', '-e', 'HOME=/tmp', '-e', 'GOFLAGS=-mod=mod',
            '-w', '/dosgolem', 'golang:1.24-bookworm', 'go', 'run', './cmd/shots', '-exe', '/orig/start.exe', '-root', '/orig',
            '-scratch', '/scratch', '-out', '/work/' + tag, '-budget', str(budget), '-idle', '60000000',
            '-peek', '0850:6772:2,0850:4954:1', '-intercept-damage', hook, '-keys', keys, '-load-state', '/work/guards2.state']
    p = subprocess.run(args, capture_output=True, text=True, timeout=7200)
    open(os.path.join(WORK, tag + '.txt'), 'w').write(p.stdout + p.stderr)
    if p.returncode != 0:
        raise SystemExit('shots failed: ' + tag)

os.makedirs(WORK, exist_ok=True)
problems, results = [], {}
for tag, hook, keys, budget in RUNS:
    run(tag, hook, keys, budget)
    shots = json.load(open(os.path.join(WORK, tag, 'shots.json')))
    damage = [d for s in shots for d in (s.get('damage') or [])]
    counts = None
    for s in shots:
        value = s['peek'].get('0850:6772') or s['peek'].get('ds:6772', '')
        if value.startswith('0850|'):
            counts = value.split('|')[1]
    callers = sorted({'%04X:%04X' % (d['caller_cs'], d['caller_ip']) for d in damage})
    results[tag] = {'hook': hook, 'keys': keys, 'calls': len(damage), 'callers': callers,
                    'sides': sorted({d['side'] for d in damage}),
                    'party_calls': sum(1 for d in damage if d['side'] == 0),
                    'party_changed_to_zero': sum(1 for d in damage if d['side'] == 0 and d['changed'] == 0 and d['damage'] > 0),
                    'foe_calls': sum(1 for d in damage if d['side'] == 1),
                    'foe_set_to_hp': sum(1 for d in damage if d['side'] == 1 and d['changed'] == d['hp'] and d['damage'] < d['hp']),
                    'final_counts_party_foe': counts, 'last_frame': shots[-1]['path'], 'last_frame_sha256': shots[-1]['sha256']}

log, lock, kill = results['log-q6'], results['lock-q'], results['lockkill-q']
if log['calls'] == 0 or log['sides'] != [0, 1]:
    problems.append('log: 沒攔到兩邊的扣血 %s' % log)
if log['party_changed_to_zero'] != 0 or log['foe_set_to_hp'] != 0:
    problems.append('log: 只記錄那一趟卻改了傷害')
for name, r in (('lock', lock), ('lock-kill', kill)):
    if r['party_calls'] == 0 or r['party_changed_to_zero'] != r['party_calls']:
        problems.append('%s: 我方 %d 次扣血只有 %d 次改成 0' % (name, r['party_calls'], r['party_changed_to_zero']))
    if r['final_counts_party_foe'] != '0500':
        problems.append('%s: 結局人數 %s，要 0500（我方 5、敵方 0）' % (name, r['final_counts_party_foe']))
if kill['foe_calls'] == 0 or kill['foe_set_to_hp'] == 0:
    problems.append('lock-kill: 沒有敵方扣血被改成目標 HP：%s' % kill)
if lock['foe_set_to_hp'] != 0:
    problems.append('lock: 沒開 kill 卻有敵方扣血被改')
out = {'schema': 'pool-dosgolem-intercept-damage/1', 'generator': 'dosgolem',
       'generator_revision': os.popen('git -C %s rev-parse --short HEAD' % DOSGOLEM).read().strip(),
       'start_state': 'workplace/orc-drive/guards2.state（貧民窟衛兵攔截，戰鬥剛開始）',
       'stub': '010A:00AC（overlay-25 entry 28 `2266h`）', 'party_side_field': '+10Eh：我方 0、敵方 1',
       'runs': results,
       'negative_control_note': 'log 那一趟最後一張是空的隊伍名單與 PRESS ANY KEY TO CONTINUE（全滅）',
       'verdict': 'ok' if not problems else 'failed', 'problems': problems}
json.dump(out, open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-intercept-damage.json'), 'w'), ensure_ascii=False, indent=1)
print(out['verdict'], problems)
for k, v in results.items():
    print(k, {x: v[x] for x in ('calls', 'party_calls', 'party_changed_to_zero', 'foe_calls', 'foe_set_to_hp', 'final_counts_party_foe', 'callers')})
sys.exit(1 if problems else 0)
