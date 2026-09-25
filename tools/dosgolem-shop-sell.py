#!/usr/bin/env python3
"""武具店賣出的原版收據（#60，spec 067〈賣出〉）：拿 dosgolem-shop-payment.py 同一個前置角色
（docs/audit/dos-thief-training-skills.json 那一趟的 HUMAN.cha，只改錢欄），走同一條路進武具店，
買一件、V I S Y 賣掉，讀記錄的五個錢欄與公款（DS:6752h 起五個 longint）在賣之前與之後的值，
並從賣之前那一幀的畫面記下原版的出價。

三個情境：
  partisan    白金 100 → 買 PARTISAN（10 金）再賣：出價 5 → 白金 +1
  long-sword  白金 100 → 買 LONG SWORD（15 金）再賣：出價 7 → 白金 +1 金 +2
  overload    白金 1500 → P 全部進公款、公款買 PLATE MAIL（400 金）、S 平分把角色塞到負重上限，
              再賣板甲：出價 200 → 40 白金全部進公款（overlay-21 entry 4 的可容納量是 0）

路徑可用環境變數改：POOL_DOSGOLEM（dosgolem 原始碼）、POOL_DOS_ORIG（原版檔案）、
POOL_SELL_WORK（可寫工作目錄）。輸出 docs/audit/dosgolem-shop-sell.json。"""
import hashlib, json, os, shutil, subprocess

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DOSGOLEM = os.environ.get('POOL_DOSGOLEM', os.path.join(ROOT, 'workplace', 'dosgolem'))
SOURCE = os.environ.get('POOL_DOS_ORIG', os.path.join(ROOT, 'workplace', 'oracle', 'dos'))
BASE = os.environ.get('POOL_SELL_BASE', os.path.join(ROOT, 'workplace', 'dosgolem-thief-training', 'route13-scratch'))
WORK = os.environ.get('POOL_SELL_WORK', os.path.join(ROOT, 'workplace', 'shop-sell'))
os.makedirs(WORK, exist_ok=True)
COINS = ['copper', 'silver', 'electrum', 'gold', 'platinum']
EXE_SHA256 = '12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f'
# 與 dosgolem-shop-payment.py 相同：建角那一段換成把現成的 HUMAN 加進隊伍，走到武具店按 y 進門。
APPROACH = ('rep:9:Space,Return,Return,a,a,e,b,rep:27:Return,Right,Right,Up,Return,Right,'
            'rep:7:Up,Left,rep:7:Up,y')
# 買的清單：b 開第一頁（游標在第 1 列），n 換頁（列號不變），End 往下一列。
SCENARIOS = {
    'partisan': {'money': {'platinum': 100}, 'item': 'Partisan',
                 'keys': 'b,n,rep:4:End,Return,Escape,v,i,s,y,Return,Escape'},
    'long-sword': {'money': {'platinum': 100}, 'item': 'Long Sword',
                   'keys': 'b,n,rep:15:End,Return,Escape,v,i,s,y,Return,Escape'},
    'overload': {'money': {'platinum': 1500}, 'item': 'Plate Mail',
                 'keys': 'p,b,n,n,rep:16:End,Return,Escape,s,v,i,s,y,Return,Escape'},
}

# 出價畫面（offer_frame）上印的數字，人工讀出。畫面是色號陣列，這裡不做 OCR；
# 對照的另一半是錢欄的前後差，兩者都寫進收據。
OFFER_ON_SCREEN = {'partisan': 5, 'long-sword': 7, 'overload': 200}


def run(keys, scratch, load, save, tag, peek):
    out = os.path.join(WORK, tag)
    shutil.rmtree(out, ignore_errors=True)
    os.makedirs(out)
    args = ['docker', 'run', '--rm', '--name', 'pool60-shop-sell-' + tag, '--network', 'none', '--memory', '4g',
            '--cpus', '4', '--pids-limit', '256', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
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


def wallet(record_hex):
    b = bytes.fromhex(record_hex)
    return {COINS[i]: int.from_bytes(b[0x88 + 2 * i:0x8a + 2 * i], 'little') for i in range(5)}


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
    strength = b[0x10]
    shots = run(APPROACH, scratch, None, name + '-shop.state', name + '-approach', 'ds:5CF0:4')
    seg, v = shots[-1]['peek']['ds:5CF0'].split('|')
    assert seg == '0850', seg
    off = int.from_bytes(bytes.fromhex(v)[0:2], 'little')
    rseg = int.from_bytes(bytes.fromhex(v)[2:4], 'little')
    key = '%04X:%04X' % (rseg, off)
    shots = run(spec['keys'], scratch, name + '-shop.state', None, name + '-sell',
                'ds:6752:20,%s:%d' % (key, 0x120))
    frames = []
    for s in shots:
        rec = bytes.fromhex(s['peek'][key].split('|')[1])
        frames.append({'label': s['label'], 'wallet': wallet(rec.hex()), 'pool': pool(s['peek']['ds:6752']),
                       'load': int.from_bytes(rec[0x102:0x104], 'little'), 'items': rec[0xC7],
                       'sha256_of_frame': s['sha256']})
    # 賣掉的那一步是最後一個 `y`；它前一幀是出價畫面（`I'll give you N gold pieces for your X`）。
    sell = max(i for i, f in enumerate(frames) if f['label'] == 'y')
    before, after = frames[sell - 1], frames[sell]
    print(name, before['wallet'], '->', after['wallet'], 'pool', before['pool']['platinum'], '->',
          after['pool']['platinum'], 'offer frame', os.path.join(WORK, name + '-sell'))
    return {'item': spec['item'], 'injected_money': spec['money'], 'injected_character_sha256': injected,
            'strength': strength, 'record': key, 'keys': spec['keys'],
            'offer_frame': '%02d-%s.png' % (sell - 1, frames[sell - 1]['label']),
            'offer': OFFER_ON_SCREEN.get(name),
            'wallet_before_sale': before['wallet'], 'wallet_after_sale': after['wallet'],
            'pool_platinum_before_sale': before['pool']['platinum'],
            'pool_platinum_after_sale': after['pool']['platinum'],
            'load_before_sale': before['load'], 'items_before_sale': before['items'],
            'items_after_sale': after['items'], 'frames': frames}


out = {'schema': 'pool-dosgolem-shop-sell/1', 'generator': 'dosgolem', 'original_exe_sha256': EXE_SHA256,
       'base': '同 dosgolem-shop-payment.json（HUMAN.cha 只改五個錢欄）', 'approach_keys': APPROACH,
       'offer_note': 'offer 由出價畫面人工讀出後填入；該幀的 sha256 在 frames 裡', 'scenarios': {}}
for name, spec in SCENARIOS.items():
    out['scenarios'][name] = scenario(name, spec)
out['generator_revision'] = os.popen('git -C %s rev-parse --short HEAD' % DOSGOLEM).read().strip()
target = os.path.join(ROOT, 'docs', 'audit', 'dosgolem-shop-sell.json')
json.dump(out, open(target, 'w'), ensure_ascii=False, indent=1)
print('寫出', target)
