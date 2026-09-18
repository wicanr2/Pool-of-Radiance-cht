#!/usr/bin/env python3
"""原版戰利品畫面與分錢的基準（#47）。

從市政廳交件拿到獎金那一刻（戰利品頂層選單）出發，抓四張畫面並量分錢前後的錢包：

  頂層 → TAKE（幣種清單）→ 回頂層 → VIEW → 回頂層 → SHARE → 分完的頂層

位址（spec 040）：全隊池 `DS:6752h + 4*i`（七個 uint32）、角色錢包 record `+88h + 2*i`（七個 uint16）、
現行負重 `+102h`、隊伍鏈 `DS:5CF4h`（far pointer）、下一個 `+104h`、`+84h` 是 `00h`／`B3h` 才算 active。

輸出 docs/audit/dos-treasure-screens.json 與 workplace/dosgolem-cheat/treasure/*.png。
用法：python3 tools/dosgolem-treasure-screens.py（主機端會自己起容器）"""
import hashlib, json, os, subprocess, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if not os.path.exists('/bin/shots'):
    work = os.path.join(ROOT, 'workplace', 'dosgolem-cheat')
    sys.exit(subprocess.call(
        ['timeout', '1800', 'docker', 'run', '--rm', '-i', '--network', 'none', '--memory', '4g', '--cpus', '2',
         '--pids-limit', '256', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
         '-u', '%d:%d' % (os.getuid(), os.getgid()),
         '-v', os.path.join(ROOT, 'workplace', 'dosgolem', 'workplace', 'bin', 'shots') + ':/bin/shots:ro',
         '-v', os.path.join(ROOT, 'workplace', 'oracle', 'dos') + ':/orig:ro',
         '-v', os.path.join(ROOT, 'tools') + ':/tools:ro',
         '-v', work + ':/work', '-v', os.path.join(work, 'scratch') + ':/scratch',
         '-e', 'PYTHONUNBUFFERED=1', '-e', 'PYTHONPATH=/tools', '-w', '/work',
         'python:3.13-alpine', 'python3', '/tools/dosgolem-treasure-screens.py'] + sys.argv[1:]))

sys.path.insert(0, '/tools')
import dos_text
import dosgolem_drive as D

WORK = '/work'
OUT = os.path.join(WORK, 'treasure')
STATE = os.path.join(WORK, 'handin-slums-stuck.state')
CURRENCIES = ['copper', 'silver', 'electrum', 'gold', 'platinum', 'gems', 'jewelry']


def party(d):
    """沿 DS:5CF4h 的鏈讀每個角色：是否 active（+84h）、七欄錢包（+88h）、現行負重（+102h）、名字（record 開頭）。"""
    seg, off = d.far(0x5CF4)
    out = []
    for index in range(8):
        if seg == 0 and off == 0:
            break
        record = d.mem(seg, off, 0x108)
        name = record[:15].split(b'\x00')[0].decode('ascii', 'replace').strip()
        out.append({'record': '%04X:%04X' % (seg, off), 'name': name, 'active_84': record[0x84],
                    'wallet': {CURRENCIES[i]: record[0x88 + 2 * i] | record[0x89 + 2 * i] << 8 for i in range(7)},
                    'load_102': record[0x102] | record[0x103] << 8})
        # `+104h` 是 far pointer（offset 在前、segment 在後）：只讀兩個位元組會在第二個人就讀到別的地方。
        nxt_off = record[0x104] | record[0x105] << 8
        nxt_seg = record[0x106] | record[0x107] << 8
        if (nxt_seg, nxt_off) in ((0, 0), (0xFFFF, 0xFFFF)):
            break
        seg, off = nxt_seg, nxt_off
    return out


def pool(d):
    raw = d.mem(D.DS, 0x6752, 28)
    return {CURRENCIES[i]: int.from_bytes(raw[4 * i:4 * i + 4], 'little') for i in range(7)}


def shot(d, name, keys):
    for key in keys:
        d.key(key)
    idx = d.frame()
    open(os.path.join(OUT, name + '.idx'), 'wb').write(idx)
    dos_text.write_png(os.path.join(OUT, name + '.png'), idx)
    screen = D.Screen(idx, d.font)
    rows = [r for r in screen.rows if r.strip()]
    print('==', name, 'keys', keys)
    print('\n'.join(rows))
    return {'name': name, 'keys': keys, 'rows': rows, 'bottom': screen.bottom(),
            'options': [[w, h] for w, h, _ in screen.options()],
            'sha256': hashlib.sha256(idx).hexdigest()}


def main():
    os.makedirs(OUT, exist_ok=True)
    d = D.Dos(load=STATE, intercept=False)
    screens = [shot(d, 'top', [])]
    before = {'party': party(d), 'pool': pool(d)}
    screens.append(shot(d, 'take', ['t']))
    screens.append(shot(d, 'top-after-take-exit', ['e']))
    screens.append(shot(d, 'view', ['v']))
    screens.append(shot(d, 'top-after-view-exit', ['e']))
    screens.append(shot(d, 'after-share', ['s']))
    after = {'party': party(d), 'pool': pool(d)}
    out = {'schema': 'pool-dos-treasure-screens/1', 'generator': 'dosgolem',
           'generator_revision': open(os.path.join(WORK, 'dosgolem-revision.txt')).read().strip(),
           'state': 'workplace/dosgolem-cheat/handin-slums-stuck.state',
           'state_sha256': hashlib.sha256(open(STATE, 'rb').read()).hexdigest(),
           'note': '市政廳交完貧民窟的件、職員給獎金的那一刻（戰利品頂層選單）。位址見 spec 040。',
           'screens': screens, 'before': before, 'after': after}
    json.dump(out, open(os.path.join(WORK, 'dos-treasure-screens.json'), 'w'), ensure_ascii=False, indent=1)
    print('\npool  ', before['pool'], '→', after['pool'])
    for b, a in zip(before['party'], after['party']):
        print('%-8s active=%02X load %d→%d %s → %s' % (b['name'], b['active_84'], b['load_102'], a['load_102'],
                                                       {k: v for k, v in b['wallet'].items() if v},
                                                       {k: v for k, v in a['wallet'].items() if v}))
    d.quit()


if __name__ == '__main__':
    main()
