#!/usr/bin/env python3
"""兩邊作弊通關的逐段對照（#5）：讀 remake 與原版（dosgolem）的 checkpoint，寫
docs/audit/cheat-playthrough-compare.md。

對照的只有 goal 第 8 步的三項：每段的必經區塊（依第一次出現的順序）、段末的委任槽與主線旗標、
結局旗標。隨機遭遇、回合數、時刻不比。

段落以旗標切（兩邊同一把尺）：
  slums         4ABB == FE      貧民窟 25 場
  handin-slums  4AC1 >= 1       交貧民窟的件
  podol         4AC1 >= 2       波多廣場拍賣並交件
  norris        4AC1 >= 3       諾里斯並交件
  sokal         4AC1 >= 4       索寇要塞並交件
  ending        4ABA == FE      東航線 → 城堡 → 覲見廳

用法：python3 tools/cheat-playthrough-compare.py（在容器裡跑；主機端會自己起容器）"""
import json, os, subprocess, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if not os.environ.get('COMPARE_IN_CONTAINER'):
    sys.exit(subprocess.call(['timeout', '300', 'docker', 'run', '--rm', '--network', 'none', '--memory', '512m',
                              '--cpus', '1', '--pids-limit', '64', '--log-opt', 'max-size=10m', '--log-opt', 'max-file=3',
                              '-u', '%d:%d' % (os.getuid(), os.getgid()), '-v', ROOT + ':/repo', '-w', '/repo',
                              '-e', 'COMPARE_IN_CONTAINER=1', 'python:3.13-alpine', 'python3', 'tools/cheat-playthrough-compare.py']))
ROOT = '/repo'

SEGMENTS = [
    ('slums', '貧民窟 25 場', lambda f: f['4ABB'] in ('FE', 'FF'), ['ECL2/20']),
    ('handin-slums', '交貧民窟的件', lambda f: int(f['4AC1'], 16) >= 1, ['ECL3/8']),
    ('podol', '波多廣場拍賣並交件', lambda f: int(f['4AC1'], 16) >= 2, ['ECL8/29', 'ECL1/18', 'ECL3/8']),
    ('norris', '諾里斯並交件', lambda f: int(f['4AC1'], 16) >= 3, ['ECL8/29', 'ECL3/8']),
    ('sokal', '索寇要塞並交件', lambda f: int(f['4AC1'], 16) >= 4, ['ECL4/21', 'ECL3/8']),
    ('ending', '東航線 → 城堡 → 覲見廳', lambda f: f['4ABA'] == 'FE',
     ['ECL8/27', 'ECL7/26', 'ECL1/18', 'ECL2/9', 'ECL5/3', 'ECL5/5', 'ECL5/7']),
]
# 主線旗標（spec 137 的段落出口與委任槽）與路線旗標分開報：4A77／4A78 是斯托亞諾夫城門的經過紀錄，
# 隨走哪幾格而變（原版駕駛穿牆經過衛兵格），不是主線的閘門。
MAIN = ['4ABB', '4AC1', '4AB0', '4AA6', '4AA7', '4A24', '4AC4', '4ABA']
ROUTE = ['4A77', '4A78']


def split(checkpoints):
    """依旗標把 checkpoint 切成段；滿足結束條件的那一筆算進這一段，之後的換下一段。"""
    out, index = {name: [] for name, _, _, _ in SEGMENTS}, 0
    for cp in checkpoints:
        if index >= len(SEGMENTS):
            break
        out[SEGMENTS[index][0]].append(cp)
        while index < len(SEGMENTS) and SEGMENTS[index][2](cp['flags']):
            index += 1
    return out


def key(cp):
    return 'ECL%d/%d' % (cp['ecl_archive'], cp['ecl_block'])


def follows(cps, required):
    """required 是不是走訪序列的子序列（中間可以插別的區塊、同一個區塊可以來回）。"""
    it = iter(key(cp) for cp in cps)
    return all(any(k == want for k in it) for want in required)


# 已經開了 issue 的差異：鍵是報表裡出現的片段，值是 issue 編號。沒有列在這裡的差異算「沒開 issue」。
FILED = {
    'ECL4/21 之後 remake 進 ECL2/20／原版進 ECL3/0': '#44',
    '結局頁': '#45',
}

# 過渡區塊：remake 在同一個 tick 裡就 `NEWECL` 換掉，記不到 checkpoint（spec 137 第 8 段：ECL5/6 → 立刻 NEWECL 3）。
TRANSIENT = {'ECL5/6'}


def exits(cps, required):
    """每個必經區塊（最後一次）離開後進的下一個區塊。"""
    keys = [key(cp) for cp in cps if key(cp) not in TRANSIENT]
    out = {}
    for i, k in enumerate(keys[:-1]):
        if k in required and keys[i + 1] != k:
            out[k] = keys[i + 1]
    return out


def extras(cps, required):
    out = []
    for cp in cps:
        k = key(cp)
        if k not in required and k not in ('ECL3/0',) and k not in out:
            out.append(k)
    return out


