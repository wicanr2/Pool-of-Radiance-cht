#!/usr/bin/env python3
"""武具店 P 之後直接 S 的原版收據（#79，spec 067〈P 之後直接按 S〉）。

沿用 dosgolem-shop-pool-identify.py 的前置角色、注入錢包與進店鍵序，只把公款讀滿七欄
（DS:6752h 起 28 bytes）、錢包讀滿七欄，另外記現重 `+102h` 與 `+84h`。鍵序 `p,s,p`。

注入的白金沒有算進 `+102h`（現重 50 只是物品），所以 P 之後現重繞回 65486，
S 走 overlay-21 entry 7 的容量 helper 時「上限 − 現重」以 16 位元繞回——收據要看到的是
珠寶那一欄，不是白金。

路徑變數與 dosgolem-shop-pool-identify.py 相同；container 名稱加 POOL_CONTAINER_PREFIX（預設 `pool79-`）。
輸出 docs/audit/dosgolem-shop-share-wrap.json。"""
import importlib.util
import json
import os
import subprocess

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
spec = importlib.util.spec_from_file_location('identify', os.path.join(ROOT, 'tools', 'dosgolem-shop-pool-identify.py'))
identify = importlib.util.module_from_spec(spec)
spec.loader.exec_module(identify)
identify.COINS = ['copper', 'silver', 'electrum', 'gold', 'platinum', 'gems', 'jewelry']
KEYS = 'p,s,p'


def wallet(rec):
    value = {identify.COINS[i]: int.from_bytes(rec[0x88 + 2 * i:0x8a + 2 * i], 'little') for i in range(7)}
    value['load'] = int.from_bytes(rec[0x102:0x104], 'little')
    value['control'] = rec[0x84]
    return value


def pool(peek_value):
    raw = bytes.fromhex(peek_value.split('|')[1])
    return {identify.COINS[i]: int.from_bytes(raw[4 * i:4 * i + 4], 'little') for i in range(7)}


_run = subprocess.run


def run(args, **kwargs):
    if args and args[0] == 'docker':
        args = [a.replace('ds:6752:20,', 'ds:6752:28,') for a in args]
        index = args.index('--name')
        args[index + 1] = os.environ.get('POOL_CONTAINER_PREFIX', 'pool79-') + args[index + 1]
    return _run(args, **kwargs)


identify.subprocess = type('Subprocess', (), {'run': staticmethod(run)})
identify.wallet = wallet
identify.pool = pool


def main():
    sha = identify.hashlib.sha256(open(os.path.join(identify.SOURCE, 'start.exe'), 'rb').read()).hexdigest()
    assert sha == identify.EXE_SHA256, sha
    head = _run(['git', '-C', identify.DOSGOLEM, 'rev-parse', 'HEAD'], capture_output=True, text=True).stdout.strip()
    result = identify.scenario('share-wrap', {'money': {'platinum': 100}, 'keys': KEYS})
    receipt = {'generator': 'dosgolem cmd/shots', 'dosgolem_commit': head, 'start_exe_sha256': sha,
               'approach_keys': identify.APPROACH, 'keys': KEYS,
               'injected_money': result['injected_money'],
               'injected_character_sha256': result['injected_character_sha256'],
               'frames': [{'label': f['label'], 'wallet': f['wallet'], 'pool': f['pool']} for f in result['frames']]}
    out = os.path.join(ROOT, 'docs', 'audit', 'dosgolem-shop-share-wrap.json')
    json.dump(receipt, open(out, 'w'), indent=1, ensure_ascii=False)
    for frame in receipt['frames']:
        print(frame['label'], frame['wallet'], 'pool', frame['pool'])
    print('wrote', out)


if __name__ == '__main__':
    main()
