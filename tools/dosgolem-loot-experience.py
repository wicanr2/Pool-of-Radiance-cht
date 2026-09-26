#!/usr/bin/env python3
"""原版戰後結算把戰利品折成經驗值的收據（#94，spec 148）。

比兩個 dosgolem 狀態檔：交件前（`slums.state`，cheat 通關的 slums 段收尾）與職員發完獎金、
停在戰利品頂層選單的那一刻（`handin-slums-stuck.state`，spec 040／#47 的畫面基準用的同一份）。
兩者之間只有走路、睡覺與交件，沒有戰鬥（`run-handin-slums.out` 的檢查點）；職員的
`TREASURE → COMBAT` 走 overlay-05 entry 1 `14CAh` → `04ADh` → entry 2（`0000h`）→ entry 3
（`033Ah`），選單（`0E85h`）開的時候經驗值已經發完。所以兩份之間的經驗值差就是 entry 2 的
`0224h..0306h` 把公款折出來的那一份，除以分的人數、再依職業調整。

只讀：狀態檔、原版目錄、字形表全部唯讀掛載；dosgolem 的暫存層用一份複本。
位址：隊伍鏈 `DS:5CF4h`（下一個 `+104h`）、經驗值 `+0ACh`（dword）、職業碼 `+2Fh`、能力值 `+10h..`、
公款 `DS:6752h + 4*i`、戰利品串列 `DS:676Eh`（下一件 `+2Ah`）、`DS:5CF8h`、`DS:829Bh`。

用法：python3 tools/dosgolem-loot-experience.py <dosgolem-cheat 目錄> <暫存層複本> > docs/audit/dosgolem-loot-experience.json
（主機端會自己起容器；JSON 印在標準輸出）"""
import hashlib, json, os, subprocess, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if not os.path.exists('/bin/shots'):
    main = os.environ.get('POOL_MAIN_REPO', ROOT)
    work, scratch = sys.argv[1], sys.argv[2]
    for path in (work, scratch, os.path.join(main, 'workplace', 'oracle', 'dos'),
                 os.path.join(main, 'workplace', 'dosgolem', 'workplace', 'bin', 'shots')):
        if not os.path.exists(path):
            sys.exit('missing %s' % path)
    sys.exit(subprocess.call(
        ['timeout', '600', 'docker', 'run', '--rm', '-i', '--name', 'pool94-loot-xp', '--network', 'none',
         '--memory', '2g', '--cpus', '3', '--pids-limit', '128', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
         '-u', '%d:%d' % (os.getuid(), os.getgid()),
         '-v', os.path.join(main, 'workplace', 'dosgolem', 'workplace', 'bin', 'shots') + ':/bin/shots:ro',
         '-v', os.path.join(main, 'workplace', 'oracle', 'dos') + ':/orig:ro',
         '-v', os.path.join(ROOT, 'tools') + ':/tools:ro',
         '-v', work + ':/work:ro', '-v', scratch + ':/scratch',
         '-e', 'PYTHONUNBUFFERED=1', '-e', 'PYTHONPATH=/tools', '-w', '/scratch',
         'python:3.13-alpine', 'python3', '/tools/dosgolem-loot-experience.py']))

sys.path.insert(0, '/tools')
import dosgolem_drive as D

CURRENCIES = ['copper', 'silver', 'electrum', 'gold', 'platinum', 'gems', 'jewelry']


def dword(b, o):
    return b[o] | b[o + 1] << 8 | b[o + 2] << 16 | b[o + 3] << 24


def snapshot(name):
    path = os.path.join('/work', name)
    d = D.Dos(load=path, intercept=False, log=sys.stderr)
    seg, off = d.far(0x5CF4)
    party = []
    for _ in range(8):
        if (seg, off) in ((0, 0), (0xFFFF, 0xFFFF)):
            break
        r = d.mem(seg, off, 0x11D)
        party.append({'name': r[1:1 + r[0]].decode('ascii', 'replace') if r[0] < 16 else '',
                      'experience_0AC': dword(r, 0xAC), 'class_2F': r[0x2F],
                      'abilities_10': list(r[0x10:0x16]), 'npc_84': r[0x84], 'share_85': r[0x85],
                      'status_10C': r[0x10C], 'field_10D': r[0x10D], 'side_10E': r[0x10E]})
        off, seg = r[0x104] | r[0x105] << 8, r[0x106] | r[0x107] << 8
    pool = d.mem(D.DS, 0x6752, 28)
    items = []
    iseg, ioff = d.far(0x676E)
    for _ in range(64):
        if (iseg, ioff) == (0, 0):
            break
        it = d.mem(iseg, ioff, 0x3F)
        items.append({'at': '%04X:%04X' % (iseg, ioff), 'plus_32': it[0x32] - 256 if it[0x32] > 127 else it[0x32]})
        ioff, iseg = it[0x2A] | it[0x2B] << 8, it[0x2C] | it[0x2D] << 8
    stop = d.far(0x5CF8)
    out = {'state': 'workplace/dosgolem-cheat/' + name,
           'state_sha256': hashlib.sha256(open(path, 'rb').read()).hexdigest(),
           'where': d.where(), 'party': party,
           'pool_6752': {CURRENCIES[i]: dword(pool, 4 * i) for i in range(7)},
           'items_676E': items, 'stop_5CF8': '%04X:%04X' % stop,
           'unsharing_829B': d.mem(D.DS, 0x829B, 1)[0]}
    d.quit()
    return out


before = snapshot('slums.state')
after = snapshot('handin-slums-stuck.state')
pool = after['pool_6752']
money = (pool['copper'] // 200 + pool['silver'] // 20 + pool['electrum'] // 2 + pool['gold'] +
         pool['platinum'] * 5 + pool['gems'] * 250 + pool['jewelry'] * 2200)
items = 0
stop = after['stop_5CF8']
for it in after['items_676E']:
    if it['at'] == stop:
        break
    if it['plus_32'] > 0:
        product = (it['plus_32'] * 400) & 0xFFFF
        items += product - 0x10000 if product & 0x8000 else product
total = money + items
sharers = len(after['party']) - after['unsharing_829B']
share = total // sharers
deltas = [a['experience_0AC'] - b['experience_0AC'] for a, b in zip(after['party'], before['party'])]
print(json.dumps({'schema': 'pool-dosgolem-loot-experience/1', 'generator': 'dosgolem',
                  'generator_revision': open('/work/dosgolem-revision.txt').read().strip(),
                  'note': '交件前後的經驗值差對 overlay-05 entry 2 `0224h..0306h` 的公款折算（spec 148）',
                  'before': before, 'after': after,
                  'formula': {'money_value': money, 'item_value': items, 'total': total,
                              'sharers': sharers, 'share': share},
                  'experience_delta': deltas}, ensure_ascii=False, indent=1))