def main():
    remake = json.load(open(os.path.join(ROOT, 'docs', 'audit', 'remake-cheat-playthrough.json')))
    dos = json.load(open(os.path.join(ROOT, 'docs', 'audit', 'dosgolem-cheat-playthrough.json')))
    r_split = split(remake['checkpoints'])
    lines = ['# 兩邊作弊通關的逐段對照（#5）', '',
             '由 `tools/cheat-playthrough-compare.py` 產生，別手改。來源：',
             '`docs/audit/remake-cheat-playthrough.json`（`%s`）與 `docs/audit/dosgolem-cheat-playthrough.json`'
             '（dosgolem `%s`，駕駛 `%s`）。' % (remake.get('test', '?'), dos.get('generator_revision', '—'), dos.get('driver', '—')), '',
             '對照三項（goal 第 8 步）：必經區塊（spec 137 的段落，比「兩邊都走過」與先後順序）、段末主線旗標、結局。',
             '隨機遭遇、回合數、時刻、路過的其他區塊不列入；多走的區塊寫在備註（城區 ECL3/0 不列）。',
             'remake 的段落以旗標切；原版以駕駛的段落收據為準。', '',
             '| 段 | 內容 | 必經區塊 | 段末主線旗標 | 路線旗標 | 備註 |', '|---|---|---|---|---|---|']
    problems = filed = 0
    for name, label, _, required in SEGMENTS:
        seg = dos['segments'].get(name)
        r = r_split[name]
        if not seg or seg.get('status') != 'done':
            lines.append('| %s | %s | 原版還沒跑到 | — | — | |' % (name, label))
            continue
        rk, dk = [key(cp) for cp in r], [key(cp) for cp in seg['checkpoints']]
        missing_r = [k for k in required if k not in rk]
        missing_d = [k for k in required if k not in dk]
        if missing_r or missing_d:
            block = '不一致：remake 缺 %s／原版缺 %s' % ('、'.join(missing_r) or '無', '、'.join(missing_d) or '無')
        elif not follows(r, required) or not follows(seg['checkpoints'], required):
            block = '順序不同：remake %s／原版 %s' % (' → '.join(rk), ' → '.join(dk))
        else:
            block = '一致（%s）' % ' → '.join(required)
        re_, de = exits(r, required), exits(seg['checkpoints'], required)
        exit_diff = ['%s 之後 remake 進 %s／原版進 %s' % (k, re_[k], de[k]) for k in required
                     if k in re_ and k in de and re_[k] != de[k]]
        if exit_diff:
            block += '；出口不同：' + '、'.join(d + ('（%s）' % FILED[d] if d in FILED else '') for d in exit_diff)
        rf, df = r[-1]['flags'], seg['flags']
        diff = [k for k in MAIN if rf.get(k) != df.get(k)]
        flags = '一致' if not diff else '不一致：' + '、'.join('%s remake %s／原版 %s' % (k, rf.get(k), df.get(k)) for k in diff)
        rdiff = [k for k in ROUTE if rf.get(k) != df.get(k)]
        route = '相同' if not rdiff else '、'.join('%s remake %s／原版 %s' % (k, rf.get(k), df.get(k)) for k in rdiff)
        if not block.startswith('一致') or diff or [d for d in exit_diff if d not in FILED]:
            problems += 1
        filed += len([d for d in exit_diff if d in FILED])
        notes = []
        er, ed = extras(r, required), extras(seg['checkpoints'], required)
        if er:
            notes.append('remake 另走 ' + '、'.join(er))
        if ed:
            notes.append('原版另走 ' + '、'.join(ed))
        if seg.get('wall_clears'):
            notes.append('原版穿牆 %d 次' % seg['wall_clears'])
        lines.append('| %s | %s | %s | %s | %s | %s |' % (name, label, block, flags, route, '；'.join(notes)))
    ending = dos['segments'].get('ending', {})
    pages = ending.get('ending_pages', [])
    lines += ['', '## 結局', '',
              '| 項 | remake | 原版 |', '|---|---|---|',
              '| `4ABA` | %s | %s |' % (remake['checkpoints'][-1]['flags']['4ABA'], ending.get('flags', {}).get('4ABA', '—')),
              '| 結局後回到 | %s | ECL%s/%s |' % (key(remake['checkpoints'][-1]), ending.get('end', {}).get('archive', '—'), ending.get('end', {}).get('block', '—')),
              '| 結局頁 | 3 頁（spec 137 第 11 段） | 駕駛攔到 %d 頁：%s |' % (
                  len(pages), '／'.join('「%s」' % p.split(' | ')[-2] if ' | ' in p else p for p in pages) or '—'),
              '', '結局頁數不同（%s）。' % FILED['結局頁'] if len(pages) != 3 else '結局頁數相同。',
              '', '沒開 issue 的主線不一致：%d 段（共 %d 段）；已開 issue 的差異：%d 項。' % (
                  problems, len(SEGMENTS), filed + (1 if len(pages) != 3 else 0)), '']
    open(os.path.join(ROOT, 'docs', 'audit', 'cheat-playthrough-compare.md'), 'w').write('\n'.join(lines))
    print('\n'.join(lines))


if __name__ == '__main__':
    main()
