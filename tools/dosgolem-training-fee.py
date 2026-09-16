#!/usr/bin/env python3
"""訓練所收費的原版收據（#29，spec 097〈收費與門〉）：拿 docs/audit/dos-thief-training-skills.json
那一趟的前置狀態（workplace/dosgolem-thief-training/route13-scratch：HUMAN.cha 注入 DEX 18、XP 1251、
盜賊 1 級）改錢欄，從開機走同一條鍵序到盜賊門前，`t` 之後讀記錄的五個錢欄與職業等級。

三個情境：
  mixed   銅 50 銀 30 金 1003 白金 2 → 過門，`42C1h` 一種一種扣、找零
  short   金 500                      → "Training costs 1000 gp."，不升不扣
  exact   金 1000                     → 剛好付完

輸出 docs/audit/dosgolem-training-fee.json。"""
import json, os, shutil, subprocess, sys, hashlib
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DOSGOLEM = os.path.join(ROOT, 'workplace', 'dosgolem')
SOURCE = os.path.join(ROOT, 'workplace', 'oracle', 'dos')
BASE = os.path.join(ROOT, 'workplace', 'dosgolem-thief-training', 'route13-scratch')
WORK = os.path.join(ROOT, 'workplace', 'training-fee')
os.makedirs(WORK, exist_ok=True)
RECEIPT = json.load(open(os.path.join(ROOT, 'docs', 'audit', 'dos-thief-training-skills.json')))
KEYS = RECEIPT['dynamic_evidence']['keys'].split(',')
assert KEYS[-4:] == ['t', 'y', 's', 'a'], KEYS[-4:]
APPROACH = ','.join(KEYS[:-4])  # 走到盜賊門前、選好人
COINS = ['copper', 'silver', 'electrum', 'gold', 'platinum']

def run(keys, scratch, load, save, tag, peek):
    out = os.path.join(WORK, tag); shutil.rmtree(out, ignore_errors=True); os.makedirs(out)
    args = ['docker', 'run', '--rm', '--network', 'none', '--memory', '4g', '--cpus', '2', '--pids-limit', '256',
            '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3', '-u', '%d:%d' % (os.getuid(), os.getgid()),
            '-v', DOSGOLEM + ':/dosgolem', '-v', SOURCE + ':/orig:ro', '-v', WORK + ':/work', '-v', scratch + ':/scratch',
            '-v', os.path.join(DOSGOLEM, 'workplace', 'gocache') + ':/gocache', '-v', os.path.join(DOSGOLEM, 'workplace', 'gomodcache') + ':/gomodcache',
            '-e', 'GOCACHE=/gocache', '-e', 'GOMODCACHE=/gomodcache', '-e', 'HOME=/tmp', '-e', 'GOFLAGS=-mod=mod',
            '-w', '/dosgolem', 'golang:1.24-bookworm', 'go', 'run', './cmd/shots', '-exe', '/orig/start.exe', '-root', '/orig',
            '-scratch', '/scratch', '-out', '/work/' + tag, '-budget', '200000000', '-idle', '40000000', '-peek', peek, '-keys', keys]
    if load: args += ['-load-state', '/work/' + load]
    if save: args += ['-save-state', '/work/' + save]
    p = subprocess.run(args, capture_output=True, text=True, timeout=3600)
    open(os.path.join(out, 'stdout.txt'), 'w').write(p.stdout + p.stderr)
    if p.returncode != 0:
        print(p.stdout[-3000:], p.stderr[-2000:]); raise SystemExit('shots failed: ' + tag)
    return json.load(open(os.path.join(out, 'shots.json')))

def wallet(record_hex):
    b = bytes.fromhex(record_hex)
    return {COINS[i]: int.from_bytes(b[0x88 + 2 * i:0x8a + 2 * i], 'little') for i in range(5)}

def scenario(name, money):
    scratch = os.path.join(WORK, name + '-scratch'); shutil.rmtree(scratch, ignore_errors=True); shutil.copytree(BASE, scratch)
    # 加進隊伍時原版把 charlist.txt 裡的名字拿掉了，所以那一份是空的；補回來才加得到人。
    open(os.path.join(scratch, 'charlist.txt'), 'wb').write(b'HUMAN\r\n')
    path = os.path.join(scratch, 'HUMAN.cha'); b = bytearray(open(path, 'rb').read())
    for i, coin in enumerate(COINS):
        b[0x88 + 2 * i:0x8a + 2 * i] = money.get(coin, 0).to_bytes(2, 'little')
    open(path, 'wb').write(b)
    injected = hashlib.sha256(b).hexdigest()
    # 走到門前，記狀態；這一幀讀目前角色的遠指標 DS:5CF0h。
    shots = run(APPROACH, scratch, None, name + '-door.state', name + '-approach', 'ds:2D0:16,ds:5CF0:4')
    seg, v = shots[-1]['peek']['ds:5CF0'].split('|'); assert seg == '0850', seg
    off = int.from_bytes(bytes.fromhex(v)[0:2], 'little'); rseg = int.from_bytes(bytes.fromhex(v)[2:4], 'little')
    rec = '%04X:%04X:%d' % (rseg, off, 0x120)
    # t，看它問不問；再 y。
    shots = run('t,y', scratch, name + '-door.state', None, name + '-train', 'ds:2D0:16,ds:5CF0:4,' + rec)
    key = '%04X:%04X' % (rseg, off)
    frames = []
    for s in shots:
        h = s['peek'][key].split('|')[1]
        b2 = bytes.fromhex(h)
        frames.append({'label': s['label'], 'wallet': wallet(h), 'thief_level': b2[0x96 + 6], 'experience': int.from_bytes(b2[0xAC:0xB0], 'little'), 'sha256_of_frame': s['sha256']})
    print(name, [(f['label'], f['wallet'], f['thief_level']) for f in frames])
    return {'injected_money': money, 'injected_character_sha256': injected, 'record': key, 'frames': frames}

out = {'schema': 'pool-dosgolem-training-fee/1', 'generator': 'dosgolem', 'original_exe_sha256': RECEIPT['dynamic_evidence']['executable_sha256'],
       'base': 'docs/audit/dos-thief-training-skills.json 的前置狀態（route13-scratch 的 HUMAN.cha：DEX 18、XP 1251、盜賊 1 級），只改 +88h.. 五個錢欄',
       'approach_keys': APPROACH, 'train_keys': 't,y', 'scenarios': {}}
for name, money in (('mixed', {'copper': 50, 'silver': 30, 'gold': 1003, 'platinum': 2}), ('short', {'gold': 500}), ('exact', {'gold': 1000})):
    out['scenarios'][name] = scenario(name, money)
out['generator_revision'] = os.popen('git -C %s rev-parse --short HEAD' % DOSGOLEM).read().strip()
json.dump(out, open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-training-fee.json'), 'w'), ensure_ascii=False, indent=1)
print('寫出 docs/audit/dosgolem-training-fee.json')
