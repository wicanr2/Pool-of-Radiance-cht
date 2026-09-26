#!/usr/bin/env python3
"""原版戰後結算頁（overlay-05 `08E0h`）與之後的戰利品選單（`0E85h`）的基準（#95／#103，spec 150）。

沿用 cheat 通關的駕駛（tools/dosgolem-cheat-playthrough.py）跑兩段：市政廳交貧民窟的件
（`slums.state` → 職員的 `TREASURE → COMBAT`）與波多（`handin-slums.state` → 兩場打哥布林、
一次交件）。駕駛每看到一張畫面就交給 `generic`；這裡把含 `HAS WON`／`FOUND TREASURE`／
`HAS FLED`／`EXPERIENCE`／`HIDES HIS SHARE` 的那一張與緊接著的戰利品選單拍下來，
連同當下的 DS:829Ah（決鬥）、829Bh（沒資格的人數）、829Ch（每份）、439Ch（打過怪）、
439Dh（逃走）。

只讀：原版目錄、dosgolem、tools、docs/audit 唯讀掛載；狀態檔與暫存層用一份複本（第一個參數），
不動主 repo 那一份。

用法：python3 tools/dosgolem-postcombat-screens.py <dosgolem-cheat 複本目錄> > docs/audit/dosgolem-postcombat-screens.json
複本目錄要有 geo.json、font.json、dosgolem-revision.txt、slums.state、handin-slums.state 與 scratch/。
（主機端會自己起容器；JSON 印在標準輸出，PNG 寫在複本目錄的 cap/）"""
import hashlib, importlib.util, json, os, subprocess, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
RUNS = [('handin-slums', 'slums.state'), ('podol', 'handin-slums.state')]
KEYS = ('EXPERIENCE', 'HAS WON', 'FOUND TREASURE', 'HAS FLED', 'HIDES HIS SHARE')

if not os.path.exists('/bin/shots'):
    main = os.environ.get('POOL_MAIN_REPO', ROOT)
    work = os.path.abspath(sys.argv[1])
    for path in [os.path.join(work, name) for name in ('geo.json', 'font.json', 'dosgolem-revision.txt', 'scratch')] + \
            [os.path.join(work, state) for _, state in RUNS] + \
            [os.path.join(main, 'workplace', 'oracle', 'dos'),
             os.path.join(main, 'workplace', 'dosgolem', 'workplace', 'bin', 'shots')]:
        if not os.path.exists(path):
            sys.exit('missing %s' % path)
    os.makedirs(os.path.join(work, 'cap'), exist_ok=True)
    sys.exit(subprocess.call(
        ['timeout', '3600', 'docker', 'run', '--rm', '-i', '--name', 'pool95-postcombat', '--network', 'none',
         '--memory', '3g', '--cpus', '3', '--pids-limit', '256', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
         '-u', '%d:%d' % (os.getuid(), os.getgid()),
         '-v', os.path.join(main, 'workplace', 'dosgolem', 'workplace', 'bin', 'shots') + ':/bin/shots:ro',
         '-v', os.path.join(main, 'workplace', 'oracle', 'dos') + ':/orig:ro',
         '-v', os.path.join(ROOT, 'tools') + ':/tools:ro',
         '-v', os.path.join(ROOT, 'docs', 'audit') + ':/audit:ro',
         '-v', work + ':/work', '-v', os.path.join(work, 'scratch') + ':/scratch',
         '-e', 'PYTHONUNBUFFERED=1', '-e', 'PYTHONPATH=/tools', '-w', '/work',
         'python:3.13-alpine', 'python3', '/tools/dosgolem-postcombat-screens.py']))

sys.path.insert(0, '/tools')
spec = importlib.util.spec_from_file_location('cheat', '/tools/dosgolem-cheat-playthrough.py')
cheat = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cheat)
import dos_text
import dosgolem_drive as D

shots = []
armed = {'menu': False}
original_generic = cheat.Driver.generic


def record(dr, s, why):
    name = '%s-%02d' % (dr.segment, len([x for x in shots if x['segment'] == dr.segment]))
    dos_text.write_png('/work/cap/%s.png' % name, s.idx)
    ds = lambda address, size=1: int.from_bytes(dr.d.mem(D.DS, address, size), 'little')
    shots.append({'name': name, 'why': why, 'segment': dr.segment, 'where': dr.where(),
                  'rows': s.rows, 'bottom': s.bottom(), 'sha256': hashlib.sha256(s.idx).hexdigest(),
                  'ds_829A_duel': ds(0x829A), 'ds_829B_ineligible': ds(0x829B), 'ds_829C_share': ds(0x829C, 4),
                  'ds_439C_fought': ds(0x439C), 'ds_439D_fled': ds(0x439D)})
    print('CAPTURE %s %s' % (name, why), file=sys.stderr)


def generic(self, s):
    text = ' '.join(s.rows)
    if any(key in text for key in KEYS):
        record(self, s, 'result')
        armed['menu'] = True
    elif armed['menu'] and 'POOL' in s.bottom() and 'EXIT' in s.bottom():
        record(self, s, 'menu-after-result')
        armed['menu'] = False
    return original_generic(self, s)


cheat.Driver.generic = generic


def main():
    # 駕駛的進度訊息印在標準輸出；收據要乾淨的 JSON，所以開跑期間把它們導到標準錯誤。
    out, sys.stdout = sys.stdout, sys.stderr
    runs = []
    for segment, state in RUNS:
        load = os.path.join('/work', state)
        dr = cheat.Driver(segment, load)
        try:
            cheat.SEGMENTS[segment](dr)
            status = 'done'
        except Exception as error:  # 拍到就好；段落卡在後面不影響前面拍到的畫面
            status = 'stopped: %r' % error
        runs.append({'segment': segment, 'load': 'workplace/dosgolem-cheat/' + state,
                     'load_sha256': hashlib.sha256(open(load, 'rb').read()).hexdigest(), 'status': status})
        dr.d.quit()
    json.dump({'schema': 'pool-dos-postcombat-screens/1', 'generator': 'dosgolem',
               'generator_revision': open('/work/dosgolem-revision.txt').read().strip(),
               'driver': 'tools/dosgolem-cheat-playthrough.py（generic 前面加一層拍照）',
               'note': 'overlay-05 `08E0h` 結算頁與 `0E85h` 選單；位址見 spec 150。',
               'runs': runs, 'shots': shots}, out, ensure_ascii=False, indent=1)
    print(file=out)


if __name__ == '__main__':
    main()
