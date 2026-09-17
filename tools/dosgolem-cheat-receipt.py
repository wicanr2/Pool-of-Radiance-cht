#!/usr/bin/env python3
"""把原版作弊通關（tools/dosgolem-cheat-playthrough.py）各段的執行整合成
docs/audit/dosgolem-cheat-playthrough.json。

一段常常要接續好幾次：前一次卡住存下 <段>-stuck.state，修好駕駛之後拷成新的名字接著跑。
CHAIN 列出實際串起來的那幾次（以日誌的開始時刻認），其餘是除錯時丟掉的嘗試。
checkpoint 從段落日誌讀（每一次執行都有），鍵序、狀態檔 SHA-256、穿牆紀錄與攔截統計從
workplace/dosgolem-cheat/receipt.json 讀（收據是後來才加的，最早幾次沒有）。

用法：python3 tools/dosgolem-cheat-receipt.py（主機端會自己起容器；輸出寫到 workplace，再由主機拷進 docs/audit）"""
import ast, hashlib, json, os, re, subprocess, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if not os.environ.get('RECEIPT_IN_CONTAINER'):
    code = subprocess.call(['timeout', '300', 'docker', 'run', '--rm', '--network', 'none', '--memory', '512m', '--cpus', '1',
                            '--pids-limit', '64', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
                            '-u', '%d:%d' % (os.getuid(), os.getgid()), '-v', ROOT + ':/repo', '-w', '/repo',
                            '-e', 'RECEIPT_IN_CONTAINER=1', 'python:3.13-alpine', 'python3', 'tools/dosgolem-cheat-receipt.py'])
    sys.exit(code)
ROOT = '/repo'
WORK = os.path.join(ROOT, 'workplace', 'dosgolem-cheat')

# 段落 → 串起來的那幾次執行的開始時刻（段落日誌的 `== <段> from` 那一行，UTC）。
CHAIN = {
    'slums': ['12:06:44', '12:27:29', '12:29:59', '12:38:18', '12:39:00', '12:54:38'],
    'handin-slums': ['13:48:18'],
    'podol': ['13:50:01'],
    'norris': ['13:53:28'],
    'sokal': ['13:57:05'],
    'ending': ['14:03:49', '14:04:10', '14:04:49', '15:01:04', '15:04:45', '15:05:43', '15:06:31', '15:06:52', '15:14:40'],
}
LINE = re.compile(r'^(\d\d:\d\d:\d\d) (.*)$')


def log_runs(segment):
    runs, cur = [], None
    for raw in open(os.path.join(WORK, segment + '.log'), encoding='utf-8'):
        m = LINE.match(raw.rstrip('\n'))
        if not m:
            continue
        t, body = m.groups()
        if body.startswith('== %s from ' % segment):
            cur = {'start': t, 'load': body.split(' from ', 1)[1], 'checkpoints': [], 'status': None, 'seconds': None}
            runs.append(cur)
        elif cur is not None and body.startswith('checkpoint '):
            mm = re.match(r'checkpoint ECL(\d+)/(\d+) \((\d+),(\d+)\) (\{.*\})$', body)
            a, b, x, y, flags = mm.groups()
            cur['checkpoints'].append({'ecl_archive': int(a), 'ecl_block': int(b), 'x': int(x), 'y': int(y),
                                       'flags': ast.literal_eval(flags)})
        elif cur is not None and body.startswith(segment + ': '):
            mm = re.match(r'.*: (done|stuck: .*) in (\d+)s, (\d+) actions$', body)
            if mm:
                cur['status'], cur['seconds'] = mm.group(1), int(mm.group(2))
    return runs


