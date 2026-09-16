#!/usr/bin/env python3
"""武具店付款的原版收據（#30，spec 116〈付款〉）：拿 docs/audit/dos-thief-training-skills.json
那一趟的前置角色（HUMAN，只改五個錢欄），走 workplace/dosgolem-ref-shop 那條路進武具店，
`b,Return` 買清單第一件，讀記錄的五個錢欄前後。

兩個情境：
  platinum  白金 100（= 500 金）      → 角色自己付、餘額重鑄成白金＋金（overlay-21 entry 15）
  mixed     銅 199 銀 19 金 30 白金 2 → 四捨五入的等值與重鑄

輸出 docs/audit/dosgolem-shop-payment.json。"""
import json, os, shutil, sys, hashlib, importlib.util
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
spec = importlib.util.spec_from_file_location('fee', os.path.join(ROOT, 'tools', 'dosgolem-training-fee.py'))
# 借它的 run／wallet／COINS，但不要跑它的主程式：只載入定義。
src = open(os.path.join(ROOT, 'tools', 'dosgolem-training-fee.py')).read().split("\nout = {'schema'")[0]
namespace = {'__name__': 'fee', '__file__': os.path.join(ROOT, 'tools', 'dosgolem-training-fee.py')}
exec(compile(src, 'dosgolem-training-fee.py', 'exec'), namespace)
run, wallet, COINS, BASE = namespace['run'], namespace['wallet'], namespace['COINS'], namespace['BASE']
WORK = os.path.join(ROOT, 'workplace', 'shop-payment'); os.makedirs(WORK, exist_ok=True)
namespace['WORK'] = WORK
SHOP = json.load(open(os.path.join(ROOT, 'workplace', 'dosgolem-ref-shop', 'provenance.json')))['keys'].split(',')
# 建角那一段換成把現成的 HUMAN 加進隊伍。
assert SHOP[:3] == ['rep:9:Space', 'Return', 'Return'] and 'a' in SHOP
first_a = SHOP.index('a')
APPROACH = ','.join(SHOP[:3] + SHOP[first_a:])

def scenario(name, money):
    scratch = os.path.join(WORK, name + '-scratch'); shutil.rmtree(scratch, ignore_errors=True); shutil.copytree(BASE, scratch)
    open(os.path.join(scratch, 'charlist.txt'), 'wb').write(b'HUMAN\r\n')
    path = os.path.join(scratch, 'HUMAN.cha'); b = bytearray(open(path, 'rb').read())
    for i, coin in enumerate(COINS):
        b[0x88 + 2 * i:0x8a + 2 * i] = money.get(coin, 0).to_bytes(2, 'little')
    open(path, 'wb').write(b)
    injected = hashlib.sha256(b).hexdigest()
    shots = run(APPROACH, scratch, None, name + '-shop.state', name + '-approach', 'ds:2D0:16,ds:5CF0:4')
    seg, v = shots[-1]['peek']['ds:5CF0'].split('|'); assert seg == '0850', seg
    off = int.from_bytes(bytes.fromhex(v)[0:2], 'little'); rseg = int.from_bytes(bytes.fromhex(v)[2:4], 'little')
    rec = '%04X:%04X:%d' % (rseg, off, 0x120)
    shots = run('b,Return,Escape', scratch, name + '-shop.state', None, name + '-buy', 'ds:2D0:16,ds:5CF0:4,' + rec)
    key = '%04X:%04X' % (rseg, off)
    frames = []
    for s in shots:
        h = s['peek'][key].split('|')[1]
        frames.append({'label': s['label'], 'wallet': wallet(h), 'sha256_of_frame': s['sha256']})
    print(name, [(f['label'], f['wallet']) for f in frames])
    return {'injected_money': money, 'injected_character_sha256': injected, 'record': key, 'frames': frames}

out = {'schema': 'pool-dosgolem-shop-payment/1', 'generator': 'dosgolem',
       'original_exe_sha256': '12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f',
       'base': '同 dosgolem-training-fee.json（HUMAN.cha 只改五個錢欄）', 'approach_keys': APPROACH, 'buy_keys': 'b,Return,Escape', 'scenarios': {}}
for name, money in (('platinum', {'platinum': 100}), ('mixed', {'copper': 199, 'silver': 19, 'gold': 30, 'platinum': 2})):
    out['scenarios'][name] = scenario(name, money)
out['generator_revision'] = os.popen('git -C %s rev-parse --short HEAD' % namespace['DOSGOLEM']).read().strip()
json.dump(out, open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-shop-payment.json'), 'w'), ensure_ascii=False, indent=1)
print('寫出 docs/audit/dosgolem-shop-payment.json')
