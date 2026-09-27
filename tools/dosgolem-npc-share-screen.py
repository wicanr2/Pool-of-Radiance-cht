#!/usr/bin/env python3
"""原版 NPC 分錢頁（overlay-05 `1295h`，`1387h..146Ah`）的版面收據（#111，spec 150）。

cheat 通關的隊伍裡沒有 NPC，所以這一份是**診斷樣本**：從 `slums.state`（市政廳交件之前）
出發，把隊伍鏈上第二與第四個人的記錄改成 `1295h` 認得的形狀——`+84h = B2h`（位元 7 立著）、
`+85h = 1`（有份額）——之後照 cheat 通關的駕駛走到職員的 `TREASURE → COMBAT`。只量
`198h:0039h` 把名字那一行放在第幾欄第幾列，不拿它證明「有 NPC 的隊伍走得到這裡」：
那一條是正常玩家路徑的事，另外量。

`1295h` 挑人的條件是 `+84h > 7Fh`、`+10Ch == 0`、`+85h > 0`（spec 148），印的是
`198h:0039h(5, 5 + [bp-8], 22h, 16h, 0Ah, 1, 名字 + " takes and hides his share.")`、
`[bp-8]` 每列加 2——兩個人應該落在第 5 欄的第 5 與第 7 列。

只讀：原版目錄、dosgolem、tools 唯讀掛載；狀態檔與暫存層用一份複本（第一個參數）。

用法：python3 tools/dosgolem-npc-share-screen.py <dosgolem-cheat 複本目錄> > docs/audit/dosgolem-npc-share-screen.json
複本目錄要有 geo.json、font.json、dosgolem-revision.txt、slums.state 與 scratch/。"""
import hashlib, importlib.util, json, os, subprocess, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LOAD = 'slums.state'
SEGMENT = 'handin-slums'
NPC_SLOTS = (1, 3)  # 隊伍鏈上的第二與第四個人（0 起算）

if not os.path.exists('/bin/shots'):
    main = os.environ.get('POOL_MAIN_REPO', ROOT)
    work = os.path.abspath(sys.argv[1])
    for path in [os.path.join(work, name) for name in ('geo.json', 'font.json', 'dosgolem-revision.txt', 'scratch', LOAD)] + \
            [os.path.join(main, 'workplace', 'oracle', 'dos'),
             os.path.join(main, 'workplace', 'dosgolem', 'workplace', 'bin', 'shots')]:
        if not os.path.exists(path):
            sys.exit('missing %s' % path)
    os.makedirs(os.path.join(work, 'cap'), exist_ok=True)
    sys.exit(subprocess.call(
        ['timeout', '3600', 'docker', 'run', '--rm', '-i', '--name', 'pool111-npc-share', '--network', 'none',
         '--memory', '3g', '--cpus', '3', '--pids-limit', '256', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
         '-u', '%d:%d' % (os.getuid(), os.getgid()),
         '-v', os.path.join(main, 'workplace', 'dosgolem', 'workplace', 'bin', 'shots') + ':/bin/shots:ro',
         '-v', os.path.join(main, 'workplace', 'oracle', 'dos') + ':/orig:ro',
         '-v', os.path.join(ROOT, 'tools') + ':/tools:ro',
         '-v', work + ':/work', '-v', os.path.join(work, 'scratch') + ':/scratch',
         '-e', 'PYTHONUNBUFFERED=1', '-e', 'PYTHONPATH=/tools', '-w', '/work',
         'python:3.13-alpine', 'python3', '/tools/dosgolem-npc-share-screen.py']))

sys.path.insert(0, '/tools')
spec = importlib.util.spec_from_file_location('cheat', '/tools/dosgolem-cheat-playthrough.py')
cheat = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cheat)
import dos_text
import dosgolem_drive as D

shots = []
original_generic = cheat.Driver.generic


class Done(Exception):
    pass


def party_records(d):
    """沿 DS:5CF4h 走隊伍鏈（下一個在 `+104h`），回 (seg, off) 串列。"""
    records = []
    seg, off = d.far(0x5CF4)
    while (seg or off) and len(records) < 8:
        records.append((seg, off))
        b = d.mem(seg, (off + 0x104) & 0xFFFF, 4)
        seg, off = b[2] | b[3] << 8, b[0] | b[1] << 8
    return records


def name_of(d, seg, off):
    raw = d.mem(seg, off, 16)
    return raw[1:1 + raw[0]].decode('ascii', 'replace')


def record(dr, s, why):
    name = 'npc-share-%02d' % len(shots)
    dos_text.write_png('/work/cap/%s.png' % name, s.idx)
    placed = []
    for row, text in enumerate(s.rows):
        if 'HIDES HIS SHARE' in text:
            inner = text[1:]  # 第 0 欄是外框
            placed.append({'row': row, 'column': 1 + len(inner) - len(inner.lstrip(' ')),
                           'text': inner.strip(' ?')})
    shots.append({'name': name, 'why': why, 'where': dr.where(), 'rows': s.rows, 'bottom': s.bottom(),
                  'sha256': hashlib.sha256(s.idx).hexdigest(), 'hides_lines': placed})
    print('CAPTURE %s %s' % (name, why), file=sys.stderr)


def generic(self, s):
    text = ' '.join(s.rows)
    if 'HIDES HIS SHARE' in text:
        record(self, s, 'npc-share')
    elif any(key in text for key in ('HAS WON', 'FOUND TREASURE')) and shots:
        record(self, s, 'result-after-share')
        raise Done()
    return original_generic(self, s)


cheat.Driver.generic = generic


def main():
    out, sys.stdout = sys.stdout, sys.stderr
    load = os.path.join('/work', LOAD)
    dr = cheat.Driver(SEGMENT, load)
    records = party_records(dr.d)
    injected = []
    for slot in NPC_SLOTS:
        seg, off = records[slot]
        before = dr.d.mem(seg, (off + 0x84) & 0xFFFF, 2)
        dr.d.poke(seg, (off + 0x84) & 0xFFFF, [0xB2, 0x01])
        injected.append({'slot': slot, 'name': name_of(dr.d, seg, off),
                         '84_85_before': before.hex(), '84_85_after': 'b201'})
    try:
        cheat.SEGMENTS[SEGMENT](dr)
        status = 'done'
    except Done:
        status = 'captured'
    except Exception as error:
        status = 'stopped: %r' % error
    dr.d.quit()
    json.dump({'schema': 'pool-dos-npc-share-screen/1', 'generator': 'dosgolem',
               'generator_revision': open('/work/dosgolem-revision.txt').read().strip(),
               'driver': 'tools/dosgolem-cheat-playthrough.py（generic 前面加一層拍照）',
               'load': 'workplace/dosgolem-cheat/' + LOAD,
               'load_sha256': hashlib.sha256(open(load, 'rb').read()).hexdigest(),
               'diagnostic_injection': injected,
               'note': '診斷樣本：+84h/+85h 是注入的，只量 overlay-05 `13E6h` 的欄與列（spec 150）。',
               'status': status, 'shots': shots}, out, ensure_ascii=False, indent=1)
    print(file=out)


if __name__ == '__main__':
    main()
