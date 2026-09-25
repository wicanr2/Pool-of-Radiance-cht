#!/usr/bin/env python3
"""武具店的公款與鑑定原版收據（#67／#68，spec 067〈公款〉〈鑑定〉）：沿用
dosgolem-shop-sell.py 的前置角色與進店鍵序，讀記錄的五個錢欄、公款（DS:6752h 起五個
longint）與物品串列第一件的 `+35h`，在每一次按鍵之後各記一次。

情境：
  identify-paid   白金 100 → 買 PARTISAN（10 金）→ V I I Y：自己付 200 金
  identify-short  白金 4   → 買 PARTISAN → V I I Y：角色 10 金、公款 0 → Not Enough Money
  identify-pool   白金 100 → P 進公款 → 公款買 PARTISAN → V I I Y：公款付 200 金
                  → E（離開人物頁）→ E 離店：公款有錢，被問 → N 離開 → 往西退一格再走回來進店
  leave-yes       白金 100 → P 進公款 → E → Y 回到店裡 → S 平分 → E 直接離開

POOL_IDENTIFY_ONLY=名稱,名稱 只重跑那幾個情境，其餘沿用既有收據。

路徑可用環境變數改：POOL_DOSGOLEM、POOL_DOS_ORIG、POOL_SELL_BASE、POOL_IDENTIFY_WORK。
輸出 docs/audit/dosgolem-shop-pool-identify.json。"""
import hashlib, json, os, shutil, subprocess

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DOSGOLEM = os.environ.get('POOL_DOSGOLEM', os.path.join(ROOT, 'workplace', 'dosgolem'))
SOURCE = os.environ.get('POOL_DOS_ORIG', os.path.join(ROOT, 'workplace', 'oracle', 'dos'))
BASE = os.environ.get('POOL_SELL_BASE', os.path.join(ROOT, 'workplace', 'dosgolem-thief-training', 'route13-scratch'))
WORK = os.environ.get('POOL_IDENTIFY_WORK', os.path.join(ROOT, 'workplace', 'shop-pool-identify'))
os.makedirs(WORK, exist_ok=True)
COINS = ['copper', 'silver', 'electrum', 'gold', 'platinum']
EXE_SHA256 = '12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f'
APPROACH = ('rep:9:Space,Return,Return,a,a,e,b,rep:27:Return,Right,Right,Up,Return,Right,'
            'rep:7:Up,Left,rep:7:Up,y')
BUY_PARTISAN = 'b,n,rep:4:End,Return,Escape'
SCENARIOS = {
    'identify-paid': {'money': {'platinum': 100}, 'keys': BUY_PARTISAN + ',v,i,i,y,Return,Escape'},
    'identify-short': {'money': {'platinum': 4}, 'keys': BUY_PARTISAN + ',v,i,i,y,Return,Escape'},
    'identify-pool': {'money': {'platinum': 100},
                      'keys': 'p,' + BUY_PARTISAN + ',v,i,i,y,Return,Escape,e,e,n,Left,Left,Up,Left,Left,Up,y'},
    'leave-yes': {'money': {'platinum': 100}, 'keys': 'p,s,p,e,y,s,e,Return'},
}


def run(keys, scratch, load, save, tag, peek):
    out = os.path.join(WORK, tag)
    shutil.rmtree(out, ignore_errors=True)
    os.makedirs(out)
    args = ['docker', 'run', '--rm', '--name', 'pool67-shop-' + tag, '--network', 'none', '--memory', '4g',
            '--cpus', '3', '--pids-limit', '256', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
            '-u', '%d:%d' % (os.getuid(), os.getgid()),
            '-v', DOSGOLEM + ':/dosgolem', '-v', SOURCE + ':/orig:ro', '-v', WORK + ':/work', '-v', scratch + ':/scratch',
            '-v', os.path.join(DOSGOLEM, 'workplace', 'gocache') + ':/gocache',
            '-v', os.path.join(DOSGOLEM, 'workplace', 'gomodcache') + ':/gomodcache',
            '-e', 'GOCACHE=/gocache', '-e', 'GOMODCACHE=/gomodcache', '-e', 'HOME=/tmp', '-e', 'GOFLAGS=-mod=mod',
            '-w', '/dosgolem', 'golang:1.24-bookworm', 'go', 'run', './cmd/shots', '-exe', '/orig/start.exe',
            '-root', '/orig', '-scratch', '/scratch', '-out', '/work/' + tag, '-budget', '200000000',
            '-idle', '40000000', '-peek', peek, '-keys', keys]
    if load:
        args += ['-load-state', '/work/' + load]
    if save:
        args += ['-save-state', '/work/' + save]
    p = subprocess.run(args, capture_output=True, text=True, timeout=3600)
    open(os.path.join(out, 'stdout.txt'), 'w').write(p.stdout + p.stderr)
    if p.returncode != 0:
        print(p.stdout[-3000:], p.stderr[-2000:])
        raise SystemExit('shots failed: ' + tag)
    return json.load(open(os.path.join(out, 'shots.json')))