def main():
    receipt = json.load(open(os.path.join(WORK, 'receipt.json')))
    goblins = json.load(open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-deployment-peek-goblins.json')))
    cases = json.load(open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-block-load-clear-cases.json')))
    out = {
        'schema': 'pool-dosgolem-cheat-playthrough/1',
        'generator': 'dosgolem',
        'generator_revision': open(os.path.join(WORK, 'dosgolem-revision.txt')).read().strip(),
        'driver': 'tools/dosgolem-cheat-playthrough.py',
        'original_exe_sha256': goblins['original_exe_sha256'],
        'cheats': {
            'damage_hook': '-intercept-damage stub=010A:00AC,hp=11B,side=10E,combat=4954:5,party=0,lock,kill（spec 084）',
            'walk_through_walls': '被擋住的那一步清掉 DS:69BAh 指到的 GEO 牆 nibble、踏過去再寫回（spec 141〈穿牆〉，使用者 2026-09-17）',
        },
        'boot': {
            'note': '標題 → 建六人隊伍 → 開場導覽 → 城區往西進貧民窟；狀態檔沿用 #41 的 workplace/dosgolem-4a01/slums.state',
            'keys': cases['walk_out_keys'],
            'state_sha256': hashlib.sha256(open(os.path.join(ROOT, 'workplace', 'dosgolem-4a01', 'slums.state'), 'rb').read()).hexdigest(),
        },
        'segments': {},
    }
    for segment, starts in CHAIN.items():
        runs = log_runs(segment)
        spent = [r for r in receipt.get(segment, {}).get('runs', [])]
        chain = []
        for start in starts:
            run = next(r for r in runs if r['start'] == start)
            match = next((i for i, r in enumerate(spent) if r['load'] == run['load'] and r['status'] == run['status']
                          and abs(r['seconds'] - (run['seconds'] or -99)) <= 1), None)
            entry = {'start_utc': start, 'load': run['load'], 'status': run['status'], 'seconds': run['seconds'],
                     'checkpoints': run['checkpoints']}
            if match is not None:
                r = spent.pop(match)
                entry.update({'keys': r['keys'], 'end_state_sha256': r['state_sha256'], 'wall_clears': r.get('wall_clears', []),
                              'damage_intercepts': r['damage_intercepts'],
                              'party_damage_nonzero_after_hook': r['party_damage_nonzero_after_hook'],
                              'ending_pages': r.get('ending_pages', []),
                              'end_flags': r['flags'], 'end_where': r['end']})
            else:
                entry['receipt'] = '這一次執行早於收據檔，只有日誌的 checkpoint'
            chain.append(entry)
        merged = []
        for entry in chain:
            for cp in entry['checkpoints']:
                if merged and (merged[-1]['ecl_archive'], merged[-1]['ecl_block']) == (cp['ecl_archive'], cp['ecl_block']):
                    continue
                merged.append(cp)
        last = chain[-1]
        out['segments'][segment] = {
            'status': last['status'],
            'runs': chain,
            'checkpoints': merged,
            # 段末以最後一次執行結束時讀的為準：同一個區塊裡寫的旗標（貧民窟攤位那一場的 FE）不會有新的 checkpoint。
            'end': last.get('end_where', {'archive': merged[-1]['ecl_archive'], 'block': merged[-1]['ecl_block']}),
            'flags': last.get('end_flags', merged[-1]['flags']),
            'wall_clears': sum(len(e.get('wall_clears', [])) for e in chain),
            'damage_intercepts': sum(e.get('damage_intercepts', 0) for e in chain),
            'party_damage_nonzero_after_hook': sum(e.get('party_damage_nonzero_after_hook', 0) for e in chain),
            'ending_pages': last.get('ending_pages', []),
        }
    path = os.path.join(WORK, 'dosgolem-cheat-playthrough.json')
    json.dump(out, open(path, 'w'), ensure_ascii=False, indent=1)
    for name, seg in out['segments'].items():
        print(name, seg['status'], 'runs', len(seg['runs']), 'blocks', len(seg['checkpoints']), seg['flags'],
              'walls', seg['wall_clears'], 'hooks', seg['damage_intercepts'], 'party!=0', seg['party_damage_nonzero_after_hook'])


if __name__ == '__main__':
    main()