def wallet(rec):
    return {COINS[i]: int.from_bytes(rec[0x88 + 2 * i:0x8a + 2 * i], 'little') for i in range(5)}


def pool(peek_value):
    b = bytes.fromhex(peek_value.split('|')[1])
    return {COINS[i]: int.from_bytes(b[4 * i:4 * i + 4], 'little') for i in range(5)}


def scenario(name, spec):
    scratch = os.path.join(WORK, name + '-scratch')
    shutil.rmtree(scratch, ignore_errors=True)
    shutil.copytree(BASE, scratch)
    open(os.path.join(scratch, 'charlist.txt'), 'wb').write(b'HUMAN\r\n')
    path = os.path.join(scratch, 'HUMAN.cha')
    b = bytearray(open(path, 'rb').read())
    for i, coin in enumerate(COINS):
        b[0x88 + 2 * i:0x8a + 2 * i] = spec['money'].get(coin, 0).to_bytes(2, 'little')
    open(path, 'wb').write(b)
    injected = hashlib.sha256(b).hexdigest()
    shots = run(APPROACH, scratch, None, name + '-shop.state', name + '-approach', 'ds:5CF0:4')
    seg, v = shots[-1]['peek']['ds:5CF0'].split('|')
    assert seg == '0850', seg
    off = int.from_bytes(bytes.fromhex(v)[0:2], 'little')
    rseg = int.from_bytes(bytes.fromhex(v)[2:4], 'little')
    key = '%04X:%04X' % (rseg, off)
    shots = run(spec['keys'], scratch, name + '-shop.state', None, name + '-run',
                'ds:6752:20,%s:%d' % (key, 0x120))
    frames = []
    for s in shots:
        rec = bytes.fromhex(s['peek'][key].split('|')[1])
        frames.append({'label': s['label'], 'wallet': wallet(rec), 'pool': pool(s['peek']['ds:6752']),
                       'items': rec[0xC7], 'first_item': rec[0xC8:0xCC].hex(),
                       'sha256_of_frame': s['sha256']})
    first_item = frames[-1]['first_item']
    return {'injected_money': spec['money'], 'injected_character_sha256': injected, 'record': key,
            'keys': spec['keys'], 'frames_dir': os.path.relpath(os.path.join(WORK, name + '-run'), ROOT),
            'frames': frames, 'first_item_pointer_at_end': first_item}


def main():
    sha = hashlib.sha256(open(os.path.join(SOURCE, 'start.exe'), 'rb').read()).hexdigest()
    assert sha == EXE_SHA256, sha
    head = subprocess.run(['git', '-C', DOSGOLEM, 'rev-parse', 'HEAD'], capture_output=True, text=True).stdout.strip()
    out = os.path.join(ROOT, 'docs', 'audit', 'dosgolem-shop-pool-identify.json')
    only = [n for n in os.environ.get('POOL_IDENTIFY_ONLY', '').split(',') if n]
    previous = json.load(open(out))['scenarios'] if only and os.path.exists(out) else {}
    receipt = {'generator': 'dosgolem cmd/shots', 'dosgolem_commit': head, 'start_exe_sha256': sha,
               'approach_keys': APPROACH, 'scenarios': previous}
    for name, spec in SCENARIOS.items():
        if only and name not in only:
            continue
        receipt['scenarios'][name] = scenario(name, spec)
        for f in receipt['scenarios'][name]['frames']:
            print(name, f['label'], f['wallet'], 'pool', f['pool'], 'items', f['items'])
    json.dump(receipt, open(out, 'w'), indent=1, ensure_ascii=False)
    print('wrote', out)


if __name__ == '__main__':
    main()
